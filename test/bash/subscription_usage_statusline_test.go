package bash_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- gt_sub_usage_fresh ---
// The freshness gate for the subscription usage cache: only a snapshot whose
// fetched_at is within max_age of now may drive the statusline bars — stale
// data hides rather than lies.

func TestSubUsage_fresh_accepts_recent_fetched_at(t *testing.T) {
	json := `{"rate_limits":{"seven_day":{"used_percentage":70}},"fetched_at":1000}`
	_, code := runBashFunc(t, "lib/statusline.sh", "gt_sub_usage_fresh",
		[]string{json, "1500"}, nil)
	assertExitCode(t, code, 0)
}

func TestSubUsage_fresh_rejects_stale_fetched_at(t *testing.T) {
	json := `{"rate_limits":{},"fetched_at":1000}`
	_, code := runBashFunc(t, "lib/statusline.sh", "gt_sub_usage_fresh",
		[]string{json, "20000"}, nil)
	if code == 0 {
		t.Fatal("a snapshot older than the default max age must not be fresh")
	}
}

func TestSubUsage_fresh_rejects_missing_fetched_at(t *testing.T) {
	_, code := runBashFunc(t, "lib/statusline.sh", "gt_sub_usage_fresh",
		[]string{`{"rate_limits":{}}`, "100"}, nil)
	if code == 0 {
		t.Fatal("a snapshot without fetched_at must not be fresh")
	}
}

func TestSubUsage_fresh_honors_custom_max_age(t *testing.T) {
	json := `{"fetched_at":1000}`
	_, code := runBashFunc(t, "lib/statusline.sh", "gt_sub_usage_fresh",
		[]string{json, "1200", "100"}, nil)
	if code == 0 {
		t.Fatal("snapshot outside the custom max age must not be fresh")
	}
}

// The Go cache serializes claude's rate_limits shape precisely so the existing
// extractors parse it — guard that contract from the bash side too.
func TestSubUsage_cache_shape_parses_with_existing_extractors(t *testing.T) {
	cache := `{"rate_limits":{"five_hour":{"used_percentage":55},"seven_day":{"used_percentage":81}},"provider":"zhipu","fetched_at":123}`
	out, code := runBashFunc(t, "lib/statusline.sh", "gt_five_hour_used_pct", []string{cache}, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "55" {
		t.Fatalf("five_hour pct = %q, want 55", strings.TrimSpace(out))
	}
	out, code = runBashFunc(t, "lib/statusline.sh", "gt_weekly_used_pct", []string{cache}, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "81" {
		t.Fatalf("weekly pct = %q, want 81", strings.TrimSpace(out))
	}
}

// --- statusline-wrapper: subscription panes show the SUBSCRIPTION's usage ---

// seedSubUsageCache writes a subscription usage snapshot where the wrapper
// looks for it (XDG root/wisp-deck/subscription-usage/<config>).
func seedSubUsageCache(t *testing.T, fakeHome, config, json string) {
	t.Helper()
	cfg := filepath.Join(fakeHome, ".config", "wisp-deck")
	writeTempFile(t, filepath.Join(cfg, "subscription-usage"), config, json)
}

func runWrapperWithInput(t *testing.T, env []string, stdinData string) (string, int) {
	t.Helper()
	root := projectRoot(t)
	wrapperPath := filepath.Join(root, "templates", "statusline-wrapper.sh")
	script := fmt.Sprintf(`echo '%s' | bash '%s'`, stdinData, wrapperPath)
	return runBashSnippet(t, script, env)
}

// A subscription pane renders the bars from the subscription's own cached
// usage — even when the pane has a single (Default) login, where the account
// segment is ineligible.
func TestSubUsage_wrapper_bars_come_from_subscription_cache(t *testing.T) {
	env := setupWrapperTest(t)
	env = append(env, "CLAUDE_CONFIG_DIR=", "WISP_DECK_CLAUDE_CONFIG=glm.json")
	fakeHome := wrapperHome(env)
	cfg := filepath.Join(fakeHome, ".config", "wisp-deck")
	writeTempFile(t, cfg, "settings", "usage_bars=both\n")
	writeTempFile(t, cfg, "claude-config-colors", "glm.json:205\n")
	now := time.Now().Unix()
	seedSubUsageCache(t, fakeHome, "glm.json", fmt.Sprintf(
		`{"rate_limits":{"five_hour":{"used_percentage":50},"seven_day":{"used_percentage":90}},"provider":"zhipu","fetched_at":%d}`, now))

	out, code := runWrapperWithInput(t, env,
		`{"model":{"id":"claude-fable-5","display_name":"Fable 5"},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)
	assertContains(t, out, bar(9)) // 7d: 90%
	assertContains(t, out, bar(5)) // 5h: 50%
	// Painted in the subscription's color.
	assertContains(t, out, "\x1b[38;5;205m"+bar(9))
}

// Native rate_limits belong to the LOGIN, not the subscription — on a
// subscription pane they must never drive the bars.
func TestSubUsage_wrapper_ignores_native_rate_limits_on_subscription_pane(t *testing.T) {
	env := setupWrapperTest(t)
	env = append(env, "CLAUDE_CONFIG_DIR=", "WISP_DECK_CLAUDE_CONFIG=glm.json")
	fakeHome := wrapperHome(env)
	cfg := filepath.Join(fakeHome, ".config", "wisp-deck")
	writeTempFile(t, cfg, "settings", "usage_bars=7d\n")
	now := time.Now().Unix()
	seedSubUsageCache(t, fakeHome, "glm.json", fmt.Sprintf(
		`{"rate_limits":{"seven_day":{"used_percentage":20}},"provider":"zhipu","fetched_at":%d}`, now))

	out, code := runWrapperWithInput(t, env,
		`{"model":{"id":"claude-fable-5","display_name":"Fable 5"},"rate_limits":{"seven_day":{"used_percentage":90,"resets_at":2}},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)
	assertContains(t, out, bar(2))    // the subscription's 20%
	assertNotContains(t, out, bar(9)) // never the login's 90%
}

// A stale cache hides the figures: the bar shows the "…" placeholder instead
// of yesterday's percentages (and never the login's native ones).
func TestSubUsage_wrapper_stale_cache_shows_placeholder(t *testing.T) {
	env := setupWrapperTest(t)
	env = append(env, "CLAUDE_CONFIG_DIR=", "WISP_DECK_CLAUDE_CONFIG=glm.json")
	fakeHome := wrapperHome(env)
	cfg := filepath.Join(fakeHome, ".config", "wisp-deck")
	writeTempFile(t, cfg, "settings", "usage_bars=7d\n")
	stale := time.Now().Unix() - 100000
	seedSubUsageCache(t, fakeHome, "glm.json", fmt.Sprintf(
		`{"rate_limits":{"seven_day":{"used_percentage":90}},"provider":"zhipu","fetched_at":%d}`, stale))

	out, code := runWrapperWithInput(t, env,
		`{"model":{"id":"claude-fable-5","display_name":"Fable 5"},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)
	assertNotContains(t, out, bar(9))
	assertContains(t, out, "…")
}

// Only zhipu and openai-chatgpt publish a quota API; for every other provider
// (featherless, mimo, moonshot, moonshot-coding, custom) subusage.Fetch reports
// "no usage API" without an error, so the refresher writes a FRESH snapshot
// carrying no windows. The wrapper used to read that exactly like "the first
// fetch has not landed yet" and paint the "…" placeholder — permanently, since
// the answer never changes. A fresh snapshot is the provider's final word, so
// its silence hides the pill instead.
func TestSubUsage_wrapper_hides_the_pill_when_the_provider_publishes_no_quota(t *testing.T) {
	env := setupWrapperTest(t)
	env = append(env, "CLAUDE_CONFIG_DIR=", "WISP_DECK_CLAUDE_CONFIG=featherless.json")
	fakeHome := wrapperHome(env)
	cfg := filepath.Join(fakeHome, ".config", "wisp-deck")
	writeTempFile(t, cfg, "settings", "usage_bars=both\n")
	now := time.Now().Unix()
	seedSubUsageCache(t, fakeHome, "featherless.json", fmt.Sprintf(
		`{"rate_limits":{},"provider":"featherless","fetched_at":%d}`, now))

	out, code := runWrapperWithInput(t, env,
		`{"model":{"id":"claude-fable-5","display_name":"Fable 5"},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)
	assertNotContains(t, out, "7d")
	assertNotContains(t, out, "5h")
	assertNotContains(t, out, "…")
	// The whole usage group goes with it — no orphaned " |" separator.
	if strings.HasSuffix(strings.TrimSpace(out), "|") {
		t.Fatalf("usage group left a dangling separator: %q", out)
	}
}

// A provider can publish one window and not the other — a ChatGPT Pro account
// reports its weekly window as primary with no secondary. The window it does
// report still draws; only the absent one hides.
func TestSubUsage_wrapper_hides_only_the_window_the_provider_lacks(t *testing.T) {
	env := setupWrapperTest(t)
	env = append(env, "CLAUDE_CONFIG_DIR=", "WISP_DECK_CLAUDE_CONFIG=chatgpt.json")
	fakeHome := wrapperHome(env)
	cfg := filepath.Join(fakeHome, ".config", "wisp-deck")
	writeTempFile(t, cfg, "settings", "usage_bars=both\n")
	now := time.Now().Unix()
	seedSubUsageCache(t, fakeHome, "chatgpt.json", fmt.Sprintf(
		`{"rate_limits":{"seven_day":{"used_percentage":40}},"provider":"openai-chatgpt","fetched_at":%d}`, now))

	out, code := runWrapperWithInput(t, env,
		`{"model":{"id":"claude-fable-5","display_name":"Fable 5"},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)
	assertContains(t, out, "7d")
	assertContains(t, out, bar(4))
	assertNotContains(t, out, "5h")
	assertNotContains(t, out, "…")
}

// The wrapper keeps the cache alive: every render on a subscription pane
// spawns the (self-throttling) refresher with the pane's config.
func TestSubUsage_wrapper_spawns_refresher_for_subscription_pane(t *testing.T) {
	env := setupWrapperTest(t)
	env = append(env, "CLAUDE_CONFIG_DIR=", "WISP_DECK_CLAUDE_CONFIG=glm.json")
	fakeHome := wrapperHome(env)
	logFile := filepath.Join(fakeHome, "tui-calls.log")

	// The mock stands in for wisp-deck-tui and records its argv.
	dir := filepath.Dir(filepath.Dir(fakeHome))
	_ = dir
	mockDir := filepath.Join(fakeHome, "..")
	binDir := mockCommand(t, mockDir, "wisp-deck-tui", fmt.Sprintf(`printf '%%s\n' "$*" >> %q`, logFile))
	env = prependPath(env, binDir)

	_, code := runWrapperWithInput(t, env,
		`{"model":{"id":"claude-fable-5","display_name":"Fable 5"},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)

	var recorded string
	for i := 0; i < 40; i++ { // the refresher is backgrounded; give it a beat
		if data, err := os.ReadFile(logFile); err == nil && len(data) > 0 {
			recorded = string(data)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(recorded, "subscription-usage") {
		t.Fatalf("refresher not spawned; recorded calls: %q", recorded)
	}
	if !strings.Contains(recorded, "--config glm.json") {
		t.Fatalf("refresher missing the pane's config: %q", recorded)
	}
}

// A standard-Claude pane must not spawn the refresher at all.
func TestSubUsage_wrapper_no_refresher_without_subscription(t *testing.T) {
	env := setupWrapperTest(t)
	env = append(env, "CLAUDE_CONFIG_DIR=", "WISP_DECK_CLAUDE_CONFIG=")
	fakeHome := wrapperHome(env)
	logFile := filepath.Join(fakeHome, "tui-calls.log")
	mockDir := filepath.Join(fakeHome, "..")
	binDir := mockCommand(t, mockDir, "wisp-deck-tui", fmt.Sprintf(`printf '%%s\n' "$*" >> %q`, logFile))
	env = prependPath(env, binDir)

	_, code := runWrapperWithInput(t, env,
		`{"model":{"id":"claude-fable-5","display_name":"Fable 5"},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)
	time.Sleep(200 * time.Millisecond)
	if data, err := os.ReadFile(logFile); err == nil && strings.Contains(string(data), "subscription-usage") {
		t.Fatalf("refresher spawned on a standard pane: %q", string(data))
	}
}

// prependPath returns env with dir prepended to its PATH entry.
func prependPath(env []string, dir string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			kv = "PATH=" + dir + ":" + strings.TrimPrefix(kv, "PATH=")
		}
		out = append(out, kv)
	}
	return out
}

// --- gt_allin_source ---
// Pulls the All-In router's picker-row source out of a raw model id, mirroring
// internal/allin/route.go's own grammar (wisp/acct.<dir>/<model> or
// wisp/cfg.<file>/<model>, cut on the first two "/"s only). A model id this
// build cannot place echoes nothing, the same "session" fallback Route uses.

func TestAllinSource_extracts_an_account_row(t *testing.T) {
	out, code := runBashFunc(t, "lib/statusline.sh", "gt_allin_source",
		[]string{"wisp/acct.personal/claude-opus-5"}, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "acct.personal" {
		t.Fatalf("expected %q, got %q", "acct.personal", strings.TrimSpace(out))
	}
}

func TestAllinSource_extracts_a_config_row(t *testing.T) {
	out, code := runBashFunc(t, "lib/statusline.sh", "gt_allin_source",
		[]string{"wisp/cfg.zhipu-glm/glm-5.2"}, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "cfg.zhipu-glm" {
		t.Fatalf("expected %q, got %q", "cfg.zhipu-glm", strings.TrimSpace(out))
	}
}

// A Featherless model id carries its own slash — the source must still be cut
// on the FIRST "/" after "wisp/", never on every "/" in the string.
func TestAllinSource_keeps_slashes_inside_the_model_id(t *testing.T) {
	out, code := runBashFunc(t, "lib/statusline.sh", "gt_allin_source",
		[]string{"wisp/cfg.featherless/TurboVadim/Qwen3.8-27B-OBLITERATED"}, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "cfg.featherless" {
		t.Fatalf("expected %q, got %q", "cfg.featherless", strings.TrimSpace(out))
	}
}

func TestAllinSource_empty_for_a_plain_model(t *testing.T) {
	out, code := runBashFunc(t, "lib/statusline.sh", "gt_allin_source",
		[]string{"claude-opus-5"}, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty output, got %q", strings.TrimSpace(out))
	}
}

func TestAllinSource_empty_for_an_unparseable_prefix(t *testing.T) {
	for _, id := range []string{"wisp/", "wisp/onlysource", "wisp/bogus.x/m"} {
		out, code := runBashFunc(t, "lib/statusline.sh", "gt_allin_source", []string{id}, nil)
		assertExitCode(t, code, 0)
		if strings.TrimSpace(out) != "" {
			t.Fatalf("%q: expected empty output, got %q", id, strings.TrimSpace(out))
		}
	}
}

// --- statusline-wrapper: All-In usage follows the picked row ---
// An All-In pane's own config file (WISP_DECK_CLAUDE_CONFIG=all-in.json) names
// the router, which has no quota of its own — the picker row in the model id
// names the account or profile actually serving the turn, and usage must
// follow THAT instead.

// setupAllinWrapperTest seeds a 2-login setup (so the native account segment is
// eligible) plus WISP_DECK_CLAUDE_CONFIG naming the All-In profile.
func setupAllinWrapperTest(t *testing.T) []string {
	t.Helper()
	env := setupWrapperTest(t)
	env = append(env, "CLAUDE_CONFIG_DIR=", "WISP_DECK_CLAUDE_CONFIG=all-in.json")
	fakeHome := wrapperHome(env)
	cfg := filepath.Join(fakeHome, ".config", "wisp-deck")
	writeTempFile(t, cfg, "claude-accounts.list", "Personal:personal\n")
	return env
}

// A wisp/acct.… row needs no fetch at all: Claude Code's own rate_limits
// already describe the login the router forwarded the turn to, so the
// subscription path must stand aside and let them through untouched.
func TestSubUsage_wrapper_allin_acct_row_leaves_native_bars_in_place(t *testing.T) {
	env := setupAllinWrapperTest(t)
	fakeHome := wrapperHome(env)
	logFile := filepath.Join(fakeHome, "tui-calls.log")
	mockDir := filepath.Join(fakeHome, "..")
	binDir := mockCommand(t, mockDir, "wisp-deck-tui", fmt.Sprintf(`printf '%%s\n' "$*" >> %q`, logFile))
	env = prependPath(env, binDir)

	out, code := runWrapperWithInput(t, env,
		`{"model":{"id":"wisp/acct.default/claude-opus-5","display_name":"Opus 5"},`+
			`"rate_limits":{"seven_day":{"used_percentage":90,"resets_at":2}},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)
	// The login's native 90% bar, untouched.
	assertContains(t, out, bar(9))

	time.Sleep(200 * time.Millisecond)
	if data, err := os.ReadFile(logFile); err == nil && strings.Contains(string(data), "subscription-usage") {
		t.Fatalf("an acct. row must fetch nothing; recorded calls: %q", string(data))
	}
}

// The native weekly/five-hour computation is normally gated on 2+ Claude
// logins ($account_label) — an unrelated, pre-existing eligibility rule for
// solo native users. An All-In profile needs only TWO sources total
// (SourceCount), so the common shape is one native login plus subscriptions:
// that machine's claude-accounts.list has zero MANAGED entries, so
// $account_label is empty even while the router is actively serving a
// wisp/acct. row. Without a widened gate the "stand aside" fix above regresses
// silently: the sub-usage block never runs (so it never hides the group) but
// the native block never fills the bar either, leaving the "…" placeholder
// forever instead of a number OR nothing.
func TestSubUsage_wrapper_allin_acct_row_native_bars_without_multiple_logins(t *testing.T) {
	env := setupWrapperTest(t) // no claude-accounts.list entries: a solo login
	env = append(env, "CLAUDE_CONFIG_DIR=", "WISP_DECK_CLAUDE_CONFIG=all-in.json")

	out, code := runWrapperWithInput(t, env,
		`{"model":{"id":"wisp/acct.default/claude-opus-5","display_name":"Opus 5"},`+
			`"rate_limits":{"seven_day":{"used_percentage":90,"resets_at":2}},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)
	assertContains(t, out, bar(9))
	assertNotContains(t, out, "…")
}

// A wisp/cfg.<profile>/… row fetches THAT profile's usage, keyed on its own
// filename — never the router's own all-in.json, which has no fetcher.
func TestSubUsage_wrapper_allin_cfg_row_fetches_the_routed_profile(t *testing.T) {
	env := setupAllinWrapperTest(t)
	fakeHome := wrapperHome(env)
	cfg := filepath.Join(fakeHome, ".config", "wisp-deck")
	logFile := filepath.Join(fakeHome, "tui-calls.log")
	mockDir := filepath.Join(fakeHome, "..")
	binDir := mockCommand(t, mockDir, "wisp-deck-tui", fmt.Sprintf(`printf '%%s\n' "$*" >> %q`, logFile))
	env = prependPath(env, binDir)
	writeTempFile(t, cfg, "settings", "usage_bars=both\n")
	now := time.Now().Unix()
	// The routed profile's own cache — this is what must drive the bars.
	seedSubUsageCache(t, fakeHome, "zhipu-glm.json", fmt.Sprintf(
		`{"rate_limits":{"five_hour":{"used_percentage":50},"seven_day":{"used_percentage":90}},"provider":"zhipu","fetched_at":%d}`, now))
	// A decoy snapshot under the router's OWN file: never read for a cfg. row.
	seedSubUsageCache(t, fakeHome, "all-in.json", fmt.Sprintf(
		`{"rate_limits":{"five_hour":{"used_percentage":11},"seven_day":{"used_percentage":22}},"provider":"allin","fetched_at":%d}`, now))

	out, code := runWrapperWithInput(t, env,
		`{"model":{"id":"wisp/cfg.zhipu-glm/glm-5.2","display_name":"GLM 5.2"},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)
	assertContains(t, out, bar(9)) // the routed profile's 90%, not all-in.json's 22%
	assertContains(t, out, bar(5))
	assertNotContains(t, out, bar(2)) // all-in.json's decoy 11%/22% never rendered

	var recorded string
	for i := 0; i < 40; i++ {
		if data, err := os.ReadFile(logFile); err == nil && len(data) > 0 {
			recorded = string(data)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(recorded, "--config zhipu-glm.json") {
		t.Fatalf("refresher must be keyed on the routed profile, recorded: %q", recorded)
	}
	if strings.Contains(recorded, "--config all-in.json") {
		t.Fatalf("refresher must never be keyed on the router's own file, recorded: %q", recorded)
	}
}

// A routed profile with no usage API (per Fetch's zhipu/openai-chatgpt-only
// switch) yields NO bars — not the "…" placeholder, and not a blank pair of
// bars — exactly like a dedicated pane on that same provider.
func TestSubUsage_wrapper_allin_cfg_row_with_no_fetcher_shows_no_bars(t *testing.T) {
	env := setupAllinWrapperTest(t)
	fakeHome := wrapperHome(env)
	cfg := filepath.Join(fakeHome, ".config", "wisp-deck")
	writeTempFile(t, cfg, "settings", "usage_bars=both\n")
	now := time.Now().Unix()
	// What the real refresher writes for a provider with no quota API: a FRESH
	// snapshot carrying no windows.
	seedSubUsageCache(t, fakeHome, "moonshot.json", fmt.Sprintf(
		`{"rate_limits":{},"provider":"moonshot","fetched_at":%d}`, now))

	out, code := runWrapperWithInput(t, env,
		`{"model":{"id":"wisp/cfg.moonshot/moonshot-v1","display_name":"Moonshot"},"workspace":{"current_dir":"/tmp"}}`)
	assertExitCode(t, code, 0)
	assertNotContains(t, out, "7d")
	assertNotContains(t, out, "5h")
	assertNotContains(t, out, "…")
	if strings.HasSuffix(strings.TrimSpace(out), "|") {
		t.Fatalf("usage group left a dangling separator: %q", out)
	}
}
