package subusage

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AnthropicBaseURL is the root FetchClaudeAccount queries; tests substitute an
// httptest server.
const AnthropicBaseURL = "https://api.anthropic.com"

// FetchClaudeAccount reads one Claude login's real 5-hour and 7-day usage.
//
// This is the same reading Claude Code puts in its statusline rate_limits for
// the session's OWN login. The All-In picker needs every login's, including the
// ones no pane is running, so it asks the endpoint directly with that login's
// Keychain OAuth token. Verified live on 2026-09-08 against two logins: HTTP
// 200 with utilization percentages and ISO-8601 reset times.
func FetchClaudeAccount(client *http.Client, baseURL, token string) (RateLimits, error) {
	if token == "" {
		return RateLimits{}, errors.New("claude account usage: no token")
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/oauth/usage", nil)
	if err != nil {
		return RateLimits{}, err
	}
	// The bearer is the OAuth token, not an API key: this route authenticates
	// the login, and the beta header is what admits an OAuth caller.
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("User-Agent", "wisp-deck")
	resp, err := client.Do(req)
	if err != nil {
		return RateLimits{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return RateLimits{}, fmt.Errorf("claude account usage endpoint: HTTP %d", resp.StatusCode)
	}
	var payload struct {
		FiveHour *oauthWindow `json:"five_hour"`
		SevenDay *oauthWindow `json:"seven_day"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return RateLimits{}, err
	}
	return RateLimits{
		FiveHour: mapOAuthWindow(payload.FiveHour),
		SevenDay: mapOAuthWindow(payload.SevenDay),
	}, nil
}

// oauthWindow is one window as the OAuth usage route reports it.
type oauthWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

func mapOAuthWindow(w *oauthWindow) *Window {
	if w == nil {
		return nil
	}
	mapped := &Window{UsedPercentage: w.Utilization}
	// resets_at is RFC 3339 with microseconds and a numeric offset. An absent
	// one is a real reading of a window with no scheduled reset, not a parse
	// failure, so it keeps the utilization and drops only the time.
	if reset, err := time.Parse(time.RFC3339Nano, w.ResetsAt); err == nil {
		mapped.ResetAt = reset.Unix()
	}
	return mapped
}
