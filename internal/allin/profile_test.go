package allin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
