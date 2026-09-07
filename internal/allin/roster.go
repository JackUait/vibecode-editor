package allin

import (
	"fmt"
	"os"
	"strings"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// minRosterContext is the narrowest window worth offering. Claude Code's own
// floor is ~20k tokens before a conversation starts, and a profile reserves a
// quarter of the window for the reply, so anything tighter cannot finish a task.
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
type Row struct {
	Model       string `json:"model"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	BehavesAs   string `json:"behavesAs,omitempty"`
}

// claudeModel is one first-party model offered for every Claude login.
type claudeModel struct {
	id      string
	label   string
	wide    bool // also offer a [1m] row
}

// claudeLineup is pinned rather than discovered: Claude Code ships no catalog a
// third party can read, and an id it does not know still routes fine.
var claudeLineup = []claudeModel{
	{"claude-opus-5", "Opus 5", true},
	{"claude-sonnet-5", "Sonnet 5", true},
	{"claude-fable-5-1", "Fable 5.1", false},
	{"claude-haiku-4-5-20251001", "Haiku 4.5", false},
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
			if model.wide {
				rows = append(rows, Row{
					Model:       fmt.Sprintf("wisp/acct.%s/%s[1m]", account.dir, model.id),
					Label:       account.label + " · " + model.label + " (1M)",
					Description: "Claude subscription: " + account.label,
				})
			}
		}
	}
	return rows
}

func configRows(env Env) []Row {
	var rows []Row
	for _, config := range claudeconfig.Load(env.ConfigsList) {
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
		source := strings.TrimSuffix(config.File, ".json")
		for _, model := range providerModels(env, config, provider) {
			id := model.ID
			suffix := ""
			if model.Context >= 1000000 {
				suffix = "[1m]"
			} else if model.Context != 0 && model.Context < minRosterContext {
				continue
			}
			rows = append(rows, Row{
				Model:       fmt.Sprintf("wisp/cfg.%s/%s%s", source, id, suffix),
				Label:       config.Name + " · " + id,
				Description: provider.Name,
			})
		}
	}
	return rows
}

// providerModels returns the catalog's models, or the single model the user
// supplied for a provider that ships none.
func providerModels(env Env, config claudeconfig.Config, provider claudeconfig.Provider) []claudeconfig.Model {
	if provider.SuppliesOwnModel() {
		id := claudeconfig.ReadCustomModel(env.ConfigsDir, config.File)
		if id == "" {
			return nil
		}
		// Window unknown here; the profile already declares it, and a row with
		// no declared context is offered at Claude Code's flat 200000.
		return []claudeconfig.Model{{ID: id}}
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
