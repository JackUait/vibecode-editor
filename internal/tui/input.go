package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackuait/ghost-tab/internal/util"
)

type ProjectInputModel struct {
	nameInput    textinput.Model
	pathInput    textinput.Model
	focusName    bool
	name         string
	path         string
	confirmed    bool
	quitting     bool
	err          error
	autocomplete AutocompleteModel
}

func NewProjectInput() ProjectInputModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Project name"
	nameInput.Focus()

	pathInput := textinput.New()
	pathInput.Placeholder = "Project path (e.g., ~/code/project)"

	return ProjectInputModel{
		nameInput:    nameInput,
		pathInput:    pathInput,
		focusName:    true,
		autocomplete: NewAutocomplete(PathSuggestionProvider(8), 8),
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
			if m.autocomplete.ShowSuggestions() {
				m.autocomplete.Dismiss()
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "up":
			if m.autocomplete.ShowSuggestions() && len(m.autocomplete.Suggestions()) > 0 {
				m.autocomplete.MoveUp()
				return m, nil
			}

		case "down":
			if m.autocomplete.ShowSuggestions() && len(m.autocomplete.Suggestions()) > 0 {
				m.autocomplete.MoveDown()
				return m, nil
			}

		case "tab":
			if !m.focusName && m.autocomplete.ShowSuggestions() && len(m.autocomplete.Suggestions()) > 0 {
				accepted := m.autocomplete.AcceptSelected()
				m.pathInput.SetValue(accepted)
				m.autocomplete.SetInput(m.pathInput.Value())
				m.autocomplete.RefreshSuggestions()
				return m, nil
			}

		case "enter":
			if !m.focusName && m.autocomplete.ShowSuggestions() && len(m.autocomplete.Suggestions()) > 0 {
				accepted := m.autocomplete.AcceptSelected()
				m.pathInput.SetValue(accepted)
				m.autocomplete.SetInput(m.pathInput.Value())
				m.autocomplete.RefreshSuggestions()
				return m, nil
			}

			if m.focusName {
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
				m.path = strings.TrimSpace(m.pathInput.Value())
				if m.path == "" {
					m.err = fmt.Errorf("project path cannot be empty")
					return m, nil
				}

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
		current := m.pathInput.Value()
		if current != "" {
			m.autocomplete.SetInput(current)
			m.autocomplete.RefreshSuggestions()
		} else {
			m.autocomplete.Dismiss()
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

	if acView := m.autocomplete.View(); acView != "" {
		b.WriteString(acView)
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
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

// GetPathSuggestions is a convenience wrapper around PathSuggestionProvider.
// Unlike PathSuggestionProvider, it returns nil for empty input (backward-compatible).
func GetPathSuggestions(input string) []string {
	if input == "" {
		return nil
	}
	return PathSuggestionProvider(8)(input)
}
