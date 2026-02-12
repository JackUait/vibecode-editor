package bash_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- menu-tui.sh tests (TestMenu_*) ----------

func TestMenu_selects_project_and_parses_JSON(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
echo "name=$_selected_project_name"
echo "path=$_selected_project_path"
echo "action=$_selected_project_action"
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	out, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)
	assertContains(t, out, "name=proj1")
	assertContains(t, out, "path=/tmp/p1")
	assertContains(t, out, "action=select-project")
}

func TestMenu_passes_correct_flags_to_main_menu(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "captured_args")
	binDir := mockCommand(t, dir, "ghost-tab-tui", fmt.Sprintf(`
echo "$*" > %q
echo '{"action":"quit"}'
`, argsFile))
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	configDir := filepath.Join(dir, "config", "ghost-tab")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, dir, "config/ghost-tab/settings", "ghost_display=static\n")

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude" "codex")
SELECTED_AI_TOOL="codex"
_update_version="2.0.0"
select_project_interactive %q || true
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, _ = runBashSnippet(t, script, env)

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("failed to read captured args: %v", err)
	}
	args := string(data)
	assertContains(t, args, "main-menu")
	assertContains(t, args, "--projects-file")
	assertContains(t, args, "--ai-tool")
	assertContains(t, args, "codex")
	assertContains(t, args, "--ai-tools")
	assertContains(t, args, "claude,codex")
	assertContains(t, args, "--ghost-display")
	assertContains(t, args, "static")
	assertContains(t, args, "--update-version")
	assertContains(t, args, "2.0.0")
}

func TestMenu_handles_AI_tool_change(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"codex"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude" "codex")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
echo "ai_tool=$_selected_ai_tool"
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	out, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)
	assertContains(t, out, "ai_tool=codex")
}

func TestMenu_handles_quit_action(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"quit"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, code := runBashSnippet(t, script, env)
	if code == 0 {
		t.Errorf("expected non-zero exit code for quit, got 0")
	}
}

func TestMenu_handles_open_once_action(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"open-once","name":"temp","path":"/tmp/temp","ai_tool":"claude"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
echo "action=$_selected_project_action"
echo "name=$_selected_project_name"
echo "path=$_selected_project_path"
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	out, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)
	assertContains(t, out, "action=open-once")
	assertContains(t, out, "name=temp")
	assertContains(t, out, "path=/tmp/temp")
}

func TestMenu_handles_plain_terminal_action(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"plain-terminal","ai_tool":"claude"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
echo "action=$_selected_project_action"
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	out, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)
	assertContains(t, out, "action=plain-terminal")
}

func TestMenu_handles_settings_action(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"settings","ai_tool":"claude"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
echo "action=$_selected_project_action"
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	out, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)
	assertContains(t, out, "action=settings")
}

func TestMenu_persists_ghost_display_change(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude","ghost_display":"static"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	configDir := filepath.Join(dir, "config", "ghost-tab")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)

	settingsFile := filepath.Join(dir, "config", "ghost-tab", "settings")
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("settings file not created: %v", err)
	}
	assertContains(t, string(data), "ghost_display=static")
}

func TestMenu_updates_existing_ghost_display(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude","ghost_display":"none"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	writeTempFile(t, dir, "config/ghost-tab/settings", "ghost_display=animated\n")

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)

	settingsFile := filepath.Join(dir, "config", "ghost-tab", "settings")
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("settings file not found: %v", err)
	}
	assertContains(t, string(data), "ghost_display=none")
	assertNotContains(t, string(data), "ghost_display=animated")
}

func TestMenu_reads_ghost_display_from_settings(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "captured_args")
	binDir := mockCommand(t, dir, "ghost-tab-tui", fmt.Sprintf(`
echo "$*" > %q
echo '{"action":"quit"}'
`, argsFile))
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	writeTempFile(t, dir, "config/ghost-tab/settings", "ghost_display=none\n")

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q || true
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, _ = runBashSnippet(t, script, env)

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args file not found: %v", err)
	}
	args := string(data)
	assertContains(t, args, "--ghost-display")
	assertContains(t, args, "none")
}

func TestMenu_defaults_ghost_display_to_animated(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "captured_args")
	binDir := mockCommand(t, dir, "ghost-tab-tui", fmt.Sprintf(`
echo "$*" > %q
echo '{"action":"quit"}'
`, argsFile))
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	// No settings file

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q || true
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, _ = runBashSnippet(t, script, env)

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args file not found: %v", err)
	}
	args := string(data)
	assertContains(t, args, "--ghost-display")
	assertContains(t, args, "animated")
}

func TestMenu_validates_null_name_on_select_project(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":null,"path":"/tmp/p1","ai_tool":"claude"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	out, code := runBashSnippet(t, script, env)
	if code == 0 {
		t.Error("expected non-zero exit code for null name")
	}
	assertContains(t, out, "invalid project name")
}

func TestMenu_validates_null_path_on_select_project(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":"proj1","path":null,"ai_tool":"claude"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	out, code := runBashSnippet(t, script, env)
	if code == 0 {
		t.Error("expected non-zero exit code for null path")
	}
	assertContains(t, out, "invalid project path")
}

func TestMenu_handles_jq_parse_failure(t *testing.T) {
	dir := t.TempDir()
	// ghost-tab-tui outputs invalid JSON
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo 'not json at all'`)
	// Mock jq to fail
	mockCommand(t, dir, "jq", `cat > /dev/null; exit 1`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	out, code := runBashSnippet(t, script, env)
	if code == 0 {
		t.Error("expected non-zero exit code for jq parse failure")
	}
	assertContains(t, out, "Failed to parse")
}

func TestMenu_handles_binary_missing(t *testing.T) {
	dir := t.TempDir()
	// Don't put ghost-tab-tui in PATH at all
	// Create a bin dir with no ghost-tab-tui
	binDir := filepath.Join(dir, "emptybin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)

	// Use a PATH that does NOT include ghost-tab-tui, but does include jq, bash, etc.
	env := buildEnv(t, nil,
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
		// Override PATH to remove any real ghost-tab-tui but keep system commands
		"PATH="+binDir+":/usr/bin:/bin:/usr/sbin:/sbin",
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	out, code := runBashSnippet(t, script, env)
	if code == 0 {
		t.Error("expected non-zero exit code for missing binary")
	}
	assertContains(t, out, "ghost-tab-tui binary not found")
}

func TestMenu_handles_ghost_tab_tui_failure(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `exit 1`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, code := runBashSnippet(t, script, env)
	if code == 0 {
		t.Error("expected non-zero exit code for ghost-tab-tui failure")
	}
}

func TestMenu_omits_update_version_flag_when_empty(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "captured_args")
	binDir := mockCommand(t, dir, "ghost-tab-tui", fmt.Sprintf(`
echo "$*" > %q
echo '{"action":"quit"}'
`, argsFile))
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q || true
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, _ = runBashSnippet(t, script, env)

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args file not found: %v", err)
	}
	assertNotContains(t, string(data), "--update-version")
}

func TestMenu_reads_tab_title_from_settings(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "captured_args")
	binDir := mockCommand(t, dir, "ghost-tab-tui", fmt.Sprintf(`
echo "$*" > %q
echo '{"action":"quit"}'
`, argsFile))
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	writeTempFile(t, dir, "config/ghost-tab/settings", "tab_title=project\n")

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q || true
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, _ = runBashSnippet(t, script, env)

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args file not found: %v", err)
	}
	assertContains(t, string(data), "--tab-title project")
}

func TestMenu_defaults_tab_title_to_full(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "captured_args")
	binDir := mockCommand(t, dir, "ghost-tab-tui", fmt.Sprintf(`
echo "$*" > %q
echo '{"action":"quit"}'
`, argsFile))
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	// No settings file

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q || true
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, _ = runBashSnippet(t, script, env)

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args file not found: %v", err)
	}
	args := string(data)
	assertContains(t, args, "--tab-title")
	assertContains(t, args, "full")
}

func TestMenu_persists_tab_title_change(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude","tab_title":"project"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	configDir := filepath.Join(dir, "config", "ghost-tab")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)

	settingsFile := filepath.Join(dir, "config", "ghost-tab", "settings")
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("settings file not created: %v", err)
	}
	assertContains(t, string(data), "tab_title=project")
}

func TestMenu_persists_ai_tool_change_to_file(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"codex"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	configDir := filepath.Join(dir, "config", "ghost-tab")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude" "codex")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)

	aiToolFile := filepath.Join(dir, "config", "ghost-tab", "ai-tool")
	data, err := os.ReadFile(aiToolFile)
	if err != nil {
		t.Fatalf("ai-tool file not created: %v", err)
	}
	if strings.TrimSpace(string(data)) != "codex" {
		t.Errorf("ai-tool file content = %q, want %q", strings.TrimSpace(string(data)), "codex")
	}
}

func TestMenu_does_not_write_ai_tool_file_when_unchanged(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	configDir := filepath.Join(dir, "config", "ghost-tab")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)

	aiToolFile := filepath.Join(dir, "config", "ghost-tab", "ai-tool")
	if _, err := os.Stat(aiToolFile); err == nil {
		t.Error("ai-tool file should not exist when tool is unchanged")
	}
}

func TestMenu_sets_selected_ai_tool_for_settings_action(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"settings","ai_tool":"codex"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude" "codex")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
echo "ai_tool=$_selected_ai_tool"
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	out, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)
	assertContains(t, out, "ai_tool=codex")
}

func TestMenu_ai_tool_persists_between_sessions(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "config", "ghost-tab")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	root := projectRoot(t)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	// Session 1: user cycles to codex and selects a project
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"codex"}'`)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script1 := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude" "codex" "copilot" "opencode")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, code := runBashSnippet(t, script1, env)
	assertExitCode(t, code, 0)

	// Verify file was written
	aiToolFile := filepath.Join(dir, "config", "ghost-tab", "ai-tool")
	data, err := os.ReadFile(aiToolFile)
	if err != nil {
		t.Fatalf("ai-tool file not created: %v", err)
	}
	if strings.TrimSpace(string(data)) != "codex" {
		t.Errorf("ai-tool file = %q, want %q", strings.TrimSpace(string(data)), "codex")
	}

	// Session 2: simulate wrapper reading the file and passing to TUI
	argsFile := filepath.Join(dir, "captured_args")
	// Recreate the mock to capture args this time
	binDir2 := filepath.Join(dir, "bin2")
	if err := os.MkdirAll(binDir2, 0755); err != nil {
		t.Fatal(err)
	}
	mockScript := fmt.Sprintf(`#!/bin/bash
# Capture the --ai-tool flag
ai_flag=""
while [[ $# -gt 0 ]]; do
  if [[ "$1" == "--ai-tool" ]]; then
    ai_flag="$2"
    break
  fi
  shift
done
echo "$ai_flag" > %q
echo "{\"action\":\"select-project\",\"name\":\"proj1\",\"path\":\"/tmp/p1\",\"ai_tool\":\"$ai_flag\"}"
`, argsFile)
	if err := os.WriteFile(filepath.Join(binDir2, "ghost-tab-tui"), []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	env2 := buildEnv(t, []string{binDir2},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script2 := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude" "codex" "copilot" "opencode")
# Read the saved preference like the wrapper does
AI_TOOL_PREF_FILE="%s"
SELECTED_AI_TOOL=""
if [ -f "$AI_TOOL_PREF_FILE" ]; then
  SELECTED_AI_TOOL="$(cat "$AI_TOOL_PREF_FILE" 2>/dev/null | tr -d '[:space:]')"
fi
validate_ai_tool
_update_version=""
select_project_interactive %q
echo "ai_tool=$_selected_ai_tool"
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/ai-tools.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		aiToolFile,
		projectsFile)

	out2, code2 := runBashSnippet(t, script2, env2)
	assertExitCode(t, code2, 0)

	// Verify the TUI received "codex"
	capturedData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("captured args not found: %v", err)
	}
	if strings.TrimSpace(string(capturedData)) != "codex" {
		t.Errorf("captured ai_tool = %q, want %q", strings.TrimSpace(string(capturedData)), "codex")
	}

	assertContains(t, out2, "ai_tool=codex")

	// File should still have codex
	finalData, err := os.ReadFile(aiToolFile)
	if err != nil {
		t.Fatalf("ai-tool file not found: %v", err)
	}
	if strings.TrimSpace(string(finalData)) != "codex" {
		t.Errorf("ai-tool file = %q, want %q", strings.TrimSpace(string(finalData)), "codex")
	}
}

func TestMenu_does_not_persist_ai_tool_on_quit(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"quit","ai_tool":"codex"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	// Pre-set ai-tool file to "claude"
	writeTempFile(t, dir, "config/ghost-tab/ai-tool", "claude")

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude" "codex")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q || true
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, _ = runBashSnippet(t, script, env)

	aiToolFile := filepath.Join(dir, "config", "ghost-tab", "ai-tool")
	data, err := os.ReadFile(aiToolFile)
	if err != nil {
		t.Fatalf("ai-tool file not found: %v", err)
	}
	if strings.TrimSpace(string(data)) != "claude" {
		t.Errorf("ai-tool should still be 'claude' after quit, got %q", strings.TrimSpace(string(data)))
	}
}

func TestMenu_updates_existing_tab_title_in_settings(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude","tab_title":"project"}'`)
	projectsFile := writeTempFile(t, dir, "projects", "proj1:/tmp/p1\n")

	writeTempFile(t, dir, "config/ghost-tab/settings", "tab_title=full\n")

	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
select_project_interactive %q
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)

	settingsFile := filepath.Join(dir, "config", "ghost-tab", "settings")
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("settings file not found: %v", err)
	}
	assertContains(t, string(data), "tab_title=project")
	assertNotContains(t, string(data), "tab_title=full")
}

func TestMenu_passes_sound_enabled_flag_when_sound_enabled(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "captured_args")
	binDir := mockCommand(t, dir, "ghost-tab-tui", fmt.Sprintf(`
echo "$*" > %q
echo '{"action":"quit"}'
`, argsFile))
	projectsFile := writeTempFile(t, dir, "projects", "test:/tmp/test\n")
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir},
		"XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
AI_TOOLS_AVAILABLE=("claude")
SELECTED_AI_TOOL="claude"
_update_version=""
# Mock is_sound_enabled to return true
is_sound_enabled() { echo "true"; }
select_project_interactive %q || true
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/menu-tui.sh"),
		projectsFile)

	_, _ = runBashSnippet(t, script, env)

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args file not found: %v", err)
	}
	assertContains(t, string(data), "--sound-enabled")
}

// ---------- settings-menu-tui.sh tests (TestSettingsMenu_*) ----------

func TestSettingsMenu_calls_ghost_tab_tui_and_parses_JSON(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `
if [[ "$1" == "settings-menu" ]]; then
  echo '{"action":"toggle-ghost"}'
  exit 0
fi
exit 1
`)
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir})

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
settings_menu_interactive
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/settings-menu-tui.sh"))

	out, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "toggle-ghost" {
		t.Errorf("expected output 'toggle-ghost', got %q", strings.TrimSpace(out))
	}
}

func TestSettingsMenu_handles_quit_action(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `
if [[ "$1" == "settings-menu" ]]; then
  echo '{"action":"quit"}'
  exit 0
fi
exit 1
`)
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir})

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
settings_menu_interactive
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/settings-menu-tui.sh"))

	out, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "quit" {
		t.Errorf("expected output 'quit', got %q", strings.TrimSpace(out))
	}
}

func TestSettingsMenu_handles_binary_missing(t *testing.T) {
	dir := t.TempDir()
	// Create empty bin dir with no ghost-tab-tui
	binDir := filepath.Join(dir, "emptybin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	root := projectRoot(t)
	env := buildEnv(t, nil,
		"PATH="+binDir+":/usr/bin:/bin:/usr/sbin:/sbin",
	)

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
settings_menu_interactive
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/settings-menu-tui.sh"))

	out, code := runBashSnippet(t, script, env)
	if code == 0 {
		t.Error("expected non-zero exit code for missing binary")
	}
	assertContains(t, out, "ghost-tab-tui binary not found")
}

func TestSettingsMenu_handles_jq_parse_failure(t *testing.T) {
	dir := t.TempDir()
	// ghost-tab-tui returns valid JSON but we mock jq to fail
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":"toggle-ghost"}'`)
	mockCommand(t, dir, "jq", `cat > /dev/null; exit 1`)
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir})

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
settings_menu_interactive
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/settings-menu-tui.sh"))

	out, code := runBashSnippet(t, script, env)
	if code == 0 {
		t.Error("expected non-zero exit code for jq failure")
	}
	assertContains(t, out, "Failed to parse")
}

func TestSettingsMenu_validates_null_action(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":null}'`)
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir})

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
settings_menu_interactive
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/settings-menu-tui.sh"))

	out, code := runBashSnippet(t, script, env)
	if code == 0 {
		t.Error("expected non-zero exit code for null action")
	}
	assertContains(t, out, "invalid action")
}

func TestSettingsMenu_allows_empty_action_for_quit(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "ghost-tab-tui", `echo '{"action":""}'`)
	root := projectRoot(t)
	env := buildEnv(t, []string{binDir})

	script := fmt.Sprintf(`
source %q 2>/dev/null || true
source %q
error() { echo "ERROR: $*" >&2; }
settings_menu_interactive
`, filepath.Join(root, "lib/tui.sh"),
		filepath.Join(root, "lib/settings-menu-tui.sh"))

	out, code := runBashSnippet(t, script, env)
	assertExitCode(t, code, 0)
	// Empty action should produce empty output
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty output for empty action, got %q", strings.TrimSpace(out))
	}
}
