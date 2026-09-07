package allin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
	"github.com/jackuait/wisp-deck/internal/rolefix"
)

func readPicker(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	picker, _ := settings["modelPicker"].(map[string]any)
	if picker == nil {
		t.Fatalf("no modelPicker in %s", data)
	}
	return picker
}

func TestEnsureProfile_creates_the_profile_and_registers_it(t *testing.T) {
	env := rosterEnv(t)
	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	file, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	picker := readPicker(t, filepath.Join(env.ConfigsDir, file))
	options, _ := picker["options"].([]any)
	if len(options) == 0 {
		t.Fatal("no rows written")
	}
	if picker["replaceBuiltInOptions"] != true {
		t.Fatalf("built-ins not replaced: %v", picker)
	}
	registered := false
	for _, config := range readLines(listFile) {
		if config == ProfileName+":"+file {
			registered = true
		}
	}
	if !registered {
		t.Fatalf("not registered in %s", listFile)
	}
}

func TestEnsureProfile_refreshes_rows_without_a_second_registration(t *testing.T) {
	env := rosterEnv(t)
	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	first, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("%q != %q", first, second)
	}
	if lines := readLines(listFile); len(lines) != 1 {
		t.Fatalf("registered %d times: %v", len(lines), lines)
	}
}

func TestEnsureProfile_keeps_every_other_key_in_the_profile(t *testing.T) {
	env := rosterEnv(t)
	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	file, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(env.ConfigsDir, file)
	data, _ := os.ReadFile(path)
	var settings map[string]any
	_ = json.Unmarshal(data, &settings)
	settings["statusLine"] = "keep me"
	patched, _ := json.Marshal(settings)
	_ = os.WriteFile(path, patched, 0o600)

	if _, err := EnsureProfile(env, listFile, env.ConfigsDir); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	_ = json.Unmarshal(data, &settings)
	if settings["statusLine"] != "keep me" {
		t.Fatalf("unrelated key lost: %s", data)
	}
}

func TestEnsureProfile_a_corrupted_existing_profile_fails_loudly_and_is_left_untouched(t *testing.T) {
	env := rosterEnv(t)
	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	file, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(env.ConfigsDir, file)
	corrupted := []byte("{ this is not valid json")
	if err := os.WriteFile(path, corrupted, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := EnsureProfile(env, listFile, env.ConfigsDir); err == nil {
		t.Fatal("expected an error reading a corrupted profile, got nil")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(corrupted) {
		t.Fatalf("corrupted profile was overwritten: got %q, want %q", data, corrupted)
	}
}

func TestEnsureProfile_recomputes_the_roster_on_every_call(t *testing.T) {
	env := rosterEnv(t)
	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	file, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(env.ConfigsDir, file)
	before := readPicker(t, path)
	beforeOptions, _ := before["options"].([]any)

	// A second account appears after the first EnsureProfile call, the same
	// way a real login can be added while the profile already exists.
	existing, err := os.ReadFile(env.AccountsList)
	if err != nil {
		t.Fatal(err)
	}
	updated := string(existing) + "Second:second\n"
	if err := os.WriteFile(env.AccountsList, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := EnsureProfile(env, listFile, env.ConfigsDir); err != nil {
		t.Fatal(err)
	}
	after := readPicker(t, path)
	afterOptions, _ := after["options"].([]any)
	if len(afterOptions) <= len(beforeOptions) {
		t.Fatalf("roster did not grow: before=%d after=%d", len(beforeOptions), len(afterOptions))
	}

	found := false
	for _, row := range afterOptions {
		fields, _ := row.(map[string]any)
		if model, _ := fields["model"].(string); strings.Contains(model, "acct.second/") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no row for the newly registered account in %v", afterOptions)
	}
}

func readEnv(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var settings struct {
		Env map[string]string `json:"env"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	return settings.Env
}

func generatedProfile(t *testing.T) (Env, string, string) {
	t.Helper()
	env := rosterEnv(t)
	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	file, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	return env, file, filepath.Join(env.ConfigsDir, file)
}

// Every place All-In can be chosen — the switcher's rows and the main page's
// subscription ring — hides a config ConfigReady refuses, so a profile that
// fails this is unreachable from the UI no matter how good its picker is.
func TestEnsureProfile_the_generated_profile_is_selectable(t *testing.T) {
	env, file, _ := generatedProfile(t)
	if !claudeconfig.ConfigReady(env.ConfigsDir, claudeconfig.Config{Name: ProfileName, File: file}) {
		t.Fatal("the generated profile is not ConfigReady, so nothing in the UI offers it")
	}
}

// The launch wrapper reads the profile's own endpoint and rewrites it to the
// loopback router. No endpoint means runLoopbackWrappedLaunch falls through and
// every wisp/… row goes verbatim to the session's own upstream.
func TestEnsureProfile_declares_the_endpoint_the_launch_wrapper_rewrites(t *testing.T) {
	_, _, path := generatedProfile(t)
	upstream, err := rolefix.UpstreamFromSettings(path)
	if err != nil {
		t.Fatalf("the router would never start: %v", err)
	}
	if upstream == "" {
		t.Fatal("empty upstream")
	}
}

// A profile whose name matches no provider alias resolves to Providers[0], so
// without an explicit marker All-In labels and colours as Zhipu / GLM.
func TestEnsureProfile_carries_its_own_provider_identity(t *testing.T) {
	env, file, _ := generatedProfile(t)
	provider := claudeconfig.ProviderForConfig(env.ConfigsDir,
		claudeconfig.Config{Name: ProfileName, File: file})
	if provider.Key != claudeconfig.AllInProvider.Key {
		t.Fatalf("provider %q (%s), want the All-In identity", provider.Key, provider.Name)
	}
}

// Every row is 200k (the roster no longer emits [1m]), but the session's
// STARTING model is the user's global one — and Claude Code reads a "[1m]" off
// that raw string alone. Nothing else narrows it here: stampContextBudget
// returns early for a profile with no model mappings, so this profile would be
// the one with no 1M guard at all.
func TestEnsureProfile_disarms_an_inherited_1m_model_marker(t *testing.T) {
	_, _, path := generatedProfile(t)
	if got := readEnv(t, path)["CLAUDE_CODE_DISABLE_1M_CONTEXT"]; got != "1" {
		t.Fatalf("CLAUDE_CODE_DISABLE_1M_CONTEXT = %q, want \"1\"", got)
	}
}

// A sub-1M profile carries FOUR keys, not one. CLAUDE_CODE_DISABLE_1M_CONTEXT
// gates only the string-marker branch of Claude Code's window choice — the
// decoded `sae()` is read by `Ov()` alone, while the beta path
// (`betas.includes(1m) && EW(model)`) and the native path (`L2(model)`) reach
// 1e6 ungated — and CLAUDE_CODE_AUTO_COMPACT_WINDOW is the direct cap on
// current versions. This is the one sub-1M profile stampContextBudget cannot
// write for (it has no model mappings to size a window from), so it declares
// the set itself.
func TestEnsureProfile_declares_every_key_a_200k_window_implies(t *testing.T) {
	_, _, path := generatedProfile(t)
	env := readEnv(t, path)
	for key, want := range map[string]string{
		"CLAUDE_CODE_MAX_CONTEXT_TOKENS":  "200000",
		"CLAUDE_CODE_AUTO_COMPACT_WINDOW": "200000",
		"CLAUDE_CODE_DISABLE_1M_CONTEXT":  "1",
		"CLAUDE_CODE_MAX_OUTPUT_TOKENS":   "32000",
	} {
		if env[key] != want {
			t.Errorf("%s = %q, want %q", key, env[key], want)
		}
	}
}

// bin/wisp-deck runs ensure-budget over every profile on every install. Once a
// window is declared this stops being vacuous: the sweep recomputes all four
// keys from it, and reports no change only if the declared set matches exactly.
func TestEnsureProfile_survives_the_context_budget_sweep(t *testing.T) {
	env, file, path := generatedProfile(t)
	changed, err := claudeconfig.EnsureContextBudget(env.ConfigsDir, file)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("the context-budget sweep rewrote the All-In profile")
	}
	env2 := readEnv(t, path)
	for _, key := range []string{
		"CLAUDE_CODE_MAX_CONTEXT_TOKENS", "CLAUDE_CODE_AUTO_COMPACT_WINDOW",
		"CLAUDE_CODE_DISABLE_1M_CONTEXT", "CLAUDE_CODE_MAX_OUTPUT_TOKENS",
	} {
		if env2[key] == "" {
			t.Errorf("the sweep dropped %s", key)
		}
	}
}
