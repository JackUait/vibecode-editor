package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands ~ to $HOME in path
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home := os.Getenv("HOME")
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		return os.Getenv("HOME")
	}
	return path
}

// TruncatePath shortens a path to maxWidth characters, replacing the middle
// with "...". If the path fits within maxWidth, it is returned unchanged.
// Returns an error if maxWidth is less than 1, matching the bash behavior
// where (maxWidth - 3) / 2 produces a negative substring index.
func TruncatePath(path string, maxWidth int) (string, error) {
	if maxWidth < 1 {
		return "", fmt.Errorf("maxWidth must be positive, got %d", maxWidth)
	}

	runes := []rune(path)
	if len(runes) <= maxWidth {
		return path, nil
	}

	half := (maxWidth - 3) / 2
	if half <= 0 {
		// When half is 0 (e.g. maxWidth=3), just return the ellipsis
		return "...", nil
	}

	start := string(runes[:half])
	end := string(runes[len(runes)-half:])
	return start + "..." + end, nil
}

// ValidatePath checks if path exists and is a directory
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	expanded := ExpandPath(path)
	info, err := os.Stat(expanded)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", expanded)
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", expanded)
	}

	return nil
}
