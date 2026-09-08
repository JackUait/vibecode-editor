package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackuait/wisp-deck/internal/allin"
	"github.com/jackuait/wisp-deck/internal/subusage"
)

// allInUsageFixture lays out one login list, one ready Zhipu profile pointed at
// the caller's stub host, and swaps the Keychain read for a per-login stub.
func allInUsageFixture(t *testing.T, zhipuHost string) allin.Env {
	t.Helper()
	root := t.TempDir()
	accountsDir := filepath.Join(root, "claude-accounts")
	configsDir := filepath.Join(root, "claude-configs")
	for _, dir := range []string{filepath.Join(accountsDir, "personal"), configsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "claude-accounts.list"), "Personal:personal\n")
	write(filepath.Join(root, "claude-configs.list"), "Zhipu GLM:zhipu-glm.json\n")
	write(filepath.Join(configsDir, "zhipu-glm.json"), fmt.Sprintf(
		`{"env":{"ANTHROPIC_BASE_URL":%q,"ANTHROPIC_AUTH_TOKEN":"k"}}`, zhipuHost+"/api/anthropic"))

	previous := accountUsageToken
	accountUsageToken = func(_, login string) (string, error) { return "tok-" + login, nil }
	t.Cleanup(func() { accountUsageToken = previous })

	return allin.Env{
		AccountsList: filepath.Join(root, "claude-accounts.list"),
		AccountsDir:  accountsDir,
		ConfigsList:  filepath.Join(root, "claude-configs.list"),
		ConfigsDir:   configsDir,
	}
}

func pickerDescriptions(t *testing.T, env allin.Env) map[string]string {
	t.Helper()
	file := allin.ProfileFile(env.ConfigsList)
	if file == "" {
		t.Fatal("the All-In profile was never written")
	}
	data, err := os.ReadFile(filepath.Join(env.ConfigsDir, file))
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		ModelPicker struct {
			Options []allin.Row `json:"options"`
		} `json:"modelPicker"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, row := range parsed.ModelPicker.Options {
		out[row.Model] = row.Description
	}
	return out
}

// One round refreshes every source the picker offers — the logins no pane is
// running included — and then rewrites the profile, which is the only thing
// that puts the numbers in front of the user.
func TestRefreshAllInUsage_annotates_the_picker_from_every_source(t *testing.T) {
	zhipu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"limits":[
		  {"type":"TOKENS_LIMIT","unit":3,"number":5,"percentage":25.0,"nextResetTime":0}]}}`)
	}))
	defer zhipu.Close()
	anthropic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		used := 10.0
		if r.Header.Get("Authorization") == "Bearer tok-personal" {
			used = 82.0
		}
		_, _ = fmt.Fprintf(w, `{"five_hour":{"utilization":%f,"resets_at":null}}`, used)
	}))
	defer anthropic.Close()

	env := allInUsageFixture(t, zhipu.URL)
	refreshAllInUsageWith(usageRefreshOptions{env: env, anthropic: anthropic.URL})

	// Keyed by source rather than by model id: every row spending one
	// credential carries that credential's number, and a Claude row's id also
	// carries the 1M marker the roster appends.
	want := map[string]string{
		"acct.default":  "Claude subscription: Default · 90% left",
		"acct.personal": "Claude subscription: Personal · 18% left",
		"cfg.zhipu-glm": "Zhipu / GLM · 75% left",
	}
	seen := map[string]bool{}
	for model, got := range pickerDescriptions(t, env) {
		source := allin.SourceKey(model)
		if want[source] == "" {
			t.Errorf("row %q spends an unexpected source %q", model, source)
			continue
		}
		seen[source] = true
		if got != want[source] {
			t.Errorf("%s description = %q, want %q", model, got, want[source])
		}
	}
	for source := range want {
		if !seen[source] {
			t.Errorf("no picker row spends %q", source)
		}
	}
}

// The refresh is throttled per source exactly like subscription-usage: every
// All-In launch arms it, and a deck opens tabs in bursts.
func TestRefreshAllInUsage_leaves_a_source_it_checked_recently_alone(t *testing.T) {
	calls := 0
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"limits":[]}}`)
	}))
	defer stub.Close()

	env := allInUsageFixture(t, stub.URL)
	now := time.Now().Unix()
	for _, cache := range []string{
		allin.AccountUsageFile(env.ConfigsList, "default"),
		allin.AccountUsageFile(env.ConfigsList, "personal"),
		allin.ConfigUsageFile(env.ConfigsList, "zhipu-glm"),
	} {
		if err := subusage.WriteCache(cache, subusage.Snapshot{CheckedAt: now}); err != nil {
			t.Fatal(err)
		}
	}

	refreshAllInUsageWith(usageRefreshOptions{env: env, anthropic: stub.URL, minInterval: 300})

	if calls != 0 {
		t.Fatalf("the throttle was ignored: %d requests", calls)
	}
}

// Claude Code snapshots modelPicker once, at launch, so the refresh can only
// make the NEXT tab's numbers fresh — which is exactly why the launch must arm
// it and must not wait for it.
func TestClaudeAllIn_arms_the_usage_refresh_without_waiting_for_it(t *testing.T) {
	armed := make(chan allin.Env, 1)
	previous := allInUsageRefresh
	allInUsageRefresh = func(env allin.Env) { armed <- env }
	t.Cleanup(func() { allInUsageRefresh = previous })

	dir := t.TempDir()
	command := newClaudeAllInCommand(func([]string) error { return nil })
	command.SetArgs([]string{
		"--settings", filepath.Join(dir, "absent.json"),
		"--accounts-list", filepath.Join(dir, "accounts.list"),
		"--accounts-dir", filepath.Join(dir, "accounts"),
		"--configs-list", filepath.Join(dir, "configs.list"),
		"--configs-dir", filepath.Join(dir, "configs"),
		"--", "true"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	select {
	case env := <-armed:
		if env.ConfigsList != filepath.Join(dir, "configs.list") {
			t.Fatalf("the refresh was armed with %+v", env)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the launch never armed the usage refresh")
	}
}
