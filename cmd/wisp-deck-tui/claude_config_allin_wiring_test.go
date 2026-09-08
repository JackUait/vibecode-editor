package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wiringFixture lays out a machine with one login and one ready provider (two
// sources) and returns the four paths the allin-aware commands take.
func wiringFixture(t *testing.T) (list, cfgDir, ptr, accountsList, accountsDir string) {
	t.Helper()
	dir := t.TempDir()
	list = filepath.Join(dir, "claude-configs.list")
	cfgDir = filepath.Join(dir, "claude-configs")
	ptr = filepath.Join(dir, "claude-config")
	accountsList = filepath.Join(dir, "claude-accounts.list")
	accountsDir = filepath.Join(dir, "claude-accounts")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(accountsDir, "personal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(accountsList, []byte("Personal:personal\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "zhipu-glm.json"),
		[]byte(`{"env":{"ANTHROPIC_BASE_URL":"https://api.z.ai/api/anthropic","ANTHROPIC_AUTH_TOKEN":"k"}}`),
		0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(list, []byte("Zhipu GLM:zhipu-glm.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return list, cfgDir, ptr, accountsList, accountsDir
}

// A machine that reaches two sources through the legacy "Manage Claude
// configs" delete flow (lib/config-tui.sh), not the unified Subscription
// modal, must still get the profile — this is the gap the whole feature
// closes, exercised at the CLI boundary that flow actually calls.
func TestClaudeConfigDelete_refreshes_an_existing_allin_profile_when_a_source_disappears(t *testing.T) {
	list, cfgDir, ptr, accountsList, accountsDir := wiringFixture(t)
	execRoot(t, "claude-config", "ensure-allin", "--configs-dir", cfgDir, "--configs-list", list,
		"--accounts-list", accountsList, "--accounts-dir", accountsDir)

	execRoot(t, "claude-config", "delete", "--list", list, "--dir", cfgDir, "--pointer", ptr,
		"--file", "zhipu-glm.json", "--accounts-list", accountsList, "--accounts-dir", accountsDir)

	entries, err := os.ReadDir(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	var allInFile string
	for _, e := range entries {
		if e.Name() != "zhipu-glm.json" {
			allInFile = e.Name()
		}
	}
	if allInFile == "" {
		t.Fatalf("All-In profile did not survive the delete: %v", entries)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, allInFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "cfg.zhipu-glm/") {
		t.Fatalf("deleted provider's rows survived the refresh: %s", data)
	}
}

// Deleting the All-In profile itself must not bring it straight back, even
// though the machine still has two other sources to route between — the
// delete just asked to remove it.
func TestClaudeConfigDelete_never_recreates_the_profile_it_just_deleted(t *testing.T) {
	list, cfgDir, ptr, accountsList, accountsDir := wiringFixture(t)
	execRoot(t, "claude-config", "ensure-allin", "--configs-dir", cfgDir, "--configs-list", list,
		"--accounts-list", accountsList, "--accounts-dir", accountsDir)

	entries, err := os.ReadDir(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	var allInFile string
	for _, e := range entries {
		if e.Name() != "zhipu-glm.json" {
			allInFile = e.Name()
		}
	}
	if allInFile == "" {
		t.Fatal("setup: All-In profile was not created")
	}

	execRoot(t, "claude-config", "delete", "--list", list, "--dir", cfgDir, "--pointer", ptr,
		"--file", allInFile, "--accounts-list", accountsList, "--accounts-dir", accountsDir)

	if _, err := os.Stat(filepath.Join(cfgDir, allInFile)); !os.IsNotExist(err) {
		t.Fatalf("deleted All-In profile came back: %v", err)
	}
	data, _ := os.ReadFile(list)
	if strings.Contains(string(data), "All-In") {
		t.Fatalf("All-In re-registered in the list after its own delete: %s", data)
	}
}

// The legacy "Manage Claude configs" menu (lib/config-tui.sh) reaches the
// gate through `claude-config add`/`delete`, not through the modal's own
// ensureAllIn — the Default login's tag has to reach the roster from there
// too, or a user on that flow never sees their own label.
func TestClaudeConfigAdd_wires_the_default_label_file_into_the_roster(t *testing.T) {
	list, cfgDir, ptr, accountsList, accountsDir := wiringFixture(t)
	labelFile := filepath.Join(t.TempDir(), "claude-account-default-label")
	if err := os.WriteFile(labelFile, []byte("Work"), 0o600); err != nil {
		t.Fatal(err)
	}

	// wiringFixture already lays out one login + one ready provider (two
	// sources), so a second `add` crosses no new threshold — it exercises
	// the refresh path, which is what a real "Manage Claude configs" session
	// actually calls repeatedly.
	execRoot(t, "claude-config", "add", "--list", list, "--dir", cfgDir, "--pointer", ptr,
		"--name", "Moonshot", "--accounts-list", accountsList, "--accounts-dir", accountsDir,
		"--default-label-file", labelFile)

	entries, err := os.ReadDir(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	var allInFile string
	for _, e := range entries {
		if e.Name() != "zhipu-glm.json" && e.Name() != "moonshot.json" {
			allInFile = e.Name()
		}
	}
	if allInFile == "" {
		t.Fatalf("setup: All-In profile was not created: %v", entries)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, allInFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Work ·") {
		t.Fatalf("Default login's tag did not reach the roster via add: %s", data)
	}
	if !strings.Contains(string(data), "acct.default/") {
		t.Fatalf("row id should still name the directory, not the label: %s", data)
	}
}
