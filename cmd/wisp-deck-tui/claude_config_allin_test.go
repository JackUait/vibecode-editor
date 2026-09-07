package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	execRoot(t, "claude-config", "ensure-allin", "--dir", configs, "--list", list,
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
	if err := os.WriteFile(accountsList, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		execRoot(t, "claude-config", "ensure-allin", "--dir", configs, "--list", list,
			"--accounts-list", accountsList, "--accounts-dir", accounts)
	}
	entries, _ := os.ReadDir(configs)
	if len(entries) != 1 {
		t.Fatalf("wrote %d profiles, want 1", len(entries))
	}
}
