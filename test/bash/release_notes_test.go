package bash_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// addCommit creates a unique file and commits it with the given message.
func addCommit(t *testing.T, dir string, message string) {
	t.Helper()
	// Create a unique file based on the message hash to avoid conflicts
	filename := fmt.Sprintf("file-%d", len(message)+hash(message))
	writeTempFile(t, dir, filename, message)
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", message},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git commit failed: %v\n%s", err, out)
		}
	}
}

// hash returns a simple hash of a string for unique filenames.
func hash(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// addTag creates a git tag at the current HEAD.
func addTag(t *testing.T, dir string, tag string) {
	t.Helper()
	cmd := exec.Command("git", "tag", tag)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git tag failed: %v\n%s", err, out)
	}
}

// releaseNotesSnippet builds a bash snippet that sources
// scripts/generate-release-notes.sh then runs the provided bash code.
func releaseNotesSnippet(t *testing.T, body string) string {
	t.Helper()
	root := projectRoot(t)
	scriptPath := filepath.Join(root, "scripts", "generate-release-notes.sh")
	return fmt.Sprintf("source %q && %s", scriptPath, body)
}

// ============================================================
// generate_release_notes tests
// ============================================================

func TestGenerateReleaseNotes_groups_feat_commits(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "feat: add dark mode")
	addCommit(t, dir, "feat: add light mode")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "## Features")
	assertContains(t, out, "Add dark mode")
	assertContains(t, out, "Add light mode")
}

func TestGenerateReleaseNotes_groups_fix_commits(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "fix: resolve crash on startup")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "## Bug Fixes")
	assertContains(t, out, "Resolve crash on startup")
}

func TestGenerateReleaseNotes_groups_refactor_commits(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "refactor: simplify config loading")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "## Refactoring")
	assertContains(t, out, "Simplify config loading")
}

func TestGenerateReleaseNotes_filters_test_docs_chore(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "feat: add export feature")
	addCommit(t, dir, "test: add export tests")
	addCommit(t, dir, "docs: update readme")
	addCommit(t, dir, "chore: update dependencies")
	addCommit(t, dir, "style: fix formatting")
	addCommit(t, dir, "ci: add github actions")
	addCommit(t, dir, "build: update makefile")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "## Features")
	assertContains(t, out, "Add export feature")
	assertNotContains(t, out, "add export tests")
	assertNotContains(t, out, "update readme")
	assertNotContains(t, out, "update dependencies")
	assertNotContains(t, out, "fix formatting")
	assertNotContains(t, out, "github actions")
	assertNotContains(t, out, "update makefile")
}

func TestGenerateReleaseNotes_handles_scoped_prefixes(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "feat(release): add release script")
	addCommit(t, dir, "fix(menu): fix menu alignment")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "## Features")
	assertContains(t, out, "Add release script")
	assertContains(t, out, "## Bug Fixes")
	assertContains(t, out, "Fix menu alignment")
}

func TestGenerateReleaseNotes_multiple_groups_ordered(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "refactor: extract helper")
	addCommit(t, dir, "fix: null pointer check")
	addCommit(t, dir, "feat: add search")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)

	featIdx := strings.Index(out, "## Features")
	fixIdx := strings.Index(out, "## Bug Fixes")
	refactorIdx := strings.Index(out, "## Refactoring")

	if featIdx < 0 {
		t.Fatal("missing ## Features section")
	}
	if fixIdx < 0 {
		t.Fatal("missing ## Bug Fixes section")
	}
	if refactorIdx < 0 {
		t.Fatal("missing ## Refactoring section")
	}
	if featIdx >= fixIdx {
		t.Errorf("Features (idx %d) should appear before Bug Fixes (idx %d)", featIdx, fixIdx)
	}
	if fixIdx >= refactorIdx {
		t.Errorf("Bug Fixes (idx %d) should appear before Refactoring (idx %d)", fixIdx, refactorIdx)
	}
}

func TestGenerateReleaseNotes_no_user_facing_commits(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "test: add unit tests")
	addCommit(t, dir, "docs: update changelog")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertNotContains(t, out, "## Features")
	assertNotContains(t, out, "## Bug Fixes")
	assertNotContains(t, out, "## Refactoring")
	assertNotContains(t, out, "## Other Changes")
}

func TestGenerateReleaseNotes_unprefixed_commits_go_to_other(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "feat: add search")
	addCommit(t, dir, "update config defaults")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "## Features")
	assertContains(t, out, "Add search")
	assertContains(t, out, "## Other Changes")
	assertContains(t, out, "Update config defaults")
}

func TestGenerateReleaseNotes_strips_prefix_and_capitalizes(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "feat: add new feature")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "- Add new feature")
	assertNotContains(t, out, "feat:")
}

func TestGenerateReleaseNotes_filters_version_bump_commits(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "feat: add widget")
	addCommit(t, dir, "Bump version to 1.1.0")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "## Features")
	assertContains(t, out, "Add widget")
	assertNotContains(t, out, "Bump version")
}

func TestGenerateReleaseNotes_filters_merge_commits(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	addTag(t, dir, "v1.0.0")

	addCommit(t, dir, "feat: add dashboard")
	addCommit(t, dir, "Merge pull request #1 from user/branch")
	addTag(t, dir, "v1.1.0")

	snippet := releaseNotesSnippet(t, `cd "`+dir+`" && generate_release_notes v1.0.0 v1.1.0`)
	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "## Features")
	assertContains(t, out, "Add dashboard")
	assertNotContains(t, out, "Merge pull request")
}
