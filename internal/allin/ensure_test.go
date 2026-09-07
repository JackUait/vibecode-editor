package allin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// EnsureProfileIfEligible is the one gate every mutation site (CLI and TUI)
// shares, so "create at two sources, always refresh an existing one" cannot
// drift between callers. These tests pin exactly that contract; ensure-allin's
// own tests (cmd/wisp-deck-tui) pin that the CLI still delegates to it.
//
// rosterEnv (roster_test.go) always seeds three sources (Default, one login,
// one ready provider), which is the wrong fixture for pinning a boundary at
// two — a mutant that only breaks the exactly-two case would still pass
// against it. ensureFixture instead starts from a bare Default and lets each
// test add exactly the sources its scenario needs.
func ensureFixture(t *testing.T, accountsBody string) Env {
	t.Helper()
	dir := t.TempDir()
	accounts := filepath.Join(dir, "claude-accounts")
	configs := filepath.Join(dir, "claude-configs")
	if err := os.MkdirAll(accounts, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	accountsList := filepath.Join(dir, "claude-accounts.list")
	configsList := filepath.Join(dir, "claude-configs.list")
	if err := os.WriteFile(accountsList, []byte(accountsBody), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configsList, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return Env{AccountsList: accountsList, AccountsDir: accounts, ConfigsList: configsList, ConfigsDir: configs}
}

func TestEnsureProfileIfEligible_writes_nothing_below_two_sources_with_no_profile(t *testing.T) {
	env := ensureFixture(t, "") // Default alone: one source
	if err := EnsureProfileIfEligible(env); err != nil {
		t.Fatal(err)
	}
	if file := ProfileFile(env.ConfigsList); file != "" {
		t.Fatalf("wrote a profile for a one-source machine: %s", file)
	}
}

func TestEnsureProfileIfEligible_writes_a_profile_at_two_sources(t *testing.T) {
	env := ensureFixture(t, "Personal:personal\n") // Default + Personal: two sources
	if err := EnsureProfileIfEligible(env); err != nil {
		t.Fatal(err)
	}
	if ProfileFile(env.ConfigsList) == "" {
		t.Fatal("no profile registered at two sources")
	}
}

// A profile born here has no other creation path to catch up on: it never
// goes through bin/wisp-deck's ensure-watchdog sweep at all, so the event-tier
// stream watchdog must be disarmed at the moment of creation or it is armed
// for the profile's whole life. See root CLAUDE.md's "keepalive buys 30
// pings" section for what an armed watchdog does to a gateway/self-hosted
// stream.
func TestEnsureProfileIfEligible_disarms_the_stream_watchdog_on_a_freshly_created_profile(t *testing.T) {
	env := ensureFixture(t, "Personal:personal\n") // two sources: creates
	if err := EnsureProfileIfEligible(env); err != nil {
		t.Fatal(err)
	}
	file := ProfileFile(env.ConfigsList)
	if file == "" {
		t.Fatal("setup: profile was not created")
	}
	data, err := os.ReadFile(filepath.Join(env.ConfigsDir, file))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	env2, _ := settings["env"].(map[string]any)
	if env2 == nil {
		t.Fatalf("profile has no env block: %s", data)
	}
	if got, _ := env2[claudeconfig.StreamWatchdogKey].(string); got != "0" {
		t.Fatalf("%s = %q, want \"0\" (disarmed) on a freshly created profile", claudeconfig.StreamWatchdogKey, got)
	}
}

func TestEnsureProfileIfEligible_refreshes_an_existing_profile_below_two_sources(t *testing.T) {
	env := ensureFixture(t, "Personal:personal\n") // two sources: creates
	if err := EnsureProfileIfEligible(env); err != nil {
		t.Fatal(err)
	}
	file := ProfileFile(env.ConfigsList)
	if file == "" {
		t.Fatal("setup: profile was not created at two sources")
	}
	// Drop back to one source; the gate must still rebuild the existing
	// profile's rows, even though a fresh machine at this count gets nothing.
	if err := os.WriteFile(env.AccountsList, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProfileIfEligible(env); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(env.ConfigsDir, file))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "acct.personal/") {
		t.Fatalf("stale login rows survived a refresh: %s", data)
	}
}

// A caller missing one of the four paths must be refused outright rather than
// run partway: Roster reads a missing AccountsList as "no logins", so acting
// on an incomplete Env would rewrite modelPicker with real rows stripped out
// from under it. No production caller legitimately has an empty path — every
// site builds all four from the same config root before calling this.
// Every one of the six production call sites discards this error (a failed
// refresh must never fail the mutation the user asked for), so this is the
// ONLY place a structurally broken caller (missing one of the four paths)
// leaves a trace at all. A silent nil here would make that caller invisible
// forever.
func TestEnsureProfileIfEligible_refuses_an_incomplete_env(t *testing.T) {
	env := ensureFixture(t, "Personal:personal\n") // otherwise eligible: two sources
	env.AccountsList = ""
	if err := EnsureProfileIfEligible(env); err == nil {
		t.Fatal("expected an error for an incomplete Env, got nil")
	}
	if file := ProfileFile(env.ConfigsList); file != "" {
		t.Fatalf("wrote a profile from an incomplete env: %s", file)
	}
}

func BenchmarkEnsureProfileIfEligible(b *testing.B) {
	dir := b.TempDir()
	accounts := filepath.Join(dir, "claude-accounts")
	configs := filepath.Join(dir, "claude-configs")
	if err := os.MkdirAll(filepath.Join(accounts, "personal"), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.MkdirAll(configs, 0o755); err != nil {
		b.Fatal(err)
	}
	accountsList := filepath.Join(dir, "claude-accounts.list")
	configsList := filepath.Join(dir, "claude-configs.list")
	if err := os.WriteFile(accountsList, []byte("Personal:personal\nWork:work\n"), 0o600); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(configsList,
		[]byte("Zhipu GLM:zhipu-glm.json\nMoonshot:moonshot.json\n"), 0o600); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configs, "zhipu-glm.json"),
		[]byte(`{"env":{"ANTHROPIC_BASE_URL":"https://api.z.ai/api/anthropic","ANTHROPIC_AUTH_TOKEN":"k"}}`),
		0o600); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configs, "moonshot.json"),
		[]byte(`{"env":{"ANTHROPIC_BASE_URL":"https://api.moonshot.ai/anthropic","ANTHROPIC_AUTH_TOKEN":"k"}}`),
		0o600); err != nil {
		b.Fatal(err)
	}
	env := Env{
		AccountsList: accountsList,
		AccountsDir:  accounts,
		ConfigsList:  configsList,
		ConfigsDir:   configs,
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := EnsureProfileIfEligible(env); err != nil {
			b.Fatal(err)
		}
	}
}
