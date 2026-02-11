package util

import (
	"os"
	"testing"
)

func TestOpenTTY(t *testing.T) {
	tty, err := OpenTTY()
	if err != nil {
		// In CI or non-interactive environments, /dev/tty may not be available
		t.Skipf("Cannot open /dev/tty (expected in CI): %v", err)
	}
	defer tty.Close()

	// Verify it's a valid file
	stat, err := tty.Stat()
	if err != nil {
		t.Fatalf("Failed to stat TTY: %v", err)
	}

	// /dev/tty should be a character device
	if stat.Mode()&os.ModeCharDevice == 0 {
		t.Errorf("Expected character device, got mode %v", stat.Mode())
	}
}

func TestTUITeaOptions_returnsTwoOptions(t *testing.T) {
	opts, cleanup, err := TUITeaOptions()
	if err != nil {
		t.Skipf("Cannot open /dev/tty (expected in CI): %v", err)
	}
	defer cleanup()

	// Should return exactly 2 options (WithInput and WithOutput)
	if len(opts) != 2 {
		t.Errorf("Expected 2 tea options, got %d", len(opts))
	}
}
