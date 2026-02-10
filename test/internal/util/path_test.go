package util_test

import (
	"os"
	"testing"

	"github.com/jackuait/ghost-tab/internal/util"
)

func TestExpandPath(t *testing.T) {
	home := os.Getenv("HOME")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"tilde expansion", "~/projects", home + "/projects"},
		{"no tilde", "/absolute/path", "/absolute/path"},
		{"tilde only", "~", home},
		{"tilde in middle", "/path/~/file", "/path/~/file"}, // Don't expand mid-path
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.ExpandPath(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	// Create temp dir for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid directory", tmpDir, false},
		{"nonexistent path", "/nonexistent/path/12345", true},
		{"empty path", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := util.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
