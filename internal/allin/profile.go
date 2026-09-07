package allin

import (
	"encoding/json"
	"os"
	"path/filepath"
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
//   - CLAUDE_CODE_DISABLE_1M_CONTEXT is the only 1M guard this profile can get.
//     Every roster row is 200k, but the session's starting model comes from the
//     user's global settings and Claude Code grants 1M off a "[1m]" in that raw
//     string alone. The sweep that stamps this key on every other sub-1M
//     profile (stampContextBudget) returns early here, because All-In declares
//     no model mappings for it to size a window from.
func routerEnv() map[string]any {
	return map[string]any{
		"WISP_DECK_SUBSCRIPTION_PROVIDER": claudeconfig.AllInProvider.Key,
		"ANTHROPIC_BASE_URL":              claudeconfig.AllInProvider.BaseURL,
		"CLAUDE_CODE_DISABLE_1M_CONTEXT":  "1",
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
