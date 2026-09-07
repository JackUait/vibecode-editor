package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The child is launched against a live local router: proving the rewritten
// URL is already accepting connections is what rules out a first-turn race
// against a router that has not started serving yet.
//
// Two things pin the shape of the probe. The overlay must name a NON-local
// endpoint — UpstreamFromSettings refuses 127.0.0.1/localhost/::1 as "already
// local", so pointing the fixture at an httptest server makes the wrapper fall
// through and start no router at all (verified: the child then still sees the
// fixture URL). And the probe must be one the router answers ITSELF: a request
// with no routable model is KindSession, which the router forwards to the
// overlay's own upstream — the earlier GET /healthz really did reach
// api.anthropic.com, which answered 404 through cloudflare. A cfg. row naming a
// profile that does not exist is refused locally, so nothing leaves the process
// and the 5s deadline covers loopback only. An acct. row would not do: an empty
// accounts dir sends keychainToken at the developer's real Keychain.
func TestClaudeAllIn_points_the_overlay_at_the_local_router(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(settings,
		[]byte(`{"env":{"ANTHROPIC_BASE_URL":"https://api.anthropic.com"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var seen string
	command := newClaudeAllInCommand(func([]string) error {
		data, _ := os.ReadFile(settings)
		var parsed struct {
			Env map[string]string `json:"env"`
		}
		_ = json.Unmarshal(data, &parsed)
		seen = parsed.Env["ANTHROPIC_BASE_URL"]
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Post(seen+"/v1/messages", "application/json",
			strings.NewReader(`{"model":"wisp/cfg.absent/x"}`))
		if err != nil {
			t.Errorf("router is not listening when the child starts: %v", err)
			return nil
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "absent") {
			t.Errorf("the router did not answer the probe itself: status %d, body %s",
				resp.StatusCode, body)
		}
		return nil
	})
	command.SetArgs([]string{"--settings", settings,
		"--configs-dir", filepath.Join(dir, "configs"), "--", "true"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !hasLoopbackPrefix(seen) {
		t.Fatalf("child saw %q, not a loopback router", seen)
	}
}

func TestClaudeAllIn_runs_the_child_when_the_overlay_cannot_be_read(t *testing.T) {
	ran := false
	command := newClaudeAllInCommand(func([]string) error { ran = true; return nil })
	command.SetArgs([]string{"--settings", filepath.Join(t.TempDir(), "absent.json"), "--", "true"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("child never ran")
	}
}

func hasLoopbackPrefix(url string) bool {
	return len(url) > 17 && url[:17] == "http://127.0.0.1:"
}
