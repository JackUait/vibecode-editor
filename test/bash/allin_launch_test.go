package bash_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// A settings file the All-In profile wrote carries wisp/ picker rows — that is
// the only signal the launch chain has that a profile needs the router, since
// the display name can be renamed out from under it.
const allInPickerRow = `{"modelPicker":{"options":[{"model":"wisp/acct.default/claude-opus-5"}]}}`

const ordinaryProfile = `{"env":{"ANTHROPIC_MODEL":"claude-sonnet-5"}}`

func TestClaudeLaunchWrapper_routes_a_wisp_picker_profile_through_the_router(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeTempFile(t, dir, "overlay.json", allInPickerRow)

	out, code := runBashFunc(t, "lib/tmux-session.sh", "gt_claude_launch_wrapper",
		[]string{settingsPath, ""}, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "claude-allin")
	assertNotContains(t, out, "claude-rolefix")
}

func TestClaudeLaunchWrapper_keeps_featherless_on_the_role_repair_proxy(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeTempFile(t, dir, "overlay.json", ordinaryProfile)

	out, code := runBashFunc(t, "lib/tmux-session.sh", "gt_claude_launch_wrapper",
		[]string{settingsPath, "featherless"}, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "claude-rolefix")
	assertNotContains(t, out, "claude-allin")
}

func TestClaudeLaunchWrapper_wraps_nothing_for_an_ordinary_profile(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeTempFile(t, dir, "overlay.json", ordinaryProfile)

	out, code := runBashFunc(t, "lib/tmux-session.sh", "gt_claude_launch_wrapper",
		[]string{settingsPath, ""}, nil)
	assertExitCode(t, code, 0)
	assertNotContains(t, out, "claude-allin")
	assertNotContains(t, out, "claude-rolefix")
}

// The printed argv embeds the config root's four roster paths and the
// settings path via %q, exactly like every other launch-chain quoting site.
// A directory containing a space must survive as ONE argument, not split.
func TestClaudeLaunchWrapper_quotes_a_settings_path_containing_a_space(t *testing.T) {
	dir := t.TempDir()
	spacedDir := filepath.Join(dir, "has space")
	settingsPath := writeTempFile(t, spacedDir, "overlay.json", allInPickerRow)

	out, code := runBashFunc(t, "lib/tmux-session.sh", "gt_claude_launch_wrapper",
		[]string{settingsPath, ""}, nil)
	assertExitCode(t, code, 0)

	// Reconstruct the argv the way a real shell would (this is exactly what
	// happens at the launch-chain call site: the printed prefix is spliced
	// into a larger command line and re-parsed by bash). If the space were
	// not escaped, the settings path would split into two arguments and this
	// exact string would never appear as one token.
	script := "capture() { printf '%s\\n' \"$@\"; }\ncapture " + strings.TrimSpace(out)
	argvOut, argvCode := runBashSnippet(t, script, nil)
	assertExitCode(t, argvCode, 0)
	assertContains(t, argvOut, settingsPath)
}

// The config root itself can contain a space too (WISP_DECK_CONFIG_DIR is
// user-controlled); every roster path built from it must survive the same way.
func TestClaudeLaunchWrapper_quotes_a_config_root_containing_a_space(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeTempFile(t, dir, "overlay.json", allInPickerRow)
	configRoot := filepath.Join(dir, "config root")

	env := buildEnv(t, nil, "WISP_DECK_CONFIG_DIR="+configRoot)
	out, code := runBashFunc(t, "lib/tmux-session.sh", "gt_claude_launch_wrapper",
		[]string{settingsPath, ""}, env)
	assertExitCode(t, code, 0)

	script := "capture() { printf '%s\\n' \"$@\"; }\ncapture " + strings.TrimSpace(out)
	argvOut, argvCode := runBashSnippet(t, script, nil)
	assertExitCode(t, argvCode, 0)
	assertContains(t, argvOut, filepath.Join(configRoot, "claude-accounts.list"))
	assertContains(t, argvOut, filepath.Join(configRoot, "claude-accounts"))
	assertContains(t, argvOut, filepath.Join(configRoot, "claude-configs.list"))
	assertContains(t, argvOut, filepath.Join(configRoot, "claude-configs"))
}
