package models_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yourusername/ghost-tab/internal/models"
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
