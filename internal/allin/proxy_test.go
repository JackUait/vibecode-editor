package allin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
