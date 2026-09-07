package allin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeResolver struct {
	credential Credential
	err        error
}

func (f fakeResolver) Resolve(Target) (Credential, error) { return f.credential, f.err }

func TestHandler_swaps_the_credential_and_strips_the_row_prefix(t *testing.T) {
	var gotAuth, gotBeta string
	var gotBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotBeta = r.Header.Get("Anthropic-Beta")
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	handler := NewHandler(fakeResolver{credential: Credential{
		BaseURL: upstream.URL, Header: "Authorization", Value: "Bearer swapped",
	}}, "http://unused.invalid")

	request := httptest.NewRequest(http.MethodPost, "/v1/messages",
		strings.NewReader(`{"model":"wisp/acct.personal/claude-opus-5[1m]"}`))
	request.Header.Set("Authorization", "Bearer session")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotAuth != "Bearer swapped" {
		t.Fatalf("auth %q", gotAuth)
	}
	if !strings.Contains(gotBeta, "context-1m-2025-08-07") {
		t.Fatalf("beta %q", gotBeta)
	}
	var sent map[string]any
	_ = json.Unmarshal(gotBody, &sent)
	if sent["model"] != "claude-opus-5" {
		t.Fatalf("model %v", sent["model"])
	}
}

func TestHandler_leaves_a_session_row_untouched(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	handler := NewHandler(fakeResolver{err: errors.New("must not be called")}, upstream.URL)
	request := httptest.NewRequest(http.MethodPost, "/v1/messages",
		strings.NewReader(`{"model":"claude-opus-5"}`))
	request.Header.Set("Authorization", "Bearer session")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotAuth != "Bearer session" {
		t.Fatalf("auth %q", gotAuth)
	}
}

func TestHandler_reports_a_stale_account_as_400_not_401(t *testing.T) {
	handler := NewHandler(fakeResolver{err: ErrStaleAccount}, "http://unused.invalid")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/messages",
		strings.NewReader(`{"model":"wisp/acct.personal/claude-opus-5"}`)))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "personal") {
		t.Fatalf("body does not name the account: %s", recorder.Body.String())
	}
	var envelope struct {
		Type  string `json:"type"`
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("body is not JSON: %v (%s)", err, recorder.Body.String())
	}
	if envelope.Type != "error" || envelope.Error.Type != "invalid_request_error" {
		t.Fatalf("envelope %+v", envelope)
	}
}

func TestHandler_routes_a_count_tokens_request_the_same_way(t *testing.T) {
	var gotPath, gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	handler := NewHandler(fakeResolver{credential: Credential{
		BaseURL: upstream.URL, Header: "Authorization", Value: "Bearer swapped",
	}}, "http://unused.invalid")
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost,
		"/v1/messages/count_tokens", strings.NewReader(`{"model":"wisp/acct.personal/claude-opus-5"}`)))

	if gotPath != "/v1/messages/count_tokens" || gotAuth != "Bearer swapped" {
		t.Fatalf("path %q auth %q", gotPath, gotAuth)
	}
}

func TestHandler_rejects_a_body_over_the_routing_cap(t *testing.T) {
	reached := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	handler := NewHandler(fakeResolver{err: errors.New("must not be called")}, upstream.URL)
	oversized := strings.Repeat("a", maxRouteBytes+1)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/messages",
		strings.NewReader(oversized)))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status %d", recorder.Code)
	}
	if reached {
		t.Fatal("upstream must not be reached for a body over the routing cap")
	}
}

func TestHandler_routes_correctly_when_content_length_is_unknown(t *testing.T) {
	var gotAuth string
	var gotBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	handler := NewHandler(fakeResolver{credential: Credential{
		BaseURL: upstream.URL, Header: "Authorization", Value: "Bearer swapped",
	}}, "http://unused.invalid")

	request := httptest.NewRequest(http.MethodPost, "/v1/messages",
		strings.NewReader(`{"model":"wisp/acct.personal/claude-opus-5"}`))
	// A chunked request declares no length up front; ContentLength reads -1.
	request.ContentLength = -1
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotAuth != "Bearer swapped" {
		t.Fatalf("auth %q", gotAuth)
	}
	var sent map[string]any
	_ = json.Unmarshal(gotBody, &sent)
	if sent["model"] != "claude-opus-5" {
		t.Fatalf("model %v, body %s", sent["model"], gotBody)
	}
}

func TestHandler_reports_a_malformed_upstream_as_400_not_502(t *testing.T) {
	handler := NewHandler(fakeResolver{credential: Credential{
		BaseURL: "http://exa mple.com", Header: "Authorization", Value: "Bearer x",
	}}, "http://unused.invalid")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/messages",
		strings.NewReader(`{"model":"wisp/acct.personal/claude-opus-5"}`)))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status %d, body %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandler_clears_both_credential_headers_before_swapping_in_one(t *testing.T) {
	// credential.Header == "Authorization" already deleted X-Api-Key on its
	// own, so the defect only shows through a credential that swaps in the
	// OTHER header: does the session's own Authorization survive and leak
	// upstream alongside it?
	var gotAuth, gotKey string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotKey = r.Header.Get("X-Api-Key")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	handler := NewHandler(fakeResolver{credential: Credential{
		BaseURL: upstream.URL, Header: "X-Api-Key", Value: "swapped-key",
	}}, "http://unused.invalid")

	request := httptest.NewRequest(http.MethodPost, "/v1/messages",
		strings.NewReader(`{"model":"wisp/acct.personal/claude-opus-5"}`))
	request.Header.Set("Authorization", "Bearer session")
	request.Header.Set("X-Api-Key", "session-key")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotKey != "swapped-key" {
		t.Fatalf("x-api-key %q", gotKey)
	}
	if gotAuth != "" {
		t.Fatalf("authorization leaked: %q", gotAuth)
	}
}

func TestHandler_streams_the_first_chunk_before_the_second_is_sent(t *testing.T) {
	// httputil.ReverseProxy hardcodes immediate flushing whenever
	// ContentLength == -1 or Content-Type is text/event-stream, regardless of
	// the configured FlushInterval — so only a response that declares a
	// Content-Length and isn't SSE actually exercises FlushInterval: -1.
	release := make(chan struct{})
	var closeOnce sync.Once
	closeRelease := func() { closeOnce.Do(func() { close(release) }) }
	t.Cleanup(closeRelease)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "11")
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte("first"))
		flusher.Flush()
		select {
		case <-release:
		case <-time.After(2 * time.Second):
		}
		_, _ = w.Write([]byte("second"))
	}))
	defer upstream.Close()

	handler := NewHandler(fakeResolver{credential: Credential{
		BaseURL: upstream.URL, Header: "Authorization", Value: "Bearer swapped",
	}}, "http://unused.invalid")
	proxy := httptest.NewServer(handler)
	defer proxy.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		resp, err := http.Post(proxy.URL+"/v1/messages", "application/json",
			strings.NewReader(`{"model":"wisp/acct.personal/claude-opus-5"}`))
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		buf := make([]byte, 5)
		if _, err := io.ReadFull(resp.Body, buf); err != nil {
			t.Error(err)
			return
		}
		if string(buf) != "first" {
			t.Errorf("chunk %q", buf)
		}
	}()

	select {
	case <-done:
		closeRelease()
	case <-time.After(1 * time.Second):
		closeRelease()
		<-done // let the goroutine finish before this test function returns
		t.Fatal("first chunk was not observed before the upstream sent its second write")
	}
}
