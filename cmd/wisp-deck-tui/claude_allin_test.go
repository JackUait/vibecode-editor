package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The child is launched against a live local router: proving the rewritten
// URL is already accepting connections is what rules out a first-turn race
// against a router that has not started serving yet.
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
		resp, err := client.Get(seen + "/healthz")
		if err != nil {
			t.Errorf("router is not listening when the child starts: %v", err)
			return nil
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	})
	command.SetArgs([]string{"--settings", settings, "--", "true"})
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
