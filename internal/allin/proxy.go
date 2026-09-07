package allin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// maxRouteBytes caps the body read to re-address a request. A larger one is
// forwarded unread on the session's own credential rather than truncated.
const maxRouteBytes = 8 << 20

// NewHandler routes each request by the model its body names. A row this build
// cannot place goes to sessionUpstream on the session's own credential, so an
// unrecognised id costs a turn nothing.
func NewHandler(resolver Resolver, sessionUpstream string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base, body := sessionUpstream, []byte(nil)
		if r.Method == http.MethodPost && r.Body != nil && r.ContentLength <= maxRouteBytes {
			raw, err := io.ReadAll(io.LimitReader(r.Body, maxRouteBytes))
			_ = r.Body.Close()
			if err == nil {
				body = raw
			}
		}

		var payload map[string]any
		if body != nil {
			_ = json.Unmarshal(body, &payload)
		}
		model, _ := payload["model"].(string)
		target := Route(model)

		// KindSession never calls Resolve: Resolve's source guard rejects an
		// empty Source, and a session row always has one.
		if target.Kind != KindSession {
			credential, err := resolver.Resolve(target)
			if err != nil {
				writeRoutingError(w, target, err)
				return
			}
			base = credential.BaseURL
			payload["model"] = target.Model
			if rewritten, err := json.Marshal(payload); err == nil {
				body = rewritten
			}
			r.Header.Set(credential.Header, credential.Value)
			if credential.Header == "Authorization" {
				r.Header.Del("X-Api-Key")
			}
			if target.Want1M {
				addBeta(r.Header, "context-1m-2025-08-07")
			}
		}

		if body != nil {
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
			r.Header.Set("Content-Length", fmt.Sprint(len(body)))
		}
		newReverseProxy(base).ServeHTTP(w, r)
	})
}

func newReverseProxy(upstream string) *httputil.ReverseProxy {
	target, err := url.Parse(upstream)
	if err != nil {
		return &httputil.ReverseProxy{Director: func(*http.Request) {}}
	}
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme, req.URL.Host = target.Scheme, target.Host
			req.URL.Path = strings.TrimSuffix(target.Path, "/") + req.URL.Path
			// Gateways route on Host; the loopback name reaches no virtual host.
			req.Host = target.Host
			// A compressed body is bytes the next hop cannot read.
			req.Header.Set("Accept-Encoding", "identity")
		},
		// Each write forwarded as it arrives: buffering swallows the keep-alive
		// bytes that keep Claude Code's stall watchdog from replaying a turn.
		FlushInterval: -1,
	}
}

func addBeta(header http.Header, value string) {
	current := header.Get("Anthropic-Beta")
	if strings.Contains(current, value) {
		return
	}
	if current == "" {
		header.Set("Anthropic-Beta", value)
		return
	}
	header.Set("Anthropic-Beta", current+","+value)
}

// writeRoutingError answers in Anthropic's own envelope, always with 400.
// Claude Code retries 401 and 5xx about eleven times, and every one of these is
// deterministic — the same request would fail again the same way.
func writeRoutingError(w http.ResponseWriter, target Target, err error) {
	message := fmt.Sprintf("wisp-deck: %v", err)
	if errors.Is(err, ErrStaleAccount) {
		message = fmt.Sprintf(
			"wisp-deck: the login %q has no usable credential. Open it once "+
				"(a wisp-deck tab on that account) so Claude refreshes its token, then retry.",
			target.Source)
	}
	body, _ := json.Marshal(map[string]any{
		"type":  "error",
		"error": map[string]string{"type": "invalid_request_error", "message": message},
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write(body)
}
