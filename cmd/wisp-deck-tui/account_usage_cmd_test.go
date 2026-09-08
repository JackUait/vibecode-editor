package main

import (
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

// accountUsageFixture lays out the two files the command enumerates logins
// from, and swaps the Keychain read for a per-login stub.
func accountUsageFixture(t *testing.T) (root, accountsList, accountsDir, configsList string) {
	t.Helper()
	root = t.TempDir()
	accountsDir = filepath.Join(root, "claude-accounts")
	if err := os.MkdirAll(filepath.Join(accountsDir, "personal"), 0o755); err != nil {
		t.Fatal(err)
	}
	accountsList = filepath.Join(root, "claude-accounts.list")
	if err := os.WriteFile(accountsList, []byte("Personal:personal\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configsList = filepath.Join(root, "claude-configs.list")
	if err := os.WriteFile(configsList, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	previous := accountUsageToken
	accountUsageToken = func(dir, login string) (string, error) { return "tok-" + login, nil }
	t.Cleanup(func() { accountUsageToken = previous })
	return root, accountsList, accountsDir, configsList
}

func TestAccountUsageCmd_Registered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"account-usage"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if cmd.Name() != "account-usage" {
		t.Errorf("resolved to %q", cmd.Name())
	}
}

// Every login gets a cache, including the implicit one: the picker offers rows
// for all of them, so a reading for only the session's own login would leave
// every other row unannotated forever.
func TestAccountUsageCmd_writes_a_cache_for_every_login(t *testing.T) {
	seen := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.Header.Get("Authorization")] = true
		_, _ = fmt.Fprint(w, `{"five_hour":{"utilization":24.0,"resets_at":null},
		  "seven_day":{"utilization":40.0,"resets_at":null}}`)
	}))
	defer srv.Close()

	_, accountsList, accountsDir, configsList := accountUsageFixture(t)
	execRoot(t, "account-usage",
		"--accounts-list", accountsList, "--accounts-dir", accountsDir,
		"--configs-list", configsList, "--base-url", srv.URL)

	for _, login := range []string{"default", "personal"} {
		snap, err := subusage.ReadCache(allin.AccountUsageFile(configsList, login))
		if err != nil {
			t.Fatalf("no cache for %s: %v", login, err)
		}
		if snap.RateLimits.SevenDay == nil || snap.RateLimits.SevenDay.UsedPercentage != 40 {
			t.Fatalf("%s: rate limits = %+v", login, snap.RateLimits)
		}
		if snap.FetchedAt == 0 || snap.CheckedAt == 0 {
			t.Fatalf("%s: stamps = %+v", login, snap)
		}
		if !seen["Bearer tok-"+login] {
			t.Fatalf("%s was fetched with the wrong login's token", login)
		}
	}
}

// A refused login must not blank a reading the picker is still using: only the
// attempt stamp moves, so the throttle backs off without losing the numbers.
func TestAccountUsageCmd_keeps_the_last_good_reading_when_a_fetch_fails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, accountsList, accountsDir, configsList := accountUsageFixture(t)
	cache := allin.AccountUsageFile(configsList, "default")
	good := subusage.Snapshot{
		RateLimits: subusage.RateLimits{SevenDay: &subusage.Window{UsedPercentage: 12}},
		FetchedAt:  time.Now().Add(-time.Hour).Unix(),
		CheckedAt:  time.Now().Add(-time.Hour).Unix(),
	}
	if err := subusage.WriteCache(cache, good); err != nil {
		t.Fatal(err)
	}

	execRoot(t, "account-usage",
		"--accounts-list", accountsList, "--accounts-dir", accountsDir,
		"--configs-list", configsList, "--base-url", srv.URL, "--min-interval", "0")

	snap, err := subusage.ReadCache(cache)
	if err != nil {
		t.Fatal(err)
	}
	if snap.RateLimits.SevenDay == nil || snap.RateLimits.SevenDay.UsedPercentage != 12 {
		t.Fatalf("the last good reading was lost: %+v", snap.RateLimits)
	}
	if snap.FetchedAt != good.FetchedAt {
		t.Fatalf("a failed fetch moved fetched_at: %d", snap.FetchedAt)
	}
	if snap.CheckedAt <= good.CheckedAt {
		t.Fatalf("a failed fetch must still stamp checked_at: %d", snap.CheckedAt)
	}
}

func TestAccountUsageCmd_skips_a_login_checked_recently(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = fmt.Fprint(w, `{"seven_day":{"utilization":5.0,"resets_at":null}}`)
	}))
	defer srv.Close()

	_, accountsList, accountsDir, configsList := accountUsageFixture(t)
	for _, login := range []string{"default", "personal"} {
		if err := subusage.WriteCache(allin.AccountUsageFile(configsList, login),
			subusage.Snapshot{CheckedAt: time.Now().Unix()}); err != nil {
			t.Fatal(err)
		}
	}

	execRoot(t, "account-usage",
		"--accounts-list", accountsList, "--accounts-dir", accountsDir,
		"--configs-list", configsList, "--base-url", srv.URL, "--min-interval", "300")

	if calls != 0 {
		t.Fatalf("the throttle was ignored: %d requests", calls)
	}
}
