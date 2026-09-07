package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLiveClaudeAllInReachesTheChatGPTEngine drives the whole chain a ChatGPT
// picker row takes — roster id, Resolve, a lazily started real Codex
// app-server, the loopback bridge, the engine's own model allowlist — without
// spending a turn: the model it asks for is one no app-server reports, so the
// engine refuses it before any generation happens.
//
// A green run means the row works. A red one distinguishes where it broke: a
// 400 naming Codex is a bridge that would not start, a 502 is the router
// failing open, and a 200 would mean the engine ran something.
//
//	WISP_DECK_LIVE_ALLIN_CHATGPT_E2E=1 go test ./cmd/wisp-deck-tui/ \
//	  -run TestLiveClaudeAllInReachesTheChatGPTEngine -v
func TestLiveClaudeAllInReachesTheChatGPTEngine(t *testing.T) {
	if os.Getenv("WISP_DECK_LIVE_ALLIN_CHATGPT_E2E") == "" {
		t.Skip("set WISP_DECK_LIVE_ALLIN_CHATGPT_E2E=1 to start a real Codex app-server")
	}
	dir := t.TempDir()
	configs := filepath.Join(dir, "configs")
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	configsList := filepath.Join(dir, "claude-configs.list")
	write(configsList, "OpenAI / ChatGPT:openai-chatgpt.json\n")
	write(filepath.Join(configs, "openai-chatgpt.json"),
		`{"env":{"WISP_DECK_SUBSCRIPTION_PROVIDER":"openai-chatgpt"}}`)
	settings := filepath.Join(dir, "overlay.json")
	write(settings, `{"env":{"ANTHROPIC_BASE_URL":"https://api.anthropic.com"}}`)

	var status int
	var body string
	command := newClaudeAllInCommand(func([]string) error {
		data, err := os.ReadFile(settings)
		if err != nil {
			t.Errorf("read the rewritten overlay: %v", err)
			return nil
		}
		var parsed struct {
			Env map[string]string `json:"env"`
		}
		_ = json.Unmarshal(data, &parsed)
		router := parsed.Env["ANTHROPIC_BASE_URL"]
		if !hasLoopbackPrefix(router) {
			t.Errorf("child saw %q, not a loopback router", router)
			return nil
		}
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			router+"/v1/messages", bytes.NewReader([]byte(
				`{"model":"wisp/cfg.openai-chatgpt/wisp-deck-no-such-model",`+
					`"max_tokens":16,"stream":false,`+
					`"messages":[{"role":"user","content":"hi"}]}`)))
		if err != nil {
			t.Error(err)
			return nil
		}
		request.Header.Set("content-type", "application/json")
		request.Header.Set("anthropic-version", "2023-06-01")
		client := &http.Client{Timeout: 90 * time.Second}
		start := time.Now()
		response, err := client.Do(request)
		if err != nil {
			t.Errorf("router did not answer: %v", err)
			return nil
		}
		defer response.Body.Close()
		raw, _ := io.ReadAll(response.Body)
		t.Logf("router answered HTTP %d in %s: %s", response.StatusCode, time.Since(start), raw)
		status, body = response.StatusCode, string(raw)
		return nil
	})
	command.SetArgs([]string{"--settings", settings,
		"--configs-list", configsList, "--configs-dir", configs, "--", "true"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	if status != http.StatusBadRequest {
		t.Fatalf("HTTP %d: %s", status, body)
	}
	if !strings.Contains(body, "is not available") {
		t.Fatalf("the request never reached the engine's model allowlist: %s", body)
	}
}
