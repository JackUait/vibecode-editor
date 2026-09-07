package allin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// ProfileName is the display name of the generated profile. It is matched
// verbatim in the configs list, so renaming it orphans the existing one.
const ProfileName = "All-In"

// EnsureProfile creates the All-In profile when absent and rewrites its picker
// rows every call — logins and providers come and go, and a stale roster offers
// models the machine can no longer reach.
//
// Only modelPicker and the three env keys routerEnv names are written; they are
// rewritten every call because each one is load-bearing (see routerEnv) and a
// profile missing any of them fails silently. Every other key in the file is
// the user's, and the launch overlay copies the whole object.
func EnsureProfile(env Env, listFile, configsDir string) (string, error) {
	file := ProfileFile(listFile)
	if file == "" {
		created, err := claudeconfig.Add(listFile, configsDir, ProfileName)
		if err != nil {
			return "", err
		}
		file = created
	}

	path := filepath.Join(configsDir, file)
	settings := map[string]any{}
	// A missing file starts empty; any other read or parse failure must not
	// be treated as "no keys" — that would silently drop the user's own keys
	// (env, permissions, an API key) the next time this writes the file.
	if data, err := os.ReadFile(path); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
	} else if err := json.Unmarshal(data, &settings); err != nil {
		return "", err
	}
	settingsEnv, _ := settings["env"].(map[string]any)
	if settingsEnv == nil {
		settingsEnv = map[string]any{}
	}
	for key, value := range routerEnv() {
		settingsEnv[key] = value
	}
	settings["env"] = settingsEnv

	rows := Roster(env)
	options := make([]Row, 0, len(rows))
	options = append(options, rows...)
	settings["modelPicker"] = map[string]any{
		// Every row names its account explicitly, so the built-in lineup would
		// only add rows whose credential is ambiguous.
		"replaceBuiltInOptions": true,
		"options":               options,
	}

	encoded, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	// Published by rename: a reader must never see half a settings file.
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(encoded, '\n'), 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(temporary, path); err != nil {
		return "", err
	}
	return file, nil
}

// EnsureProfileIfEligible applies the create-vs-refresh gate every mutation
// site shares (the CLI's own add/delete/ensure-allin, and the TUI's
// login/subscription add and delete): create the profile only once the
// machine has two or more sources, but refresh one that already exists no
// matter how the count moves — a login or provider removed today would
// otherwise leave rows that answer 400 for the life of the profile.
//
// A caller missing any of the four Env paths gets an error rather than a
// silent no-op: Roster reads an empty AccountsList/ConfigsList as "nothing
// there", so refreshing from an incomplete Env would silently strip real
// logins or providers out of an existing picker. Every one of today's six
// call sites discards the error (this must never break the mutation the user
// asked for), so an error here is the only signal a structurally broken
// caller leaves behind — a silent nil would make that caller invisible
// forever. No caller legitimately has an empty path — every site builds all
// four from the same config root.
func EnsureProfileIfEligible(env Env) error {
	if env.AccountsList == "" || env.AccountsDir == "" ||
		env.ConfigsList == "" || env.ConfigsDir == "" {
		return fmt.Errorf("allin: incomplete Env: AccountsList, AccountsDir, ConfigsList and ConfigsDir must all be set")
	}
	if SourceCount(env) < 2 && ProfileFile(env.ConfigsList) == "" {
		return nil
	}
	file, err := EnsureProfile(env, env.ConfigsList, env.ConfigsDir)
	if err != nil {
		return err
	}
	// A profile born here has no other creation path to reach it: it is
	// never copied from a default (there is no default) and never swept by
	// bin/wisp-deck's own ensure-watchdog, which only sees a file already on
	// disk. Never move this into routerEnv — that block rewrites its keys on
	// every call, while a declared watchdog value is documented as the
	// user's own and every other path keeps it untouched.
	_, err = claudeconfig.EnsureStreamWatchdog(env.ConfigsDir, file)
	return err
}

// routerEnv is the env block the generated profile must carry. Each key exists
// because something silently stops working without it:
//
//   - The marker resolves the profile's identity. Without it ProviderForConfig
//     falls back to alias-matching the display name, which matches nothing, so
//     All-In would label and colour itself as Providers[0] — Zhipu / GLM.
//   - ANTHROPIC_BASE_URL is what claude-allin reads to decide there is anything
//     to wrap. With no endpoint, rolefix.UpstreamFromSettings errors and
//     runLoopbackWrappedLaunch runs the child unwrapped: the launch gate still
//     fires (it greps for the picker rows), the router never starts, and every
//     wisp/… id is sent verbatim to the session's own upstream. The value is
//     the real endpoint; the launch rewrites the session's OVERLAY, never this
//     file, so the stored profile keeps naming the truth.
//   - The four window keys are the session's 1M guard, and they have to be
//     declared here: this is the one sub-1M profile stampContextBudget cannot
//     write for, because All-In has no model mappings for it to size a window
//     from. Every roster row is 200k, but the session's STARTING model comes
//     from the user's global settings, and a "[1m]" in that raw string grants
//     the whole session 1M. One key does not cover it — CLAUDE_CODE_DISABLE_1M
//     _CONTEXT is read only by the string-marker branch of the window choice,
//     while the beta and native-1M branches reach 1e6 ungated, so
//     CLAUDE_CODE_AUTO_COMPACT_WINDOW is the direct cap on current versions.
//
// The window set must equal what the ensure-budget sweep would compute for
// rosterWindow, or every install rewrites this file;
// TestEnsureProfile_survives_the_context_budget_sweep is what holds the two
// together. The reserve is outputReserve(200000) — a quarter of the window,
// capped at the 32000 Claude Code would have asked for unprompted.
func routerEnv() map[string]any {
	window := strconv.Itoa(rosterWindow)
	return map[string]any{
		"WISP_DECK_SUBSCRIPTION_PROVIDER": claudeconfig.AllInProvider.Key,
		"ANTHROPIC_BASE_URL":              claudeconfig.AllInProvider.BaseURL,
		"CLAUDE_CODE_MAX_CONTEXT_TOKENS":  window,
		"CLAUDE_CODE_AUTO_COMPACT_WINDOW": window,
		"CLAUDE_CODE_DISABLE_1M_CONTEXT":  "1",
		"CLAUDE_CODE_MAX_OUTPUT_TOKENS":   "32000",
	}
}

// ProfileFile returns the filename the All-In profile is registered under, or
// "" when the machine has none yet. Callers use it to tell creating from
// refreshing: the source gate applies to the first, never the second.
func ProfileFile(listFile string) string {
	for _, config := range claudeconfig.Load(listFile) {
		if strings.EqualFold(config.Name, ProfileName) {
			return config.File
		}
	}
	return ""
}
