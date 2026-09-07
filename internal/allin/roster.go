package allin

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// minRosterContext is the narrowest window worth offering. Claude Code's own
// floor is ~20k tokens before a conversation starts, and a profile reserves a
// quarter of the window for the reply, so anything tighter cannot finish a task.
//
// It is also the window every row gets. No row carries the "[1m]" marker, even
// where the model has a 1M window: the marker is read off the raw model string
// and grants the whole SESSION 1M, and nothing narrows it again when the user
// picks a 200k row mid-conversation — the transcript is already past the new
// endpoint's cap, and /compact sends the same oversized transcript plus a
// summarization prompt, so it fails the same way. A uniform window is the one
// shape no pick can wedge. Route still strips and reports the marker, and
// proxy.go still sends the beta header, so 1M can return behind a size guard.
const minRosterContext = 200000

// Env names the four files the roster is built from. They are the same files
// the account switcher and the subscription modal already own.
type Env struct {
	AccountsList string
	AccountsDir  string
	ConfigsList  string
	ConfigsDir   string
}

// Row is one entry of the settings key `modelPicker.options`.
//
// There is deliberately no behavesAs. Measured on a live pane: an unknown model
// id without it is given thinking:{"type":"adaptive"} and keeps effort
// available, while behavesAs:claude-sonnet-4-5 turns that into
// thinking:{"type":"enabled",budget} and "Effort not supported". Absent is the
// more capable default, and it carries no context window across the router
// either — so the field bought nothing and cost the pane its effort control.
type Row struct {
	Model       string `json:"model"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// claudeModel is one first-party model offered for every Claude login.
type claudeModel struct {
	id    string
	label string
}

// claudeLineup is pinned rather than discovered: Claude Code ships no catalog a
// third party can read, and an id it does not know still routes fine.
var claudeLineup = []claudeModel{
	{"claude-opus-5", "Opus 5"},
	{"claude-sonnet-5", "Sonnet 5"},
	{"claude-fable-5-1", "Fable 5.1"},
	{"claude-haiku-4-5-20251001", "Haiku 4.5"},
}

// SourceCount reports how many distinct credentials the roster spans: each
// Claude login and each routable provider profile counts once, however many
// models it contributes. It is derived from Roster itself, so the two can never
// disagree about what this build can actually offer.
func SourceCount(env Env) int {
	seen := map[string]bool{}
	for _, row := range Roster(env) {
		target := Route(row.Model)
		if target.Kind == KindSession || target.Source == "" {
			continue
		}
		seen[fmt.Sprintf("%d/%s", target.Kind, target.Source)] = true
	}
	return len(seen)
}

// Roster builds every picker row, accounts first, then configured providers.
// A source that cannot be read contributes nothing rather than failing the set.
func Roster(env Env) []Row {
	rows := accountRows(env)
	return append(rows, configRows(env)...)
}

func accountRows(env Env) []Row {
	accounts := []struct{ label, dir string }{{"Default", "default"}}
	for _, line := range readLines(env.AccountsList) {
		label, dir, ok := strings.Cut(line, ":")
		if !ok || label == "" || dir == "" {
			continue
		}
		accounts = append(accounts, struct{ label, dir string }{label, dir})
	}
	var rows []Row
	for _, account := range accounts {
		for _, model := range claudeLineup {
			rows = append(rows, Row{
				Model:       fmt.Sprintf("wisp/acct.%s/%s", account.dir, model.id),
				Label:       account.label + " · " + model.label,
				Description: "Claude subscription: " + account.label,
			})
		}
	}
	return rows
}

func configRows(env Env) []Row {
	var rows []Row
	for _, config := range claudeconfig.Load(env.ConfigsList) {
		// The generated profile is registered in the very list this iterates,
		// and a row naming it would send a turn back into the router that asked
		// for it. Its own AuthWispRouter identity is skipped by the Auth check
		// below, but that identity is only on disk AFTER EnsureProfile writes —
		// and EnsureProfile computes these rows first, from whatever profile it
		// adopted under this name. So the name is the only guard that holds on
		// the call that matters.
		if strings.EqualFold(config.Name, ProfileName) {
			continue
		}
		if !claudeconfig.ConfigReady(env.ConfigsDir, config) {
			continue
		}
		provider := claudeconfig.ProviderForConfig(env.ConfigsDir, config)
		// v1 routes API-key providers only. A ChatGPT profile authenticates
		// through `codex login` and is served by a bridge process, not by an
		// endpoint with a key, so a row for it would resolve to nothing.
		if provider.Auth != claudeconfig.AuthAPIKey {
			continue
		}
		// Featherless (RemoteCatalog) needs internal/rolefix's request/response
		// repair to call a tool at all: it 400s on Claude Code's role:"system"
		// messages and stops parsing tool calls when `thinking` is present.
		// This router does neither repair, so a Featherless row would 400 or
		// silently stop calling tools on its first turn. UserConfigured (the
		// self-hosted `custom` provider) speaks the Anthropic API directly and
		// needs no repair, so it is not skipped here.
		if provider.RemoteCatalog {
			continue
		}
		source := strings.TrimSuffix(config.File, ".json")
		for _, model := range providerModels(env, config, provider) {
			if model.Context != 0 && model.Context < minRosterContext {
				continue
			}
			rows = append(rows, Row{
				Model:       fmt.Sprintf("wisp/cfg.%s/%s", source, model.ID),
				Label:       config.Name + " · " + model.ID,
				Description: provider.Name,
			})
		}
	}
	return rows
}

// providerModels returns the catalog's models, or the single model the user
// supplied for a provider that ships none. If a user-supplied model's declared
// window does not parse or is too small, the provider is skipped.
func providerModels(env Env, config claudeconfig.Config, provider claudeconfig.Provider) []claudeconfig.Model {
	if provider.SuppliesOwnModel() {
		id := claudeconfig.ReadCustomModel(env.ConfigsDir, config.File)
		if id == "" {
			return nil
		}
		windowStr := claudeconfig.ReadContextWindow(env.ConfigsDir, config.File)
		window, err := strconv.Atoi(strings.TrimSpace(windowStr))
		if err != nil || window <= 0 {
			return nil
		}
		return []claudeconfig.Model{{ID: id, Context: window}}
	}
	return provider.Models
}

func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}
