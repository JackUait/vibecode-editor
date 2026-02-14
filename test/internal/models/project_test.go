package models_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackuait/ghost-tab/internal/models"
)

func TestLoadProjects(t *testing.T) {
	// Create temp projects file
	tmpDir := t.TempDir()
	projectsFile := filepath.Join(tmpDir, "projects")

	content := `# Comment line
web-app:/home/user/projects/web-app
cli-tool:~/code/cli-tool

data-pipeline:/opt/data-pipeline`

	err := os.WriteFile(projectsFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	projects, err := models.LoadProjects(projectsFile)
	if err != nil {
		t.Fatalf("LoadProjects failed: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(projects))
	}

	expected := []struct {
		name string
		path string
	}{
		{"web-app", "/home/user/projects/web-app"},
		{"cli-tool", "~/code/cli-tool"},
		{"data-pipeline", "/opt/data-pipeline"},
	}

	for i, exp := range expected {
		if projects[i].Name != exp.name {
			t.Errorf("Project %d: expected name %q, got %q", i, exp.name, projects[i].Name)
		}
		if projects[i].Path != exp.path {
			t.Errorf("Project %d: expected path %q, got %q", i, exp.path, projects[i].Path)
		}
	}
}

func TestLoadProjectsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	projectsFile := filepath.Join(tmpDir, "empty")

	err := os.WriteFile(projectsFile, []byte("# Only comments\n\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	projects, err := models.LoadProjects(projectsFile)
	if err != nil {
		t.Fatalf("LoadProjects failed: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
}

func TestProjectHasWorktrees(t *testing.T) {
	p := models.Project{
		Name: "test",
		Path: "/tmp/test",
		Worktrees: []models.Worktree{
			{Path: "/tmp/wt1", Branch: "feature/auth"},
			{Path: "/tmp/wt2", Branch: "fix/bug"},
		},
	}

	if len(p.Worktrees) != 2 {
		t.Errorf("expected 2 worktrees, got %d", len(p.Worktrees))
	}
}

func TestParseProjectName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "extracts name before colon",
			input: "myapp:/Users/me/myapp",
			want:  "myapp",
		},
		{
			name:  "handles name with spaces",
			input: "my app:/Users/me/my app",
			want:  "my app",
		},
		{
			name:  "handles name with special characters",
			input: "my-app_v2.0:/path",
			want:  "my-app_v2.0",
		},
		{
			name:  "handles empty name",
			input: ":/path/to/app",
			want:  "",
		},
		{
			name:  "handles unicode in name",
			input: "\u00e9moji\U0001F47B:/path",
			want:  "\u00e9moji\U0001F47B",
		},
		{
			name:  "handles line with no colon",
			input: "nocolon",
			want:  "nocolon",
		},
		{
			name:  "handles empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := models.ParseProjectName(tt.input)
			if got != tt.want {
				t.Errorf("ParseProjectName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseProjectPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "extracts path after first colon",
			input: "myapp:/Users/me/myapp",
			want:  "/Users/me/myapp",
		},
		{
			name:  "handles paths with colons",
			input: "myapp:/Users/me/path:with:colons",
			want:  "/Users/me/path:with:colons",
		},
		{
			name:  "handles empty path",
			input: "app:",
			want:  "",
		},
		{
			name:  "handles path with multiple colons at start",
			input: "app::/path",
			want:  ":/path",
		},
		{
			name:  "handles very long path",
			input: "app:" + strings.Repeat("/very/long/path", 100),
			want:  strings.Repeat("/very/long/path", 100),
		},
		{
			name:  "handles path with dollar signs",
			input: "app:/path/with/$VAR/here",
			want:  "/path/with/$VAR/here",
		},
		{
			name:  "handles path with backticks",
			input: "app:/path/with/`command`/here",
			want:  "/path/with/`command`/here",
		},
		{
			name:  "handles path with parentheses",
			input: "app:/path/with/(parens)/here",
			want:  "/path/with/(parens)/here",
		},
		{
			name:  "handles path with semicolons",
			input: "app:/path/with;semicolons;here",
			want:  "/path/with;semicolons;here",
		},
		{
			name:  "handles path with ampersands",
			input: "app:/path/with&ampersands&here",
			want:  "/path/with&ampersands&here",
		},
		{
			name:  "handles path with pipes",
			input: "app:/path/with|pipes|here",
			want:  "/path/with|pipes|here",
		},
		{
			name:  "handles path with asterisks",
			input: "app:/path/with/*/glob",
			want:  "/path/with/*/glob",
		},
		{
			name:  "handles path with question marks",
			input: "app:/path/with/?/glob",
			want:  "/path/with/?/glob",
		},
		{
			name:  "handles path with square brackets",
			input: "app:/path/with/[brackets]/here",
			want:  "/path/with/[brackets]/here",
		},
		{
			name:  "handles line with no colon returns original",
			input: "nocolon",
			want:  "nocolon",
		},
		{
			name:  "handles empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := models.ParseProjectPath(tt.input)
			if got != tt.want {
				t.Errorf("ParseProjectPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
