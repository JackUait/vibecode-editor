package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackuait/ghost-tab/internal/util"
)

type ProjectInputModel struct {
	nameInput textinput.Model
	pathInput textinput.Model
	focusName bool
	name      string
	path      string
	confirmed bool
	quitting  bool
	err       error
}

func NewProjectInput() ProjectInputModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Project name"
	nameInput.Focus()

	pathInput := textinput.New()
	pathInput.Placeholder = "Project path (e.g., ~/code/project)"

	return ProjectInputModel{
		nameInput: nameInput,
		pathInput: pathInput,
		focusName: true,
	}
}

func (m ProjectInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ProjectInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if m.focusName {
				// Move to path input
				m.name = strings.TrimSpace(m.nameInput.Value())
				if m.name == "" {
					m.err = fmt.Errorf("project name cannot be empty")
					return m, nil
				}
				m.focusName = false
				m.nameInput.Blur()
				m.pathInput.Focus()
				return m, textinput.Blink
			} else {
				// Validate and confirm
				m.path = strings.TrimSpace(m.pathInput.Value())
				if m.path == "" {
					m.err = fmt.Errorf("project path cannot be empty")
					return m, nil
				}

				// Validate path
				if err := util.ValidatePath(m.path); err != nil {
					m.err = err
					return m, nil
				}

				m.confirmed = true
				m.quitting = true
				return m, tea.Quit
			}

		case "tab":
			// Auto-complete path
			if !m.focusName {
				current := m.pathInput.Value()
				completed := m.autocompletePath(current)
				if completed != current {
					m.pathInput.SetValue(completed)
				}
			}
		}
	}

	if m.focusName {
		m.nameInput, cmd = m.nameInput.Update(msg)
	} else {
		m.pathInput, cmd = m.pathInput.Update(msg)
	}

	return m, cmd
}

func (m ProjectInputModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Add New Project"))
	b.WriteString("\n\n")

	b.WriteString("Project Name:\n")
	b.WriteString(m.nameInput.View())
	b.WriteString("\n\n")

	b.WriteString("Project Path:\n")
	b.WriteString(m.pathInput.View())
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
		m.err = nil // Clear error after showing
	}

	b.WriteString(hintStyle.Render("Tab: autocomplete path | Enter: next/confirm | Esc: cancel"))

	return b.String()
}

func (m ProjectInputModel) Name() string {
	return m.name
}

func (m ProjectInputModel) Path() string {
	return util.ExpandPath(m.path)
}

func (m ProjectInputModel) Confirmed() bool {
	return m.confirmed
}

func (m ProjectInputModel) autocompletePath(input string) string {
	if input == "" {
		return ""
	}

	expanded := util.ExpandPath(input)

	// If input ends with /, complete directory name
	if strings.HasSuffix(input, "/") {
		entries, err := os.ReadDir(expanded)
		if err == nil && len(entries) > 0 {
			// Return first directory
			for _, entry := range entries {
				if entry.IsDir() {
					return input + entry.Name() + "/"
				}
			}
		}
		return input
	}

	// Try to complete partial name
	dir := filepath.Dir(expanded)
	base := filepath.Base(expanded)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return input
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), base) {
			completed := filepath.Join(dir, entry.Name())
			// Convert back to original format (preserve ~)
			if strings.HasPrefix(input, "~") {
				home := os.Getenv("HOME")
				completed = "~" + strings.TrimPrefix(completed, home)
			}
			return completed + "/"
		}
	}

	return input
}
