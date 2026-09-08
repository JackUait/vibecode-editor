package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The indicator must count mappings against the NAMED provider's model list, not
// the union of all providers' models — otherwise a value that belongs to a
// different provider is mis-counted as a valid mapping for this config.
func TestConfigAPIKeyIndicator_scopes_to_named_provider(t *testing.T) {
	dir := t.TempDir()

	// A MiMo-named config whose env points at a GLM model (not a MiMo model).
	// Scoped to the MiMo model list, that value is not a valid mapping.
	os.WriteFile(filepath.Join(dir, "x.json"),
		[]byte(`{"env":{"ANTHROPIC_DEFAULT_OPUS_MODEL":"glm-4.6"}}`), 0644)
	if got := configAPIKeyIndicator(dir, "x.json", "Xiaomi MiMo"); got != "unmapped" {
		t.Errorf("mimo config mapped to a glm model: got %q, want %q", got, "unmapped")
	}

	// A GLM-named config mapping two of its own provider's models -> "2 mapped".
	os.WriteFile(filepath.Join(dir, "g.json"),
		[]byte(`{"env":{"ANTHROPIC_DEFAULT_OPUS_MODEL":"glm-4.6","ANTHROPIC_DEFAULT_SONNET_MODEL":"glm-5"}}`), 0644)
	if got := configAPIKeyIndicator(dir, "g.json", "Work GLM"); got != "2 mapped" {
		t.Errorf("glm config with two mapped models: got %q, want %q", got, "2 mapped")
	}

	// Empty config -> "unmapped".
	os.WriteFile(filepath.Join(dir, "e.json"), []byte(`{}`), 0644)
	if got := configAPIKeyIndicator(dir, "e.json", "Work GLM"); got != "unmapped" {
		t.Errorf("empty config: got %q, want %q", got, "unmapped")
	}
}

// All-In routes by picker row and deliberately writes none of the four alias
// keys, so its count is always zero and "unmapped" was permanent. It says
// nothing instead. Every other profile keeps the word — there an empty count is
// a real state the user can act on.
func TestConfigAPIKeyIndicator_says_nothing_for_the_all_in_profile(t *testing.T) {
	dir := t.TempDir()

	// Resolved by its provider marker, not by its name.
	os.WriteFile(filepath.Join(dir, "all-in.json"),
		[]byte(`{"env":{"WISP_DECK_SUBSCRIPTION_PROVIDER":"allin"}}`), 0644)
	if got := configAPIKeyIndicator(dir, "all-in.json", "All-In"); got != "" {
		t.Errorf("All-In: got %q, want no indicator", got)
	}

	// A same-shaped profile that is not All-In still reports the empty count.
	os.WriteFile(filepath.Join(dir, "e.json"), []byte(`{}`), 0644)
	if got := configAPIKeyIndicator(dir, "e.json", "Work GLM"); got != "unmapped" {
		t.Errorf("a non-All-In profile mapping nothing: got %q, want %q", got, "unmapped")
	}
}

// A dropped indicator must take its separator space with it. The Subscription
// row right-aligns its state, so a lone styled space would shift the name one
// column left of where every other row's state sits.
func TestSettingsSubscriptionRow_drops_the_separator_with_the_indicator(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "all-in.json"),
		[]byte(`{"env":{"WISP_DECK_SUBSCRIPTION_PROVIDER":"allin"}}`), 0644)

	m := subTestMenu("claude")
	m.claudeConfigsDir = dir
	m.claudeConfigs = []ClaudeConfig{{Name: "All-In", File: "all-in.json"}}
	m.selectedConfig = 1
	m.SetActiveTab(TabSettings)

	out := stripAnsi(m.renderSettingsBox())
	if !strings.Contains(out, "[All-In]") {
		t.Fatalf("the Subscription row lost its state:\n%s", out)
	}
	if strings.Contains(out, "unmapped") {
		t.Errorf("the Subscription row still labels All-In unmapped:\n%s", out)
	}

	// Right-aligned means every row's state ends in the same column. A leftover
	// separator space pushes this one in by exactly one.
	subscription := settingsRowEndingColumn(t, out, "Subscription")
	neighbour := settingsRowEndingColumn(t, out, "Auto-switch accounts")
	if subscription != neighbour {
		t.Errorf("Subscription state ends at column %d, its neighbour at %d:\n%s",
			subscription, neighbour, out)
	}
}

// settingsRowEndingColumn reports where the named settings row's bracketed state
// ends.
func settingsRowEndingColumn(t *testing.T, box, label string) int {
	t.Helper()
	for _, line := range strings.Split(box, "\n") {
		if strings.Contains(line, label) {
			return strings.LastIndex(line, "]")
		}
	}
	t.Fatalf("settings box has no %q row:\n%s", label, box)
	return 0
}
