package models_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackuait/ghost-tab/internal/models"
)

func TestDetectAITools(t *testing.T) {
	// Create temp bin directory with mock commands
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	os.Mkdir(binDir, 0755)

	// Create mock executables
	os.WriteFile(filepath.Join(binDir, "claude"), []byte("#!/bin/bash\necho test"), 0755)
	os.WriteFile(filepath.Join(binDir, "codex"), []byte("#!/bin/bash\necho test"), 0755)

	// Update PATH for test
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	tools := models.DetectAITools()

	// Check that claude and codex are detected
	claudeFound := false
	codexFound := false

	for _, tool := range tools {
		if tool.Name == "claude" && tool.Installed {
			claudeFound = true
		}
		if tool.Name == "codex" && tool.Installed {
			codexFound = true
		}
	}

	if !claudeFound {
		t.Error("Expected claude to be detected")
	}
	if !codexFound {
		t.Error("Expected codex to be detected")
	}
}

func TestAIToolString(t *testing.T) {
	tool := models.AITool{
		Name:      "claude",
		Command:   "claude",
		Installed: true,
	}

	str := tool.String()
	if str != "Claude Code ✓" {
		t.Errorf("Expected 'Claude Code ✓', got %q", str)
	}

	tool.Installed = false
	str = tool.String()
	if str != "Claude Code (not installed)" {
		t.Errorf("Expected 'Claude Code (not installed)', got %q", str)
	}
}

func TestDetectAITools_AllToolsDetected(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	os.Mkdir(binDir, 0755)

	// Create all 4 mock executables (note: "gh copilot" uses "gh" as the binary)
	for _, cmd := range []string{"claude", "codex", "gh", "opencode"} {
		os.WriteFile(filepath.Join(binDir, cmd), []byte("#!/bin/bash\necho test"), 0755)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	tools := models.DetectAITools()

	if len(tools) != 4 {
		t.Fatalf("Expected 4 tools, got %d", len(tools))
	}

	// claude, codex, opencode should be installed (their commands are single binaries)
	// copilot uses "gh copilot" — exec.LookPath only checks first word, so "gh" must exist
	expected := map[string]bool{
		"claude":   true,
		"codex":    true,
		"copilot":  false, // "gh copilot" won't pass LookPath (it looks for literal "gh copilot")
		"opencode": true,
	}

	for _, tool := range tools {
		want, ok := expected[tool.Name]
		if !ok {
			t.Errorf("Unexpected tool: %s", tool.Name)
			continue
		}
		if tool.Installed != want {
			t.Errorf("Tool %s: expected Installed=%v, got %v", tool.Name, want, tool.Installed)
		}
	}
}

func TestDetectAITools_NoneInstalled(t *testing.T) {
	// Use empty PATH so no tools are found
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", t.TempDir())
	defer os.Setenv("PATH", oldPath)

	tools := models.DetectAITools()

	for _, tool := range tools {
		if tool.Installed {
			t.Errorf("Expected %s to not be installed with empty PATH", tool.Name)
		}
	}
}

func TestAIToolString_AllTools(t *testing.T) {
	tests := []struct {
		name      string
		installed bool
		want      string
	}{
		{"claude", true, "Claude Code ✓"},
		{"claude", false, "Claude Code (not installed)"},
		{"codex", true, "Codex CLI ✓"},
		{"codex", false, "Codex CLI (not installed)"},
		{"copilot", true, "Copilot CLI ✓"},
		{"copilot", false, "Copilot CLI (not installed)"},
		{"opencode", true, "OpenCode ✓"},
		{"opencode", false, "OpenCode (not installed)"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+fmt.Sprintf("%v", tt.installed), func(t *testing.T) {
			tool := models.AITool{Name: tt.name, Installed: tt.installed}
			got := tool.String()
			if got != tt.want {
				t.Errorf("AITool{%q, installed=%v}.String() = %q, want %q", tt.name, tt.installed, got, tt.want)
			}
		})
	}
}
