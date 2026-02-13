package bash_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// releaseSnippet builds a bash snippet that sources scripts/release.sh
// (which has a source guard so main doesn't run), then runs the provided bash code.
func releaseSnippet(t *testing.T, body string) string {
	t.Helper()
	root := projectRoot(t)
	releasePath := filepath.Join(root, "scripts", "release.sh")
	return fmt.Sprintf("source %q && %s", releasePath, body)
}

// initGitRepo creates a minimal git repo in dir with one commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "init", "--initial-branch=main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup failed: %v\n%s", err, out)
		}
	}
	writeTempFile(t, dir, "dummy", "init")
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "init"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git commit failed: %v\n%s", err, out)
		}
	}
}

// ============================================================
// check_clean_tree tests
// ============================================================

func TestCheckCleanTree_passes_on_clean_repo(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	snippet := releaseSnippet(t, `cd "`+dir+`" && check_clean_tree`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	_ = out
}

func TestCheckCleanTree_fails_on_dirty_repo(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	writeTempFile(t, dir, "untracked.txt", "dirty")

	snippet := releaseSnippet(t, `cd "`+dir+`" && check_clean_tree`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 1)
	assertContains(t, out, "clean")
}

// ============================================================
// check_main_branch tests
// ============================================================

func TestCheckMainBranch_passes_on_main(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	snippet := releaseSnippet(t, `cd "`+dir+`" && check_main_branch`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	_ = out
}

func TestCheckMainBranch_fails_on_other_branch(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	cmd := exec.Command("git", "checkout", "-b", "feature")
	cmd.Dir = dir
	cmd.CombinedOutput()

	snippet := releaseSnippet(t, `cd "`+dir+`" && check_main_branch`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 1)
	assertContains(t, out, "main")
}

// ============================================================
// read_version tests
// ============================================================

func TestReadVersion_reads_valid_semver(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "VERSION", "2.0.0\n")

	snippet := releaseSnippet(t, `read_version "`+filepath.Join(dir, "VERSION")+`"`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "2.0.0" {
		t.Errorf("got %q, want %q", strings.TrimSpace(out), "2.0.0")
	}
}

func TestReadVersion_fails_on_missing_file(t *testing.T) {
	snippet := releaseSnippet(t, `read_version "/nonexistent/VERSION"`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 1)
	assertContains(t, out, "VERSION")
}

func TestReadVersion_fails_on_invalid_format(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "VERSION", "not-a-version\n")

	snippet := releaseSnippet(t, `read_version "`+filepath.Join(dir, "VERSION")+`"`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 1)
	assertContains(t, out, "semver")
}

// ============================================================
// check_tag_not_exists tests
// ============================================================

func TestCheckTagNotExists_passes_when_tag_missing(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	snippet := releaseSnippet(t, `cd "`+dir+`" && check_tag_not_exists "v2.0.0"`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	_ = out
}

func TestCheckTagNotExists_fails_when_tag_exists(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	cmd := exec.Command("git", "tag", "v1.0.0")
	cmd.Dir = dir
	cmd.CombinedOutput()

	snippet := releaseSnippet(t, `cd "`+dir+`" && check_tag_not_exists "v1.0.0"`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 1)
	assertContains(t, out, "v1.0.0")
}

// ============================================================
// check_gh_auth tests
// ============================================================

func TestCheckGhAuth_passes_when_authenticated(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "gh", `
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then exit 0; fi
exit 0
`)
	snippet := releaseSnippet(t, `check_gh_auth`)
	env := buildEnv(t, []string{binDir})
	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestCheckGhAuth_fails_when_not_authenticated(t *testing.T) {
	dir := t.TempDir()
	binDir := mockCommand(t, dir, "gh", `
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then exit 1; fi
exit 0
`)
	snippet := releaseSnippet(t, `check_gh_auth`)
	env := buildEnv(t, []string{binDir})
	out, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 1)
	assertContains(t, out, "authenticated")
}

func TestCheckGhAuth_fails_when_not_installed(t *testing.T) {
	dir := t.TempDir()
	// Create an empty bin dir with no gh command
	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)
	snippet := releaseSnippet(t, `check_gh_auth`)
	env := buildEnv(t, []string{binDir}, "PATH="+binDir)
	out, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 1)
	assertContains(t, out, "gh")
}

// ============================================================
// check_formula_exists tests
// ============================================================

func TestCheckFormulaExists_passes_when_file_exists(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "ghost-tab.rb", "class GhostTab < Formula\nend\n")
	formulaPath := filepath.Join(dir, "ghost-tab.rb")

	snippet := releaseSnippet(t, `check_formula_exists "`+formulaPath+`"`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
}

func TestCheckFormulaExists_fails_when_file_missing(t *testing.T) {
	dir := t.TempDir()
	formulaPath := filepath.Join(dir, "ghost-tab.rb")

	snippet := releaseSnippet(t, `check_formula_exists "`+formulaPath+`"`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 1)
	assertContains(t, out, "formula")
}

// ============================================================
// update_formula tests
// ============================================================

func TestUpdateFormula_updates_url_and_sha256(t *testing.T) {
	dir := t.TempDir()
	formula := `class GhostTab < Formula
  url "https://github.com/JackUait/ghost-tab/archive/refs/tags/v1.5.0.tar.gz"
  sha256 "oldsha256value"
  license "MIT"
end
`
	formulaPath := writeTempFile(t, dir, "ghost-tab.rb", formula)

	snippet := releaseSnippet(t, `update_formula "`+formulaPath+`" "2.0.0" "newsha256value"`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	_ = out

	updated, err := os.ReadFile(formulaPath)
	if err != nil {
		t.Fatalf("failed to read formula: %v", err)
	}
	content := string(updated)
	assertContains(t, content, "v2.0.0.tar.gz")
	assertContains(t, content, `sha256 "newsha256value"`)
	assertNotContains(t, content, "v1.5.0")
	assertNotContains(t, content, "oldsha256value")
}

func TestUpdateFormula_preserves_other_content(t *testing.T) {
	dir := t.TempDir()
	formula := `class GhostTab < Formula
  desc "Ghostty + tmux wrapper"
  homepage "https://github.com/JackUait/ghost-tab"
  url "https://github.com/JackUait/ghost-tab/archive/refs/tags/v1.5.0.tar.gz"
  sha256 "oldsha"
  license "MIT"

  depends_on "tmux"

  def install
    bin.install "bin/ghost-tab"
  end
end
`
	formulaPath := writeTempFile(t, dir, "ghost-tab.rb", formula)

	snippet := releaseSnippet(t, `update_formula "`+formulaPath+`" "3.0.0" "newsha"`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)

	updated, _ := os.ReadFile(formulaPath)
	content := string(updated)
	assertContains(t, content, `desc "Ghostty + tmux wrapper"`)
	assertContains(t, content, `depends_on "tmux"`)
	assertContains(t, content, `bin.install "bin/ghost-tab"`)
	assertContains(t, content, "v3.0.0.tar.gz")
	assertContains(t, content, `sha256 "newsha"`)
}

// ============================================================
// main / integration tests
// ============================================================

func TestRelease_main_fails_on_dirty_tree(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	writeTempFile(t, dir, "VERSION", "1.0.0\n")
	writeTempFile(t, dir, "untracked.txt", "dirty")

	root := projectRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release.sh")
	cmd := exec.Command("bash", scriptPath, "--yes")
	cmd.Dir = dir
	cmd.Env = buildEnv(t, nil,
		"RELEASE_VERSION_FILE="+filepath.Join(dir, "VERSION"),
	)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		}
	}
	if code == 0 {
		t.Error("expected non-zero exit code for dirty tree")
	}
	assertContains(t, string(out), "clean")
}

func TestRelease_main_shows_confirmation_and_aborts_on_no(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	writeTempFile(t, dir, "VERSION", "1.0.0\n")
	// Stage and commit VERSION so tree is clean
	cmd := exec.Command("git", "add", "VERSION")
	cmd.Dir = dir
	cmd.CombinedOutput()
	cmd = exec.Command("git", "commit", "-m", "add version")
	cmd.Dir = dir
	cmd.CombinedOutput()

	root := projectRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release.sh")
	cmd = exec.Command("bash", scriptPath)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("n\n")

	// Mock gh as authenticated, and provide a formula path that exists
	mockDir := t.TempDir()
	binDir := mockCommand(t, mockDir, "gh", `
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then exit 0; fi
exit 0
`)
	formulaDir := t.TempDir()
	writeTempFile(t, formulaDir, "Formula/ghost-tab.rb", "class GhostTab\nend\n")

	cmd.Env = buildEnv(t, []string{binDir},
		"RELEASE_VERSION_FILE="+filepath.Join(dir, "VERSION"),
		"RELEASE_FORMULA_PATH="+filepath.Join(formulaDir, "Formula", "ghost-tab.rb"),
	)

	out, _ := cmd.CombinedOutput()
	assertContains(t, string(out), "Release v1.0.0")
	assertContains(t, string(out), "Aborted")
}
