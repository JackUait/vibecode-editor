package subusage

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Captured verbatim from api.anthropic.com on 2026-09-08 for a live login, so
// the parse is pinned to the shape the endpoint actually serves.
const liveOAuthUsage = `{"five_hour":{"utilization":24.0,"resets_at":"2026-09-08T12:19:59.807128+00:00","limit_dollars":null,"used_dollars":null,"remaining_dollars":null,"locked_reason":null},"seven_day":{"utilization":40.0,"resets_at":"2026-09-14T10:59:59.807146+00:00","limit_dollars":null,"used_dollars":null,"remaining_dollars":null,"locked_reason":null},"seven_day_oauth_apps":null,"seven_day_opus":null,"nimbus_quill":{"utilization":0.0,"resets_at":null,"limit_dollars":null,"used_dollars":null,"remaining_dollars":null,"locked_reason":null},"extra_usage":{"is_enabled":false,"monthly_limit":null,"used_credits":null,"utilization":null},"limits":[{"kind":"session","group":"session","percent":24,"severity":"normal","resets_at":"2026-09-08T12:19:59.807128+00:00","scope":null,"is_active":false},{"kind":"weekly_all","group":"weekly","percent":40,"severity":"normal","resets_at":"2026-09-14T10:59:59.807146+00:00","scope":null,"is_active":true}],"member_dashboard_available":false}`

func TestFetchClaudeAccount_reads_both_windows_of_a_live_payload(t *testing.T) {
	var gotAuth, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		_, _ = w.Write([]byte(liveOAuthUsage))
	}))
	defer server.Close()

	rl, err := FetchClaudeAccount(server.Client(), server.URL, "tok-123")
	if err != nil {
		t.Fatal(err)
	}
	// The endpoint authenticates the OAuth bearer; x-api-key answers 401.
	if gotAuth != "Bearer tok-123" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotPath != "/api/oauth/usage" {
		t.Fatalf("path = %q", gotPath)
	}
	if rl.FiveHour == nil || rl.FiveHour.UsedPercentage != 24 {
		t.Fatalf("five hour = %+v, want 24%% used", rl.FiveHour)
	}
	if rl.SevenDay == nil || rl.SevenDay.UsedPercentage != 40 {
		t.Fatalf("seven day = %+v, want 40%% used", rl.SevenDay)
	}
	want, err := time.Parse(time.RFC3339Nano, "2026-09-08T12:19:59.807128+00:00")
	if err != nil {
		t.Fatal(err)
	}
	if rl.FiveHour.ResetAt != want.Unix() {
		t.Fatalf("reset = %d, want %d", rl.FiveHour.ResetAt, want.Unix())
	}
}

// A plan with no session window reports the key as null rather than omitting
// it, and a window with no reset time is still a real utilization reading.
func TestFetchClaudeAccount_tolerates_a_null_window_and_a_null_reset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"five_hour":null,"seven_day":{"utilization":90.0,"resets_at":null}}`))
	}))
	defer server.Close()

	rl, err := FetchClaudeAccount(server.Client(), server.URL, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if rl.FiveHour != nil {
		t.Fatalf("a null window must stay nil, got %+v", rl.FiveHour)
	}
	if rl.SevenDay == nil || rl.SevenDay.UsedPercentage != 90 || rl.SevenDay.ResetAt != 0 {
		t.Fatalf("seven day = %+v", rl.SevenDay)
	}
}

func TestFetchClaudeAccount_reports_a_refused_token(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	if _, err := FetchClaudeAccount(server.Client(), server.URL, "stale"); err == nil {
		t.Fatal("a 401 must be an error, not an empty reading")
	}
	if _, err := FetchClaudeAccount(server.Client(), server.URL, ""); err == nil {
		t.Fatal("no token must be an error before any request")
	}
}
