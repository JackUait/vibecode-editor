package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackuait/ghost-tab/internal/models"
)

type aiToolItem struct {
	tool models.AITool
}

func (i aiToolItem) Title() string       { return i.tool.String() }
func (i aiToolItem) Description() string { return i.tool.Command }
func (i aiToolItem) FilterValue() string { return i.tool.Name }

type AIToolSelectorModel struct {
	list     list.Model
	tools    []models.AITool
	selected *models.AITool
	quitting bool
}

func NewAIToolSelector(tools []models.AITool) AIToolSelectorModel {
	items := make([]list.Item, len(tools))
	for i, t := range tools {
		items[i] = aiToolItem{tool: t}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select AI Tool"
	l.Styles.Title = titleStyle

	return AIToolSelectorModel{
		list:  l,
		tools: tools,
	}
}

func (m AIToolSelectorModel) Init() tea.Cmd {
	return nil
}

func (m AIToolSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if item, ok := m.list.SelectedItem().(aiToolItem); ok {
				if item.tool.Installed {
					m.selected = &item.tool
				}
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m AIToolSelectorModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

func (m AIToolSelectorModel) Selected() *models.AITool {
	return m.selected
}
