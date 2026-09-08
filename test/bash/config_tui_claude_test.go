package bash_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The dispatcher delegates mutations to `wisp-deck-tui claude-config <action>`
// (the single Go source of truth). This test mocks the binary and asserts the
// dispatcher invokes it with the right action and arguments.
func TestConfigMenu_dispatch_add_invokes_binary_then_quits(t *testing.T) {
	dir := t.TempDir()
	cfgRoot := filepath.Join(dir, "wisp-deck")
	_ = os.MkdirAll(cfgRoot, 0o755)
	calls := filepath.Join(dir, "calls.log")

	// Mock wisp-deck-tui: claude-config-menu returns add once then quit;
	// claude-config records its arguments.
	bin := mockCommand(t, dir, "wisp-deck-tui", `
state="`+dir+`/n"
n=$(cat "$state" 2>/dev/null || echo 0)
echo $((n+1)) > "$state"
case "$1" in
  claude-config-menu)
    if [ "$n" = "0" ]; then echo '{"action":"add","name":"Work"}'; else echo '{"action":"quit"}'; fi ;;
  claude-config)
    shift; echo "$@" >> "`+calls+`" ;;
  *) echo '{}' ;;
esac
`)
	env := buildEnv(t, []string{bin}, "XDG_CONFIG_HOME="+dir)

	root := projectRoot(t)
	script := `
source ` + root + `/lib/claude-configs.sh
source ` + root + `/lib/config-tui.sh
manage_claude_configs_interactive
`
	_, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)

	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatalf("binary was not invoked: %v", err)
	}
	got := string(data)
	for _, want := range []string{"add", "--list", "--dir", "--name", "Work"} {
		assertContains(t, got, want)
	}
}

// Without --accounts-list/--accounts-dir, ensure-allin's gate (shared by
// add/delete) can only ever see provider profiles, never logins — a machine
// with two Claude logins and zero providers would never cross the threshold
// through this menu. Both mutating actions must pass the same roots
// bin/wisp-deck already uses for ensure-allin.
func TestConfigMenu_dispatch_passes_accounts_paths_to_add_and_delete(t *testing.T) {
	dir := t.TempDir()
	cfgRoot := filepath.Join(dir, "wisp-deck")
	_ = os.MkdirAll(cfgRoot, 0o755)
	calls := filepath.Join(dir, "calls.log")

	bin := mockCommand(t, dir, "wisp-deck-tui", `
state="`+dir+`/n"
case "$1" in
  claude-config-menu)
    n=$(cat "$state" 2>/dev/null || echo 0)
    echo $((n+1)) > "$state"
    case "$n" in
      0) echo '{"action":"add","name":"Work"}' ;;
      1) echo '{"action":"delete","file":"work.json"}' ;;
      *) echo '{"action":"quit"}' ;;
    esac
    ;;
  claude-config)
    shift; echo "$@" >> "`+calls+`" ;;
  *) echo '{}' ;;
esac
`)
	env := buildEnv(t, []string{bin}, "XDG_CONFIG_HOME="+dir)

	root := projectRoot(t)
	script := `
source ` + root + `/lib/claude-configs.sh
source ` + root + `/lib/config-tui.sh
manage_claude_configs_interactive
`
	_, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)

	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatalf("binary was not invoked: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d claude-config calls, want 2: %q", len(lines), data)
	}
	for i, want := range [][]string{
		{"add", "--accounts-list", "--accounts-dir"},
		{"delete", "--accounts-list", "--accounts-dir"},
	} {
		for _, substr := range want {
			assertContains(t, lines[i], substr)
		}
	}
}

// Same gap as the accounts paths above, but for the Default login's own tag:
// without --default-label-file, a machine reached only through this legacy
// menu can never label the implicit login by anything but the literal word
// "Default", even though the user renamed it via the Subscriptions modal.
func TestConfigMenu_dispatch_passes_default_label_file_to_add_and_delete(t *testing.T) {
	dir := t.TempDir()
	cfgRoot := filepath.Join(dir, "wisp-deck")
	_ = os.MkdirAll(cfgRoot, 0o755)
	calls := filepath.Join(dir, "calls.log")

	bin := mockCommand(t, dir, "wisp-deck-tui", `
state="`+dir+`/n"
case "$1" in
  claude-config-menu)
    n=$(cat "$state" 2>/dev/null || echo 0)
    echo $((n+1)) > "$state"
    case "$n" in
      0) echo '{"action":"add","name":"Work"}' ;;
      1) echo '{"action":"delete","file":"work.json"}' ;;
      *) echo '{"action":"quit"}' ;;
    esac
    ;;
  claude-config)
    shift; echo "$@" >> "`+calls+`" ;;
  *) echo '{}' ;;
esac
`)
	env := buildEnv(t, []string{bin}, "XDG_CONFIG_HOME="+dir)

	root := projectRoot(t)
	script := `
source ` + root + `/lib/claude-configs.sh
source ` + root + `/lib/config-tui.sh
manage_claude_configs_interactive
`
	_, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)

	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatalf("binary was not invoked: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d claude-config calls, want 2: %q", len(lines), data)
	}
	for i, want := range [][]string{
		{"add", "--default-label-file"},
		{"delete", "--default-label-file"},
	} {
		for _, substr := range want {
			assertContains(t, lines[i], substr)
		}
	}
}
