package util

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// OpenTTY opens /dev/tty for direct terminal I/O.
// The caller is responsible for closing the returned file.
func OpenTTY() (*os.File, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("could not open a new TTY: %w", err)
	}
	return tty, nil
}

// TUITeaOptions returns tea.ProgramOption values that route TUI rendering
// to /dev/tty, keeping stdout free for JSON output.
// Returns the options, a cleanup function to close the TTY, and any error.
func TUITeaOptions() ([]tea.ProgramOption, func(), error) {
	tty, err := OpenTTY()
	if err != nil {
		return nil, nil, err
	}

	// Point lipgloss at the real TTY so it detects color capabilities
	// correctly. Without this, lipgloss detects from os.Stdout which is
	// a pipe when bash invokes us via command substitution, resulting in
	// Ascii profile (no colors).
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(tty))

	opts := []tea.ProgramOption{
		tea.WithInput(tty),
		tea.WithOutput(tty),
	}

	cleanup := func() {
		tty.Close()
	}

	return opts, cleanup, nil
}
