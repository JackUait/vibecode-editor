package models_test

import (
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
