package tui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jackuait/ghost-tab/internal/models"
	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestAppendProject(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "projects")

	err := tui.AppendProject("my-app", "/home/user/my-app", file)
	if err != nil {
		t.Fatalf("AppendProject: %v", err)
	}

	data, _ := os.ReadFile(file)
	if string(data) != "my-app:/home/user/my-app\n" {
		t.Errorf("File content: expected 'my-app:/home/user/my-app\\n', got %q", string(data))
	}
}

func TestAppendProject_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "projects")
	os.WriteFile(file, []byte("first:/tmp/first\n"), 0644)

	err := tui.AppendProject("second", "/tmp/second", file)
	if err != nil {
		t.Fatalf("AppendProject: %v", err)
	}

	data, _ := os.ReadFile(file)
	expected := "first:/tmp/first\nsecond:/tmp/second\n"
	if string(data) != expected {
		t.Errorf("File content: expected %q, got %q", expected, string(data))
	}
}

func TestAppendProject_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sub", "dir", "projects")

	err := tui.AppendProject("test", "/tmp/test", file)
	if err != nil {
		t.Fatalf("AppendProject should create parent dirs: %v", err)
	}

	data, _ := os.ReadFile(file)
	if string(data) != "test:/tmp/test\n" {
		t.Errorf("File content: expected 'test:/tmp/test\\n', got %q", string(data))
	}
}

func TestRemoveProject(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "projects")
	os.WriteFile(file, []byte("first:/tmp/first\nsecond:/tmp/second\nthird:/tmp/third\n"), 0644)

	err := tui.RemoveProject("second:/tmp/second", file)
	if err != nil {
		t.Fatalf("RemoveProject: %v", err)
	}

	data, _ := os.ReadFile(file)
	expected := "first:/tmp/first\nthird:/tmp/third\n"
	if string(data) != expected {
		t.Errorf("File content: expected %q, got %q", expected, string(data))
	}
}

func TestRemoveProject_SingleEntry(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "projects")
	os.WriteFile(file, []byte("only:/tmp/only\n"), 0644)

	err := tui.RemoveProject("only:/tmp/only", file)
	if err != nil {
		t.Fatalf("RemoveProject: %v", err)
	}

	data, _ := os.ReadFile(file)
	if string(data) != "" {
		t.Errorf("File content should be empty, got %q", string(data))
	}
}

func TestRemoveProject_NoMatch(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "projects")
	original := "first:/tmp/first\nsecond:/tmp/second\n"
	os.WriteFile(file, []byte(original), 0644)

	err := tui.RemoveProject("nonexistent:/tmp/nope", file)
	if err != nil {
		t.Fatalf("RemoveProject with no match: %v", err)
	}

	data, _ := os.ReadFile(file)
	if string(data) != original {
		t.Errorf("File should be unchanged, got %q", string(data))
	}
}

func TestRemoveProject_PartialMatchNotDeleted(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "projects")
	original := "app:/tmp/app\napp-long:/tmp/app-long\n"
	os.WriteFile(file, []byte(original), 0644)

	err := tui.RemoveProject("app:/tmp/app", file)
	if err != nil {
		t.Fatalf("RemoveProject: %v", err)
	}

	data, _ := os.ReadFile(file)
	if string(data) != "app-long:/tmp/app-long\n" {
		t.Errorf("Partial match should survive, got %q", string(data))
	}
}

func TestIsDuplicateProject(t *testing.T) {
	projects := []models.Project{
		{Name: "app", Path: "/home/user/app"},
		{Name: "web", Path: "/home/user/web"},
	}

	if !tui.IsDuplicateProject("/home/user/app", projects) {
		t.Error("Should detect duplicate path")
	}
	if tui.IsDuplicateProject("/home/user/new", projects) {
		t.Error("Should not flag non-duplicate")
	}
}

func TestIsDuplicateProject_TrailingSlash(t *testing.T) {
	projects := []models.Project{
		{Name: "app", Path: "/home/user/app"},
	}

	if !tui.IsDuplicateProject("/home/user/app/", projects) {
		t.Error("Should detect duplicate even with trailing slash")
	}
}
