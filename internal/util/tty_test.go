package util

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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

func TestTUITeaOptions_setsLipglossColorProfile(t *testing.T) {
	// Simulate the production scenario: stdout is a pipe (not a TTY).
	// Before calling TUITeaOptions, lipgloss would detect Ascii profile
	// from the piped stdout. After calling TUITeaOptions, the default
	// renderer should be pointed at /dev/tty and detect a real color profile.

	// Force Ascii profile to simulate piped stdout
	prev := lipgloss.DefaultRenderer()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetDefaultRenderer(prev)

	if lipgloss.ColorProfile() != termenv.Ascii {
		t.Fatal("setup failed: color profile should be Ascii")
	}

	opts, cleanup, err := TUITeaOptions()
	if err != nil {
		t.Skipf("Cannot open /dev/tty (expected in CI): %v", err)
	}
	defer cleanup()
	_ = opts

	// After TUITeaOptions, lipgloss should have a color profile better than Ascii
	profile := lipgloss.ColorProfile()
	if profile == termenv.Ascii {
		t.Error("after TUITeaOptions, lipgloss color profile should not be Ascii — it should detect color from /dev/tty")
	}
}
