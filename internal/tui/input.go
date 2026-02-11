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
	nameInput       textinput.Model
	pathInput       textinput.Model
	focusName       bool
	name            string
	path            string
	confirmed       bool
	quitting        bool
	err             error
	suggestions     []string
	sugSelected     int
	showSuggestions bool
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
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "esc":
			if m.showSuggestions {
				m.showSuggestions = false
				m.suggestions = nil
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "up":
			if m.showSuggestions && len(m.suggestions) > 0 {
				m.sugSelected--
				if m.sugSelected < 0 {
					m.sugSelected = len(m.suggestions) - 1
				}
				return m, nil
			}

		case "down":
			if m.showSuggestions && len(m.suggestions) > 0 {
				m.sugSelected++
				if m.sugSelected >= len(m.suggestions) {
					m.sugSelected = 0
				}
				return m, nil
			}

		case "tab":
			if !m.focusName && m.showSuggestions && len(m.suggestions) > 0 {
				// Accept selected suggestion
				m.pathInput.SetValue(m.suggestions[m.sugSelected])
				// Refresh suggestions for the new path
				m.suggestions = GetPathSuggestions(m.pathInput.Value())
				m.sugSelected = 0
				m.showSuggestions = len(m.suggestions) > 0
				return m, nil
			}

		case "enter":
			if !m.focusName && m.showSuggestions && len(m.suggestions) > 0 {
				// Accept selected suggestion
				m.pathInput.SetValue(m.suggestions[m.sugSelected])
				// Refresh suggestions for the new path
				m.suggestions = GetPathSuggestions(m.pathInput.Value())
				m.sugSelected = 0
				m.showSuggestions = len(m.suggestions) > 0
				return m, nil
			}

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
		}
	}

	if m.focusName {
		m.nameInput, cmd = m.nameInput.Update(msg)
	} else {
		m.pathInput, cmd = m.pathInput.Update(msg)
		// Update suggestions on every keystroke while in path input
		current := m.pathInput.Value()
		if current != "" {
			m.suggestions = GetPathSuggestions(current)
			m.sugSelected = 0
			m.showSuggestions = len(m.suggestions) > 0
		} else {
			m.suggestions = nil
			m.showSuggestions = false
		}
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
	b.WriteString("\n")

	if m.showSuggestions && len(m.suggestions) > 0 {
		b.WriteString(m.renderSuggestions())
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
		m.err = nil // Clear error after showing
	}

	b.WriteString(hintStyle.Render("Tab: autocomplete path | Enter: next/confirm | Esc: cancel"))

	return b.String()
}

func (m ProjectInputModel) renderSuggestions() string {
	borderColor := lipgloss.Color("241")
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	selectedStyle := lipgloss.NewStyle().Reverse(true)

	// Find the maximum suggestion width for the box
	maxWidth := 0
	for _, s := range m.suggestions {
		if len(s) > maxWidth {
			maxWidth = len(s)
		}
	}
	// Minimum box width and padding
	if maxWidth < 20 {
		maxWidth = 20
	}
	boxWidth := maxWidth + 2 // 1 space padding on each side

	var b strings.Builder

	// Top border
	b.WriteString(borderStyle.Render("┌" + strings.Repeat("─", boxWidth) + "┐"))
	b.WriteString("\n")

	// Suggestion rows
	for i, s := range m.suggestions {
		padded := s + strings.Repeat(" ", boxWidth-2-len(s))
		var content string
		if i == m.sugSelected {
			content = selectedStyle.Render(" " + padded + " ")
		} else {
			content = " " + padded + " "
		}
		b.WriteString(borderStyle.Render("│") + content + borderStyle.Render("│"))
		b.WriteString("\n")
	}

	// Bottom border
	b.WriteString(borderStyle.Render("└" + strings.Repeat("─", boxWidth) + "┘"))
	b.WriteString("\n")

	// Help text
	b.WriteString(hintStyle.Render("↑↓ navigate  ⏎ complete  Esc cancel"))

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

// GetPathSuggestions returns matching directory entries for a partial path input.
// It handles tilde expansion, case-insensitive prefix matching, and returns
// up to 8 suggestions with trailing slashes for directories.
func GetPathSuggestions(input string) []string {
	if input == "" {
		return nil
	}

	expanded := util.ExpandPath(input)

	var dir string
	var prefix string

	if strings.HasSuffix(input, "/") {
		// List contents of the directory
		dir = expanded
		prefix = ""
	} else {
		// Match partial filename in parent directory
		dir = filepath.Dir(expanded)
		prefix = filepath.Base(expanded)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	const maxSuggestions = 8
	var suggestions []string

	lowerPrefix := strings.ToLower(prefix)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip hidden directories
		if strings.HasPrefix(name, ".") {
			continue
		}
		if prefix == "" || strings.HasPrefix(strings.ToLower(name), lowerPrefix) {
			// Build the full suggestion path preserving original format
			var suggestion string
			if strings.HasSuffix(input, "/") {
				suggestion = input + name + "/"
			} else {
				parentInput := input[:len(input)-len(filepath.Base(input))]
				suggestion = parentInput + name + "/"
			}
			suggestions = append(suggestions, suggestion)
			if len(suggestions) >= maxSuggestions {
				break
			}
		}
	}

	return suggestions
}
