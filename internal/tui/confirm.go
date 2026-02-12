package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	questionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	hintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type ConfirmDialogModel struct {
	Message   string
	Confirmed bool
	quitting  bool
}

func NewConfirmDialog(message string) ConfirmDialogModel {
	return ConfirmDialogModel{
		Message: message,
	}
}

func (m ConfirmDialogModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEscape:
			m.Confirmed = false
			m.quitting = true
			return m, tea.Quit
		case tea.KeyRunes:
			if len(msg.Runes) == 1 {
				r := TranslateRune(msg.Runes[0])
				switch r {
				case 'y', 'Y':
					m.Confirmed = true
					m.quitting = true
					return m, tea.Quit
				case 'n', 'N':
					m.Confirmed = false
					m.quitting = true
					return m, tea.Quit
				}
			}
		}
	}
	return m, nil
}

func (m ConfirmDialogModel) View() string {
	if m.quitting {
		return ""
	}

	return fmt.Sprintf(
		"%s\n\n%s",
		questionStyle.Render(m.Message),
		hintStyle.Render("[y/n]"),
	)
}
