package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureAllIn_creates_the_profile_on_a_fresh_machine(t *testing.T) {
	root := t.TempDir()
	configs := filepath.Join(root, "claude-configs")
	accounts := filepath.Join(root, "claude-accounts")
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(accounts, 0o755); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(root, "claude-configs.list")
	accountsList := filepath.Join(root, "claude-accounts.list")
	if err := os.WriteFile(accountsList, []byte("Personal:personal\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	execRoot(t, "claude-config", "ensure-allin", "--configs-dir", configs, "--configs-list", list,
		"--accounts-list", accountsList, "--accounts-dir", accounts)

	entries, err := os.ReadDir(configs)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no profile written: %v %v", entries, err)
	}
	data, err := os.ReadFile(filepath.Join(configs, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	if settings["modelPicker"] == nil {
		t.Fatalf("profile has no picker: %s", data)
	}
}

func TestEnsureAllIn_is_idempotent(t *testing.T) {
	root := t.TempDir()
	configs := filepath.Join(root, "claude-configs")
	accounts := filepath.Join(root, "claude-accounts")
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(accounts, 0o755); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(root, "claude-configs.list")
	accountsList := filepath.Join(root, "claude-accounts.list")
	if err := os.WriteFile(accountsList, []byte("Personal:personal\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		execRoot(t, "claude-config", "ensure-allin", "--configs-dir", configs, "--configs-list", list,
			"--accounts-list", accountsList, "--accounts-dir", accounts)
	}
	entries, _ := os.ReadDir(configs)
	if len(entries) != 1 {
		t.Fatalf("wrote %d profiles, want 1", len(entries))
	}
}

// allInFixture lays out a machine with the given logins list and returns the
// four paths ensure-allin takes.
func allInFixture(t *testing.T, accountsBody string) (configs, list, accountsList, accounts string) {
	t.Helper()
	root := t.TempDir()
	configs = filepath.Join(root, "claude-configs")
	accounts = filepath.Join(root, "claude-accounts")
	for _, dir := range []string{configs, accounts} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	list = filepath.Join(root, "claude-configs.list")
	accountsList = filepath.Join(root, "claude-accounts.list")
	if err := os.WriteFile(accountsList, []byte(accountsBody), 0o600); err != nil {
		t.Fatal(err)
	}
	return configs, list, accountsList, accounts
}

func runEnsureAllIn(t *testing.T, configs, list, accountsList, accounts string) {
	t.Helper()
	execRoot(t, "claude-config", "ensure-allin", "--configs-dir", configs, "--configs-list", list,
		"--accounts-list", accountsList, "--accounts-dir", accounts)
}

// The Default login is seeded unconditionally, so the roster is never empty and
// a length check admits every machine. One source has nothing to route between:
// the rows would only duplicate the built-in lineup behind a proxy.
func TestEnsureAllIn_writes_nothing_for_a_single_source_machine(t *testing.T) {
	configs, list, accountsList, accounts := allInFixture(t, "")
	runEnsureAllIn(t, configs, list, accountsList, accounts)
	entries, err := os.ReadDir(configs)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("wrote %d profiles for a one-source machine: %v", len(entries), entries)
	}
}

// A second login is a second source even with no provider configured at all.
func TestEnsureAllIn_writes_a_profile_for_two_logins(t *testing.T) {
	configs, list, accountsList, accounts := allInFixture(t, "Personal:personal\n")
	runEnsureAllIn(t, configs, list, accountsList, accounts)
	if entries, _ := os.ReadDir(configs); len(entries) != 1 {
		t.Fatalf("wrote %d profiles, want 1", len(entries))
	}
}

// One login plus one ready provider is also two sources.
func TestEnsureAllIn_writes_a_profile_for_one_login_and_one_provider(t *testing.T) {
	configs, list, accountsList, accounts := allInFixture(t, "")
	if err := os.WriteFile(filepath.Join(configs, "zhipu-glm.json"),
		[]byte(`{"env":{"ANTHROPIC_BASE_URL":"https://api.z.ai/api/anthropic","ANTHROPIC_AUTH_TOKEN":"k"}}`),
		0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(list, []byte("Zhipu GLM:zhipu-glm.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runEnsureAllIn(t, configs, list, accountsList, accounts)
	if entries, _ := os.ReadDir(configs); len(entries) != 2 {
		t.Fatalf("no All-In profile beside the provider: %v", entries)
	}
}

// The gate decides whether to CREATE, never whether to refresh. A machine whose
// second login was removed today still has a profile full of rows naming it,
// and every one of them answers 400 until the roster is rebuilt.
func TestEnsureAllIn_refreshes_an_existing_profile_after_its_sources_shrink(t *testing.T) {
	configs, list, accountsList, accounts := allInFixture(t, "Personal:personal\n")
	runEnsureAllIn(t, configs, list, accountsList, accounts)

	if err := os.WriteFile(accountsList, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	runEnsureAllIn(t, configs, list, accountsList, accounts)

	data, err := os.ReadFile(filepath.Join(configs, "all-in.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "acct.personal/") {
		t.Fatalf("rows for the removed login survived: %s", data)
	}
}

// The implicit login's tag has to reach the roster through the CLI too, or a
// user who only ever mutates configs through the legacy config menu
// (lib/config-tui.sh) or the ensure-allin sweep (bin/wisp-deck) never sees
// their own label.
func TestEnsureAllIn_wires_the_default_label_file_into_the_roster(t *testing.T) {
	configs, list, accountsList, accounts := allInFixture(t, "Personal:personal\n")
	labelFile := filepath.Join(t.TempDir(), "claude-account-default-label")
	if err := os.WriteFile(labelFile, []byte("Work"), 0o600); err != nil {
		t.Fatal(err)
	}

	execRoot(t, "claude-config", "ensure-allin", "--configs-dir", configs, "--configs-list", list,
		"--accounts-list", accountsList, "--accounts-dir", accounts, "--default-label-file", labelFile)

	data, err := os.ReadFile(filepath.Join(configs, "all-in.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Work ·") {
		t.Fatalf("Default login's tag did not reach the roster: %s", data)
	}
	if !strings.Contains(string(data), "acct.default/") {
		t.Fatalf("row id should still name the directory, not the label: %s", data)
	}
}
