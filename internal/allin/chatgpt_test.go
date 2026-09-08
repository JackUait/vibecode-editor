package allin

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// chatGPTProfile registers a ChatGPT subscription in the fixture. It carries no
// endpoint and no key, which is the whole point: Codex authenticates it and the
// bridge serves it.
func chatGPTProfile(t *testing.T, env Env) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(env.ConfigsDir, "openai-chatgpt.json"),
		[]byte(`{"env":{"WISP_DECK_SUBSCRIPTION_PROVIDER":"openai-chatgpt"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.ConfigsList,
		[]byte("Zhipu GLM:zhipu-glm.json\nOpenAI / ChatGPT:openai-chatgpt.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// fakeBridge counts starts and answers with a fixed endpoint, so a test can
// tell "this turn asked for a bridge" from "this turn did not".
type fakeBridge struct {
	mu    sync.Mutex
	calls int
	url   string
	key   string
	err   error
}

func (b *fakeBridge) Endpoint() (string, string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.calls++
	if b.err != nil {
		return "", "", b.err
	}
	return b.url, b.key, nil
}

func (b *fakeBridge) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.calls
}

func TestRoster_admits_a_chatgpt_profile(t *testing.T) {
	env := rosterEnv(t)
	chatGPTProfile(t, env)

	got := models(Roster(env))
	if !has(got, "wisp/cfg.openai-chatgpt/gpt-6-astra") {
		t.Fatalf("no ChatGPT row in %v", got)
	}
	if !has(got, "wisp/cfg.openai-chatgpt/gpt-5.6-terra") {
		t.Fatalf("no gpt-5.6-terra row in %v", got)
	}
}

// The whole 5.6/6 tier is 272000 tokens. Only a Claude row carries the marker:
// a marked ChatGPT row would tell the session it has 1M against an endpoint
// that refuses anything past 272k.
func TestRoster_never_marks_a_chatgpt_row_1m(t *testing.T) {
	env := rosterEnv(t)
	chatGPTProfile(t, env)

	for _, row := range Roster(env) {
		if Route(row.Model).Kind != KindConfig {
			continue
		}
		if strings.Contains(strings.ToLower(row.Model), "[1m]") {
			t.Fatalf("roster offered a 1M provider row: %s", row.Model)
		}
	}
}

func TestRoster_omits_a_chatgpt_model_too_narrow_for_claude_code(t *testing.T) {
	env := rosterEnv(t)
	chatGPTProfile(t, env)

	// gpt-5.3-codex-spark declares 128000, under Claude Code's own ~20k floor
	// plus the quarter-window output reserve.
	for _, id := range models(Roster(env)) {
		if strings.Contains(id, "gpt-5.3-codex-spark") {
			t.Fatalf("a 128000-token model was offered: %s", id)
		}
	}
}

func TestResolve_serves_a_chatgpt_target_through_the_bridge(t *testing.T) {
	env := rosterEnv(t)
	chatGPTProfile(t, env)
	bridge := &fakeBridge{url: "http://127.0.0.1:54321", key: "sk-wisp-abc"}
	resolver := NewResolver(env)
	resolver.Bridge = bridge

	got, err := resolver.Resolve(Target{Kind: KindConfig, Source: "openai-chatgpt", Model: "gpt-6-astra"})
	if err != nil {
		t.Fatal(err)
	}
	if got.BaseURL != bridge.url {
		t.Fatalf("BaseURL = %q, want the bridge's own %q", got.BaseURL, bridge.url)
	}
	if got.Header != "Authorization" || got.Value != "Bearer sk-wisp-abc" {
		t.Fatalf("credential = %s: %q", got.Header, got.Value)
	}
	if got.NeedsRepair {
		// rolefix repairs a Featherless request. The bridge speaks the
		// Anthropic API itself, so running it through those rewrites would
		// strip fields it is built to read.
		t.Fatal("a ChatGPT target was marked for rolefix repair")
	}
	if bridge.count() != 1 {
		t.Fatalf("bridge asked for an endpoint %d times; want 1", bridge.count())
	}
}

func TestResolve_reports_a_bridge_that_cannot_start(t *testing.T) {
	env := rosterEnv(t)
	chatGPTProfile(t, env)
	resolver := NewResolver(env)
	resolver.Bridge = &fakeBridge{err: errors.New("Codex is signed out; run `codex login`")}

	_, err := resolver.Resolve(Target{Kind: KindConfig, Source: "openai-chatgpt", Model: "gpt-6-astra"})
	if err == nil {
		t.Fatal("a bridge that cannot start resolved anyway")
	}
	if !strings.Contains(err.Error(), "codex login") {
		t.Fatalf("error %q does not name the reason", err)
	}
	if !strings.HasPrefix(err.Error(), "allin:") {
		t.Fatalf("error %q does not carry the package prefix", err)
	}
}

// Resolve is reachable from a build that never wired a bridge (a hand-typed id,
// a picker default saved by another launch shape). It must say so, not panic.
func TestResolve_reports_a_chatgpt_target_with_no_bridge_wired(t *testing.T) {
	env := rosterEnv(t)
	chatGPTProfile(t, env)

	_, err := NewResolver(env).Resolve(Target{Kind: KindConfig, Source: "openai-chatgpt", Model: "gpt-6-astra"})
	if err == nil {
		t.Fatal("a ChatGPT target resolved with no bridge")
	}
	if !strings.Contains(err.Error(), "ChatGPT") {
		t.Fatalf("error %q does not name the subscription", err)
	}
}

func TestHandler_sends_a_chatgpt_turn_to_the_bridge_with_the_bare_codex_model_id(t *testing.T) {
	env := rosterEnv(t)
	chatGPTProfile(t, env)
	var gotModel, gotAuth, gotAPIKey string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		_ = json.Unmarshal(body, &parsed)
		gotModel, _ = parsed["model"].(string)
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("X-Api-Key")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	resolver := NewResolver(env)
	resolver.Bridge = &fakeBridge{url: upstream.URL, key: "sk-wisp-abc"}
	server := httptest.NewServer(NewHandler(resolver, "https://api.anthropic.com"))
	defer server.Close()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		server.URL+"/v1/messages",
		bytes.NewReader([]byte(`{"model":"wisp/cfg.openai-chatgpt/gpt-6-astra","messages":[]}`)))
	if err != nil {
		t.Fatal(err)
	}
	// The session's own OAuth bearer, which must never reach the bridge.
	request.Header.Set("Authorization", "Bearer session-oauth")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("router answered HTTP %d", response.StatusCode)
	}
	if gotModel != "gpt-6-astra" {
		t.Fatalf("bridge received model %q, want the bare Codex id gpt-6-astra", gotModel)
	}
	if gotAuth != "Bearer sk-wisp-abc" {
		t.Fatalf("bridge received Authorization %q", gotAuth)
	}
	if gotAPIKey != "" {
		t.Fatalf("bridge received a stray X-Api-Key %q", gotAPIKey)
	}
}

func TestHandler_never_starts_a_bridge_for_a_non_chatgpt_turn(t *testing.T) {
	env := rosterEnv(t)
	chatGPTProfile(t, env)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	bridge := &fakeBridge{url: upstream.URL, key: "sk-wisp-abc"}
	resolver := NewResolver(env)
	resolver.Bridge = bridge
	// Rewritten so the Zhipu row's own upstream is the local fixture rather
	// than api.z.ai; nothing may leave this process.
	if err := os.WriteFile(filepath.Join(env.ConfigsDir, "zhipu-glm.json"),
		[]byte(`{"env":{"ANTHROPIC_BASE_URL":"`+upstream.URL+`","ANTHROPIC_AUTH_TOKEN":"k"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewHandler(resolver, upstream.URL))
	defer server.Close()

	for _, model := range []string{"wisp/cfg.zhipu-glm/glm-4.7", "claude-opus-5"} {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			server.URL+"/v1/messages",
			bytes.NewReader([]byte(`{"model":"`+model+`","messages":[]}`)))
		if err != nil {
			t.Fatal(err)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("%s: %v", model, err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s answered HTTP %d", model, response.StatusCode)
		}
	}
	if bridge.count() != 0 {
		t.Fatalf("a non-ChatGPT turn started the bridge %d times", bridge.count())
	}
}

func TestHandler_reports_a_bridge_that_cannot_start_as_400_not_502(t *testing.T) {
	env := rosterEnv(t)
	chatGPTProfile(t, env)
	resolver := NewResolver(env)
	resolver.Bridge = &fakeBridge{err: errors.New("Codex is signed out; run `codex login`")}
	server := httptest.NewServer(NewHandler(resolver, "https://api.anthropic.com"))
	defer server.Close()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		server.URL+"/v1/messages",
		bytes.NewReader([]byte(`{"model":"wisp/cfg.openai-chatgpt/gpt-6-astra","messages":[]}`)))
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("answered HTTP %d; Claude Code retries anything but 400 about eleven times",
			response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if !strings.Contains(string(body), "codex login") {
		t.Fatalf("the 400 does not name the reason: %s", body)
	}
	if strings.Contains(string(body), "wisp-deck: wisp-deck:") {
		t.Fatalf("doubled package prefix: %s", body)
	}
}
