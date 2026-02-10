package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type SettingsMenuItem struct {
	ItemTitle   string
	ItemDesc    string
	Action      string
}

func (i SettingsMenuItem) Title() string       { return i.ItemTitle }
func (i SettingsMenuItem) Description() string { return i.ItemDesc }
func (i SettingsMenuItem) FilterValue() string { return i.ItemTitle }

type SettingsMenuModel struct {
	list     list.Model
	selected *SettingsMenuItem
	quitting bool
}

func GetSettingsMenuItems() []SettingsMenuItem {
	return []SettingsMenuItem{
		{ItemTitle: "Add Project", ItemDesc: "Add a new project to the list", Action: "add-project"},
		{ItemTitle: "Delete Project", ItemDesc: "Remove a project from the list", Action: "delete-project"},
		{ItemTitle: "Select AI Tool", ItemDesc: "Choose default AI tool", Action: "select-ai-tool"},
		{ItemTitle: "Manage Features", ItemDesc: "Configure AI tool features", Action: "manage-features"},
		{ItemTitle: "Quit", ItemDesc: "Exit settings menu", Action: "quit"},
	}
}

func NewSettingsMenu() SettingsMenuModel {
	menuItems := GetSettingsMenuItems()
	items := make([]list.Item, len(menuItems))
	for i, item := range menuItems {
		items[i] = item
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Settings"
	l.Styles.Title = titleStyle

	return SettingsMenuModel{
		list: l,
	}
}

func (m SettingsMenuModel) Init() tea.Cmd {
	return nil
}

func (m SettingsMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.selected = &SettingsMenuItem{Action: "quit"}
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if item, ok := m.list.SelectedItem().(SettingsMenuItem); ok {
				m.selected = &item
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SettingsMenuModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

func (m SettingsMenuModel) Selected() *SettingsMenuItem {
	return m.selected
}
