package main

import (
	"net/http"
	"time"

	"github.com/jackuait/wisp-deck/internal/allin"
	"github.com/jackuait/wisp-deck/internal/subusage"
)

// allInUsageMinInterval throttles each source, matching subscription-usage. A
// deck opens tabs in bursts and every All-In launch arms a round, so without it
// the endpoints would see one request per tab.
const allInUsageMinInterval = 300

// usageRefreshOptions is what one round needs beyond the files on disk.
// anthropic and codexAuth are overridden only by tests.
type usageRefreshOptions struct {
	env         allin.Env
	anthropic   string
	codexAuth   string
	minInterval int
}

// allInUsageRefresh is the round one All-In launch arms, as a var so the wiring
// can be observed without a network.
var allInUsageRefresh = refreshAllInUsage

func refreshAllInUsage(env allin.Env) {
	refreshAllInUsageWith(usageRefreshOptions{
		env:         env,
		anthropic:   subusage.AnthropicBaseURL,
		minInterval: allInUsageMinInterval,
	})
}

// refreshAllInUsageWith reads what every roster source has left, then rewrites
// the profile so the picker carries it.
//
// Claude Code snapshots modelPicker ONCE, at launch, so a round can never
// change the picker of the session that armed it — it makes the NEXT tab's
// numbers fresh. That is the same fact that makes an unknown reading refuse to
// hide a row: a wrong hide would stand for a whole session.
func refreshAllInUsageWith(opts usageRefreshOptions) {
	if opts.env.ConfigsList == "" {
		return
	}
	client := &http.Client{Timeout: 15 * time.Second}
	for _, login := range accountUsageLogins(opts.env.AccountsList) {
		refreshAccountUsage(client, opts, login)
	}
	for _, source := range allInConfigSources(opts.env) {
		refreshConfigUsage(client, opts, source)
	}
	_ = allin.EnsureProfileIfEligible(opts.env)
}

// allInConfigSources is every provider profile the roster offers, once each. It
// comes off the roster rather than the configs list so a profile the picker
// already refuses — disabled, unready, not routable — is never fetched for.
func allInConfigSources(env allin.Env) []string {
	var sources []string
	seen := map[string]bool{}
	for _, row := range allin.Roster(env) {
		target := allin.Route(row.Model)
		if target.Kind != allin.KindConfig || target.Source == "" || seen[target.Source] {
			continue
		}
		seen[target.Source] = true
		sources = append(sources, target.Source)
	}
	return sources
}

func refreshConfigUsage(client *http.Client, opts usageRefreshOptions, source string) {
	refreshUsageCache(allin.ConfigUsageFile(opts.env.ConfigsList, source), opts.minInterval,
		func() (subusage.Snapshot, error) {
			snap, _, err := subusage.Fetch(client, opts.env.ConfigsDir,
				opts.env.ConfigsList, source+".json", opts.codexAuth)
			return snap, err
		})
}
