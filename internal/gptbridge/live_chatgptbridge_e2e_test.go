package gptbridge

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

// liveCodexPath resolves the same executable the launch chain stamps into a
// pane, falling back to PATH for a developer running this by hand.
func liveCodexPath(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("WISP_DECK_CODEX_CMD"); path != "" {
		return path
	}
	path, err := exec.LookPath("codex")
	if err != nil {
		t.Skipf("codex is not installed: %v", err)
	}
	return path
}

// TestLiveChatGPTBridgeStartsAndServes measures what a lazy start costs on this
// machine and proves the facade reaches a real signed-in Codex. It runs no
// turn, so it costs no quota.
//
// Run it after a codex upgrade:
//
//	WISP_DECK_LIVE_CHATGPT_BRIDGE_E2E=1 go test ./internal/gptbridge/ \
//	  -run TestLiveChatGPTBridgeStartsAndServes -v
func TestLiveChatGPTBridgeStartsAndServes(t *testing.T) {
	if os.Getenv("WISP_DECK_LIVE_CHATGPT_BRIDGE_E2E") == "" {
		t.Skip("set WISP_DECK_LIVE_CHATGPT_BRIDGE_E2E=1 to start a real Codex app-server")
	}
	bridge := NewChatGPTBridge(ChatGPTBridgeOptions{
		CodexPath: liveCodexPath(t), ClientVersion: "live-test",
	})
	t.Cleanup(bridge.Close)

	start := time.Now()
	url, key, err := bridge.Endpoint()
	if err != nil {
		t.Fatalf("start the ChatGPT bridge: %v", err)
	}
	coldStart := time.Since(start)
	t.Logf("lazy start took %s (Claude Code paints its stall banner after 20s of byte silence)", coldStart)
	if coldStart >= 20*time.Second {
		t.Fatalf("a lazy start of %s is past the byte-stall banner threshold; "+
			"the router must keep the socket warm before this can ship", coldStart)
	}

	warm := time.Now()
	secondURL, secondKey, err := bridge.Endpoint()
	if err != nil {
		t.Fatalf("second Endpoint: %v", err)
	}
	t.Logf("reused endpoint in %s", time.Since(warm))
	if secondURL != url || secondKey != key {
		t.Fatal("the second turn was handed a different endpoint")
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url+"/health", nil)
	if err != nil {
		t.Fatalf("build health request: %v", err)
	}
	request.Header.Set("x-api-key", key)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("bridge health: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("bridge health answered HTTP %d", response.StatusCode)
	}
}

// TestLiveCodexAppServerExitsWhenItsParentPipeCloses pins the fact the
// shutdown story rests on: a Codex app-server whose stdin reaches EOF exits by
// itself. claude-allin runs Close on every exit route it controls, but a
// respawn-pane or a SIGKILL runs no Go defer at all — and there the closing of
// the parent's pipe is the only thing left that reaps the 220MB child.
//
//	WISP_DECK_LIVE_CODEX_EOF_E2E=1 go test ./internal/gptbridge/ \
//	  -run TestLiveCodexAppServerExitsWhenItsParentPipeCloses -v
func TestLiveCodexAppServerExitsWhenItsParentPipeCloses(t *testing.T) {
	if os.Getenv("WISP_DECK_LIVE_CODEX_EOF_E2E") == "" {
		t.Skip("set WISP_DECK_LIVE_CODEX_EOF_E2E=1 to start a real Codex app-server")
	}
	command := exec.Command(liveCodexPath(t), "app-server")
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatalf("open app-server stdin: %v", err)
	}
	command.Stdout, command.Stderr = nil, nil
	if err := command.Start(); err != nil {
		t.Fatalf("start app-server: %v", err)
	}
	exited := make(chan error, 1)
	go func() { exited <- command.Wait() }()

	select {
	case err := <-exited:
		t.Fatalf("app-server exited before its stdin closed: %v", err)
	case <-time.After(2 * time.Second):
	}

	closed := time.Now()
	if err := stdin.Close(); err != nil {
		t.Fatalf("close app-server stdin: %v", err)
	}
	select {
	case <-exited:
		t.Logf("app-server exited %s after stdin EOF", time.Since(closed))
	case <-time.After(10 * time.Second):
		_ = command.Process.Kill()
		t.Fatal("app-server outlived its stdin by 10s: a killed claude-allin would now leak it, " +
			"so the launch wrapper needs signal forwarding")
	}
}
