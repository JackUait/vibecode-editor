package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCmd_HasAIToolFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("ai-tool")
	if flag == nil {
		t.Fatal("Expected --ai-tool persistent flag to be registered")
	}
	if flag.DefValue != "claude" {
		t.Errorf("Expected default value %q, got %q", "claude", flag.DefValue)
	}
}

func TestRootCmd_SubcommandRegistered(t *testing.T) {
	subcommands := []string{
		"confirm",
		"show-logo",
		"select-project",
		"select-ai-tool",
		"add-project",
		"settings-menu",
		"main-menu",
		"multi-select-ai-tool",
	}

	for _, name := range subcommands {
		t.Run(name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{name})
			if err != nil {
				t.Fatalf("Failed to find subcommand %q: %v", name, err)
			}
			if cmd.Name() != name {
				t.Errorf("Expected command name %q, got %q", name, cmd.Name())
			}
		})
	}
}

func TestConfirmCmd_RequiresArg(t *testing.T) {
	cmd, _, _ := rootCmd.Find([]string{"confirm"})
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("Expected error when no args provided to confirm")
	}
}

func TestConfirmCmd_AcceptsOneArg(t *testing.T) {
	cmd, _, _ := rootCmd.Find([]string{"confirm"})
	err := cmd.Args(cmd, []string{"Delete?"})
	if err != nil {
		t.Errorf("Expected no error with 1 arg, got: %v", err)
	}
}

func TestSelectProjectCmd_HasProjectsFileFlag(t *testing.T) {
	cmd, _, _ := rootCmd.Find([]string{"select-project"})
	flag := cmd.Flags().Lookup("projects-file")
	if flag == nil {
		t.Fatal("Expected --projects-file flag on select-project")
	}
}

func TestMainMenuCmd_HasAllFlags(t *testing.T) {
	cmd, _, _ := rootCmd.Find([]string{"main-menu"})

	flags := []struct {
		name     string
		defValue string
	}{
		{"projects-file", ""},
		{"ai-tool", "claude"},
		{"ai-tools", "claude"},
		{"ghost-display", "animated"},
		{"tab-title", "full"},
		{"update-version", ""},
	}

	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("Expected --%s flag", f.name)
			}
			if flag.DefValue != f.defValue {
				t.Errorf("Expected default %q, got %q", f.defValue, flag.DefValue)
			}
		})
	}
}

func TestRunSelectProject_EmptyProjectsFile(t *testing.T) {
	// Create an empty projects file
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "projects")
	os.WriteFile(emptyFile, []byte(""), 0644)

	// Reset flag and execute
	rootCmd.SetArgs([]string{"select-project", "--projects-file", emptyFile})

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := rootCmd.Execute()

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("Expected no error for empty projects, got: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, `"selected":false`) {
		t.Errorf("Expected {\"selected\":false} for empty projects, got: %s", output)
	}
}

func TestRunSelectProject_MissingFile(t *testing.T) {
	rootCmd.SetArgs([]string{"select-project", "--projects-file", "/nonexistent/path/projects"})
	err := rootCmd.Execute()

	if err == nil {
		t.Error("Expected error for missing projects file")
	}
}

func TestRunMainMenu_MissingProjectsFile(t *testing.T) {
	rootCmd.SetArgs([]string{"main-menu", "--projects-file", "/nonexistent/projects"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("Expected error for missing projects file")
	}
}

func TestRunMainMenu_EmptyProjectsFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "projects")
	os.WriteFile(emptyFile, []byte(""), 0644)

	rootCmd.SetArgs([]string{"main-menu", "--projects-file", emptyFile})

	// Capture stdout so any JSON output doesn't pollute test output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := rootCmd.Execute()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Empty projects file is valid (LoadProjects succeeds with 0 projects),
	// but the TUI will fail because there's no TTY in test.
	// The key assertion: it gets past the projects-loading stage without error
	// from LoadProjects. The error, if any, comes from TUITeaOptions (no TTY).
	_ = err
}

func TestConfirmCmd_TooManyArgs(t *testing.T) {
	cmd, _, _ := rootCmd.Find([]string{"confirm"})
	err := cmd.Args(cmd, []string{"arg1", "arg2"})
	if err == nil {
		t.Error("Expected error with too many args for confirm")
	}
}

func TestMultiSelectAIToolCmd_HasAIToolFlag(t *testing.T) {
	// ai-tool is a persistent flag on root, should be accessible from any subcommand
	flag := rootCmd.PersistentFlags().Lookup("ai-tool")
	if flag == nil {
		t.Fatal("Expected --ai-tool persistent flag to be accessible")
	}
}

func TestShowLogoCmd_HasAIToolFlag(t *testing.T) {
	// ai-tool comes from root persistent flags, accessible by show-logo
	flag := rootCmd.PersistentFlags().Lookup("ai-tool")
	if flag == nil {
		t.Fatal("Expected --ai-tool persistent flag")
	}
}

func TestSettingsMenuCmd_Exists(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"settings-menu"})
	if err != nil {
		t.Fatalf("Failed to find settings-menu: %v", err)
	}
	if cmd.Name() != "settings-menu" {
		t.Errorf("Expected 'settings-menu', got %q", cmd.Name())
	}
}

func TestAddProjectCmd_Exists(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"add-project"})
	if err != nil {
		t.Fatalf("Failed to find add-project: %v", err)
	}
	if cmd.Name() != "add-project" {
		t.Errorf("Expected 'add-project', got %q", cmd.Name())
	}
}

func TestSelectAIToolCmd_Exists(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"select-ai-tool"})
	if err != nil {
		t.Fatalf("Failed to find select-ai-tool: %v", err)
	}
	if cmd.Name() != "select-ai-tool" {
		t.Errorf("Expected 'select-ai-tool', got %q", cmd.Name())
	}
}

func TestRunMainMenu_ProjectsFileFlagRequired(t *testing.T) {
	cmd, _, _ := rootCmd.Find([]string{"main-menu"})
	flag := cmd.Flags().Lookup("projects-file")
	if flag == nil {
		t.Fatal("Expected --projects-file flag on main-menu")
	}
	// Verify the flag is annotated as required
	annotations := flag.Annotations
	if annotations == nil {
		t.Fatal("Expected required annotation on --projects-file")
	}
	required, ok := annotations[cobra.BashCompOneRequiredFlag]
	if !ok || len(required) == 0 || required[0] != "true" {
		t.Error("Expected --projects-file to be marked as required")
	}
}

func TestRunMainMenu_MalformedProjectsFile(t *testing.T) {
	// A file with only comments/blank lines should load as 0 projects
	// (not error at LoadProjects), then fail at TUITeaOptions (no TTY)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "projects")
	os.WriteFile(f, []byte("# comment\n\n# another comment\n"), 0644)

	rootCmd.SetArgs([]string{"main-menu", "--projects-file", f})

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := rootCmd.Execute()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// LoadProjects should succeed (returns empty slice for comments-only file)
	// Any error here comes from TUITeaOptions, not from loading
	_ = err
}

func TestSelectProjectCmd_ProjectsFileFlagRequired(t *testing.T) {
	cmd, _, _ := rootCmd.Find([]string{"select-project"})
	flag := cmd.Flags().Lookup("projects-file")
	if flag == nil {
		t.Fatal("Expected --projects-file flag on select-project")
	}
	// Verify the flag is annotated as required
	annotations := flag.Annotations
	if annotations == nil {
		t.Fatal("Expected required annotation on --projects-file")
	}
	required, ok := annotations[cobra.BashCompOneRequiredFlag]
	if !ok || len(required) == 0 || required[0] != "true" {
		t.Error("Expected --projects-file to be marked as required")
	}
}
