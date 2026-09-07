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
