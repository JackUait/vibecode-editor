package util_test

import (
	"os"
	"strings"
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

func TestTruncatePath(t *testing.T) {
	t.Run("returns short paths unchanged", func(t *testing.T) {
		result, err := util.TruncatePath("~/short", 38)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "~/short" {
			t.Errorf("TruncatePath(\"~/short\", 38) = %q, want %q", result, "~/short")
		}
	})

	t.Run("truncates long paths with ellipsis", func(t *testing.T) {
		result, err := util.TruncatePath("~/very/long/deeply/nested/project/directory/name", 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "...") {
			t.Errorf("expected ellipsis in result %q", result)
		}
		if len([]rune(result)) > 30 {
			t.Errorf("result length %d exceeds max width 30: %q", len([]rune(result)), result)
		}
	})

	t.Run("preserves start and end of path", func(t *testing.T) {
		result, err := util.TruncatePath("~/very/long/deeply/nested/project/directory/name", 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(result, "~/") {
			t.Errorf("expected result to start with \"~/\", got %q", result)
		}
		if !strings.HasSuffix(result, "name") {
			t.Errorf("expected result to end with \"name\", got %q", result)
		}
	})

	t.Run("respects max width parameter", func(t *testing.T) {
		longPath := "~/this/is/a/really/long/path/that/goes/on/forever"

		result20, err := util.TruncatePath(longPath, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len([]rune(result20)) > 20 {
			t.Errorf("result length %d exceeds max width 20: %q", len([]rune(result20)), result20)
		}

		result40, err := util.TruncatePath(longPath, 40)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len([]rune(result40)) > 40 {
			t.Errorf("result length %d exceeds max width 40: %q", len([]rune(result40)), result40)
		}
	})

	t.Run("handles empty path", func(t *testing.T) {
		result, err := util.TruncatePath("", 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("TruncatePath(\"\", 30) = %q, want %q", result, "")
		}
	})

	t.Run("handles path with spaces", func(t *testing.T) {
		longPath := "~/very/long path/with/lots of spaces/deeply nested/directory"
		result, err := util.TruncatePath(longPath, 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len([]rune(result)) > 30 {
			t.Errorf("result length %d exceeds max width 30: %q", len([]rune(result)), result)
		}
		if !strings.HasPrefix(result, "~/") {
			t.Errorf("expected result to start with \"~/\", got %q", result)
		}
	})

	t.Run("handles path with unicode", func(t *testing.T) {
		longPath := "~/very/long/\u00e9moji/\U0001F47B/deeply/nested/directory/name"
		result, err := util.TruncatePath(longPath, 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "...") {
			t.Errorf("expected ellipsis in result %q", result)
		}
		if !strings.HasPrefix(result, "~/") {
			t.Errorf("expected result to start with \"~/\", got %q", result)
		}
	})

	t.Run("handles very long path", func(t *testing.T) {
		longComponent := strings.Repeat("a", 200)
		longPath := "~/" + longComponent + "/" + longComponent + "/" + longComponent
		result, err := util.TruncatePath(longPath, 50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len([]rune(result)) > 50 {
			t.Errorf("result length %d exceeds max width 50: %q", len([]rune(result)), result)
		}
		if !strings.Contains(result, "...") {
			t.Errorf("expected ellipsis in result %q", result)
		}
	})

	t.Run("handles path with trailing slash", func(t *testing.T) {
		longPath := "~/very/long/deeply/nested/project/directory/name/"
		result, err := util.TruncatePath(longPath, 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len([]rune(result)) > 30 {
			t.Errorf("result length %d exceeds max width 30: %q", len([]rune(result)), result)
		}
	})

	t.Run("handles single character path", func(t *testing.T) {
		result, err := util.TruncatePath("a", 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "a" {
			t.Errorf("TruncatePath(\"a\", 30) = %q, want %q", result, "a")
		}
	})

	t.Run("handles path exactly at max width", func(t *testing.T) {
		path := "12345678901234567890"
		result, err := util.TruncatePath(path, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != path {
			t.Errorf("TruncatePath(%q, 20) = %q, want %q", path, result, path)
		}
	})

	t.Run("handles path one char over max width", func(t *testing.T) {
		path := "123456789012345678901"
		result, err := util.TruncatePath(path, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len([]rune(result)) > 20 {
			t.Errorf("result length %d exceeds max width 20: %q", len([]rune(result)), result)
		}
		if !strings.Contains(result, "...") {
			t.Errorf("expected ellipsis in result %q", result)
		}
	})

	t.Run("handles max_width of 3", func(t *testing.T) {
		// In bash, half=0, so result is "..." + entire path (bash ${path: -0} returns whole string)
		// Go matches: when half is 0, return just "..."
		result, err := util.TruncatePath("/very/long/path", 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty result")
		}
		if !strings.Contains(result, "...") {
			t.Errorf("expected ellipsis in result %q", result)
		}
	})

	t.Run("handles max_width of 0", func(t *testing.T) {
		// Bash fails with "substring expression < 0" error
		_, err := util.TruncatePath("/path", 0)
		if err == nil {
			t.Error("expected error for max_width of 0")
		}
	})

	t.Run("handles negative max_width", func(t *testing.T) {
		// Bash fails with "substring expression < 0" error
		_, err := util.TruncatePath("/path", -10)
		if err == nil {
			t.Error("expected error for negative max_width")
		}
	})

	t.Run("handles path with all dots", func(t *testing.T) {
		result, err := util.TruncatePath("../../../../../../..", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty result")
		}
	})

	t.Run("handles symlink paths", func(t *testing.T) {
		// Symlink paths are just strings; TruncatePath does not resolve them
		longPath := "/tmp/very_long_symlink_name/some/deeply/nested/path/here"
		result, err := util.TruncatePath(longPath, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len([]rune(result)) > 20 {
			t.Errorf("result length %d exceeds max width 20: %q", len([]rune(result)), result)
		}
		if !strings.Contains(result, "...") {
			t.Errorf("expected ellipsis in result %q", result)
		}
	})

	// Table-driven tests for exact output matching
	t.Run("exact output matching", func(t *testing.T) {
		tests := []struct {
			name     string
			path     string
			maxWidth int
			expected string
		}{
			{
				"short path unchanged",
				"~/short", 38,
				"~/short",
			},
			{
				"truncate to 30",
				"~/very/long/deeply/nested/project/directory/name", 30,
				"~/very/long/d...irectory/name",
			},
			{
				"truncate to 20",
				"~/this/is/a/really/long/path/that/goes/on/forever", 20,
				"~/this/i.../forever",
			},
			{
				"truncate to 40",
				"~/this/is/a/really/long/path/that/goes/on/forever", 40,
				"~/this/is/a/really...at/goes/on/forever",
			},
			{
				"empty path",
				"", 30,
				"",
			},
			{
				"single char",
				"a", 30,
				"a",
			},
			{
				"exactly at max width",
				"12345678901234567890", 20,
				"12345678901234567890",
			},
			{
				"one over max width",
				"123456789012345678901", 20,
				"12345678...45678901",
			},
			{
				"all dots path short enough",
				"../../../../../../..", 20,
				"../../../../../../..",
			},
			{
				"path with spaces truncated",
				"~/very/long path/with/lots of spaces/deeply nested/directory", 30,
				"~/very/long p...ted/directory",
			},
			{
				"trailing slash truncated",
				"~/very/long/deeply/nested/project/directory/name/", 30,
				"~/very/long/d...rectory/name/",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := util.TruncatePath(tt.path, tt.maxWidth)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("TruncatePath(%q, %d) = %q, want %q",
						tt.path, tt.maxWidth, result, tt.expected)
				}
			})
		}
	})
}
