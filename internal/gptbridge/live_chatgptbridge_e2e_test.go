package gptbridge

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
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

// TestLiveChatGPTCatalogMatchesTheAppServer is the drift guard for the one
// thing All-In turned from a harmless inaccuracy into a user-visible defect.
//
// claudeconfig's openai-chatgpt Models list is written by hand from a probe,
// and the roster turns every entry into a picker row. The engine's allowlist,
// though, is whatever the RUNNING app-server reports — so an id the catalog
// keeps after Codex drops it is a row that resolves, reaches the engine, and
// answers 400. Nothing about that is visible offline: the build is green, the
// unit tests are green, and only the user finds it.
//
// It compares against includeHidden:false because that is exactly what
// StartAppServer asks for, so the two lists are the same question. It costs one
// app-server start and no quota. Run it after a codex upgrade:
//
//	WISP_DECK_LIVE_CHATGPT_CATALOG_E2E=1 go test ./internal/gptbridge/ \
//	  -run TestLiveChatGPTCatalogMatchesTheAppServer -v
//
// A failure naming an extra id is the defect above. A failure naming a missing
// one is a model the user is paying for and cannot pick. Either way the fix is
// to re-probe and edit the catalog — but check the account's plan first: this
// list is per-subscription, and a narrower plan legitimately reports fewer.
func TestLiveChatGPTCatalogMatchesTheAppServer(t *testing.T) {
	if os.Getenv("WISP_DECK_LIVE_CHATGPT_CATALOG_E2E") == "" {
		t.Skip("set WISP_DECK_LIVE_CHATGPT_CATALOG_E2E=1 to start a real Codex app-server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultChatGPTBridgeStartupTimeout)
	defer cancel()
	server, err := StartAppServer(ctx, AppServerOptions{
		CodexPath: liveCodexPath(t), ClientVersion: "live-test",
	})
	if err != nil {
		t.Fatalf("start the app-server: %v", err)
	}
	t.Cleanup(func() {
		closeContext, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = server.Close(closeContext)
	})

	live := map[string]bool{}
	var liveIDs []string
	for _, model := range server.Models {
		if model.ID == "" {
			continue
		}
		live[model.ID] = true
		liveIDs = append(liveIDs, model.ID)
	}
	if len(liveIDs) == 0 {
		t.Fatal("the app-server reported no models at all")
	}

	var catalog []string
	for _, provider := range claudeconfig.Providers {
		if provider.Key != "openai-chatgpt" {
			continue
		}
		for _, model := range provider.Models {
			catalog = append(catalog, model.ID)
		}
	}
	if len(catalog) == 0 {
		t.Fatal("claudeconfig has no openai-chatgpt models")
	}
	t.Logf("live: %v\ncatalog: %v", liveIDs, catalog)

	inCatalog := map[string]bool{}
	for _, id := range catalog {
		inCatalog[id] = true
		if !live[id] {
			t.Errorf("catalog offers %q, which this app-server does not serve — "+
				"All-In turns it into a picker row that resolves and then 400s", id)
		}
	}
	for _, id := range liveIDs {
		if !inCatalog[id] {
			t.Errorf("the app-server serves %q and the catalog does not offer it — "+
				"a model this subscription pays for that no picker row can reach", id)
		}
	}
}
