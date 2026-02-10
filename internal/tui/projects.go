package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackuait/ghost-tab/internal/models"
)

var (
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
)

type projectItem struct {
	project models.Project
}

func (i projectItem) Title() string       { return i.project.Name }
func (i projectItem) Description() string { return i.project.Path }
func (i projectItem) FilterValue() string { return i.project.Name }

type ProjectSelectorModel struct {
	list     list.Model
	filter   textinput.Model
	projects []models.Project
	selected *models.Project
	quitting bool
}

func NewProjectSelector(projects []models.Project) ProjectSelectorModel {
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{project: p}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Project"
	l.Styles.Title = titleStyle

	return ProjectSelectorModel{
		list:     l,
		projects: projects,
	}
}

func (m ProjectSelectorModel) Init() tea.Cmd {
	return nil
}

func (m ProjectSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if item, ok := m.list.SelectedItem().(projectItem); ok {
				m.selected = &item.project
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ProjectSelectorModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

func (m ProjectSelectorModel) Selected() *models.Project {
	return m.selected
}

// FilterProjects filters projects by name
func FilterProjects(projects []models.Project, filter string) []models.Project {
	if filter == "" {
		return projects
	}

	var filtered []models.Project
	filter = strings.ToLower(filter)

	for _, p := range projects {
		if strings.Contains(strings.ToLower(p.Name), filter) {
			filtered = append(filtered, p)
		}
	}

	return filtered
}
