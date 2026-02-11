package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackuait/ghost-tab/internal/models"
)

// MainMenuResult represents the JSON output when the main menu exits.
type MainMenuResult struct {
	Action string `json:"action"`
	Name   string `json:"name,omitempty"`
	Path   string `json:"path,omitempty"`
	AITool string `json:"ai_tool"`
}

// MenuLayout describes how the ghost and menu are arranged at a given terminal size.
type MenuLayout struct {
	GhostPosition string // "side", "above", "hidden"
	MenuWidth     int    // Always 48
	MenuHeight    int    // Calculated from items
	FirstItemRow  int    // Row offset of first item within menu box
}

// actionNames maps action item offsets to their action strings.
var actionNames = []string{"add-project", "delete-project", "open-once", "plain-terminal"}

// actionLabels maps action items to their display labels.
var actionLabels = []struct {
	shortcut string
	label    string
}{
	{"A", "Add new project"},
	{"D", "Delete a project"},
	{"O", "Open once"},
	{"P", "Plain terminal"},
}

// aiToolDisplayNames maps tool names to their display names.
var aiToolDisplayNames = map[string]string{
	"claude":   "Claude Code",
	"codex":    "Codex CLI",
	"copilot":  "Copilot CLI",
	"opencode": "OpenCode",
}

// AIToolDisplayName returns the display name for the given AI tool.
// Unknown tools return the tool name as-is.
func AIToolDisplayName(tool string) string {
	if name, ok := aiToolDisplayNames[tool]; ok {
		return name
	}
	return tool
}

// MainMenuModel is the Bubbletea model for the unified main menu.
type MainMenuModel struct {
	projects      []models.Project
	aiTools       []string
	selectedAI    int
	selectedItem  int
	ghostDisplay  string
	ghostSleeping bool
	bobStep       int
	sleepTimer    int
	width         int
	height        int
	theme         AIToolTheme
	quitting      bool
	result        *MainMenuResult
	updateVersion string
}

// NewMainMenu creates a new main menu model.
func NewMainMenu(projects []models.Project, aiTools []string, currentAI string, ghostDisplay string) *MainMenuModel {
	selectedAI := 0
	for i, tool := range aiTools {
		if tool == currentAI {
			selectedAI = i
			break
		}
	}

	return &MainMenuModel{
		projects:     projects,
		aiTools:      aiTools,
		selectedAI:   selectedAI,
		selectedItem: 0,
		ghostDisplay: ghostDisplay,
		theme:        ThemeForTool(currentAI),
	}
}

// SetUpdateVersion sets the available update version string.
func (m *MainMenuModel) SetUpdateVersion(version string) {
	m.updateVersion = version
}

// SelectedItem returns the currently selected item index.
func (m *MainMenuModel) SelectedItem() int {
	return m.selectedItem
}

// TotalItems returns the total number of selectable items (projects + 4 actions).
func (m *MainMenuModel) TotalItems() int {
	return len(m.projects) + len(actionNames)
}

// CurrentAITool returns the name of the currently selected AI tool.
func (m *MainMenuModel) CurrentAITool() string {
	if len(m.aiTools) == 0 {
		return ""
	}
	return m.aiTools[m.selectedAI]
}

// CycleAITool cycles the AI tool selection forward ("next") or backward ("prev").
func (m *MainMenuModel) CycleAITool(direction string) {
	n := len(m.aiTools)
	if n <= 1 {
		return
	}
	if direction == "next" {
		m.selectedAI = (m.selectedAI + 1) % n
	} else {
		m.selectedAI = (m.selectedAI - 1 + n) % n
	}
	m.theme = ThemeForTool(m.aiTools[m.selectedAI])
}

// MoveUp moves the selection up by one, wrapping around.
func (m *MainMenuModel) MoveUp() {
	total := m.TotalItems()
	m.selectedItem = (m.selectedItem - 1 + total) % total
}

// MoveDown moves the selection down by one, wrapping around.
func (m *MainMenuModel) MoveDown() {
	total := m.TotalItems()
	m.selectedItem = (m.selectedItem + 1) % total
}

// JumpTo jumps to the given 1-indexed project number.
// Does nothing if n is out of range or beyond the number of projects.
func (m *MainMenuModel) JumpTo(n int) {
	if n < 1 || n > len(m.projects) {
		return
	}
	m.selectedItem = n - 1
}

// SetSize updates the stored terminal dimensions.
func (m *MainMenuModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// GhostDisplay returns the ghost display mode.
func (m *MainMenuModel) GhostDisplay() string {
	return m.ghostDisplay
}

// Result returns the menu result, or nil if the menu has not exited.
func (m *MainMenuModel) Result() *MainMenuResult {
	return m.result
}

// CalculateLayout determines how the ghost and menu should be arranged.
func (m *MainMenuModel) CalculateLayout(width, height int) MenuLayout {
	numProjects := len(m.projects)
	numSeparators := 0
	if numProjects > 0 {
		numSeparators = 1
	}
	totalItems := m.TotalItems()
	menuHeight := 7 + (totalItems * 2) + numSeparators
	menuWidth := 48

	ghostPosition := "hidden"
	// Side layout: width >= 48 + 3 + 28 + 3 = 82
	if width >= menuWidth+3+28+3 {
		ghostPosition = "side"
	} else if height >= menuHeight+15+2 {
		// Above layout: enough vertical space for ghost (15 lines) + gap (2)
		ghostPosition = "above"
	}

	return MenuLayout{
		GhostPosition: ghostPosition,
		MenuWidth:     menuWidth,
		MenuHeight:    menuHeight,
		FirstItemRow:  0,
	}
}

// selectCurrent produces a result for the currently selected item.
func (m *MainMenuModel) selectCurrent() {
	idx := m.selectedItem
	numProjects := len(m.projects)

	if idx < numProjects {
		m.result = &MainMenuResult{
			Action: "select-project",
			Name:   m.projects[idx].Name,
			Path:   m.projects[idx].Path,
			AITool: m.CurrentAITool(),
		}
	} else {
		actionIdx := idx - numProjects
		if actionIdx < len(actionNames) {
			m.result = &MainMenuResult{
				Action: actionNames[actionIdx],
				AITool: m.CurrentAITool(),
			}
		}
	}
	m.quitting = true
}

// setActionResult produces a result for the given action name.
func (m *MainMenuModel) setActionResult(action string) {
	m.result = &MainMenuResult{
		Action: action,
		AITool: m.CurrentAITool(),
	}
	m.quitting = true
}

// Init implements tea.Model. Returns nil for now (Task 5 will add tick commands).
func (m *MainMenuModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. Handles key bindings and window resize.
func (m *MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			m.MoveUp()
			return m, nil
		case tea.KeyDown:
			m.MoveDown()
			return m, nil
		case tea.KeyLeft:
			m.CycleAITool("prev")
			return m, nil
		case tea.KeyRight:
			m.CycleAITool("next")
			return m, nil
		case tea.KeyEnter:
			m.selectCurrent()
			return m, tea.Quit
		case tea.KeyEsc:
			m.setActionResult("quit")
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.setActionResult("quit")
			return m, tea.Quit
		case tea.KeyRunes:
			if len(msg.Runes) == 1 {
				return m.handleRune(msg.Runes[0])
			}
		}
	}

	return m, nil
}

// handleRune processes a single rune keypress.
func (m *MainMenuModel) handleRune(r rune) (tea.Model, tea.Cmd) {
	switch r {
	case 'j':
		m.MoveDown()
		return m, nil
	case 'k':
		m.MoveUp()
		return m, nil
	case 'a', 'A':
		m.setActionResult("add-project")
		return m, tea.Quit
	case 'd', 'D':
		m.setActionResult("delete-project")
		return m, tea.Quit
	case 'o', 'O':
		m.setActionResult("open-once")
		return m, tea.Quit
	case 'p', 'P':
		m.setActionResult("plain-terminal")
		return m, tea.Quit
	case 's', 'S':
		m.setActionResult("settings")
		return m, tea.Quit
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		n := int(r - '0')
		m.JumpTo(n)
		return m, nil
	}
	return m, nil
}

const menuInnerWidth = 46

// shortenHomePath replaces $HOME prefix with ~ for display.
func shortenHomePath(path string) string {
	home := os.Getenv("HOME")
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// renderMenuBox builds the complete menu box string.
func (m *MainMenuModel) renderMenuBox() string {
	dimStyle := lipgloss.NewStyle().Foreground(m.theme.Dim)
	primaryStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
	brightStyle := lipgloss.NewStyle().Foreground(m.theme.Bright)
	brightBoldStyle := lipgloss.NewStyle().Foreground(m.theme.Bright).Bold(true)
	dimPathStyle := lipgloss.NewStyle().Foreground(m.theme.Dim)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	updateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	boldStyle := lipgloss.NewStyle().Bold(true)

	hLine := strings.Repeat("\u2500", menuInnerWidth)
	topBorder := dimStyle.Render("\u250c" + hLine + "\u2510")
	separator := dimStyle.Render("\u251c" + hLine + "\u2524")
	bottomBorder := dimStyle.Render("\u2514" + hLine + "\u2518")
	leftBorder := dimStyle.Render("\u2502")
	rightBorder := dimStyle.Render("\u2502")

	var lines []string

	// Top border
	lines = append(lines, topBorder)

	// Title row
	title := primaryStyle.Render("\u2b21  Ghost Tab")
	aiDisplay := AIToolDisplayName(m.CurrentAITool())
	var aiPart string
	if len(m.aiTools) > 1 {
		aiPart = dimStyle.Render(" \u25c2 ") + brightStyle.Render(aiDisplay) + dimStyle.Render(" \u25b8")
	} else {
		aiPart = " " + brightStyle.Render(aiDisplay)
	}
	titleContent := title + aiPart
	// Pad the title row to inner width. We need to calculate visual width.
	titlePadding := menuInnerWidth - lipgloss.Width(titleContent) - 1 // -1 for leading space
	if titlePadding < 0 {
		titlePadding = 0
	}
	titleRow := leftBorder + " " + titleContent + strings.Repeat(" ", titlePadding) + rightBorder
	lines = append(lines, titleRow)

	// Separator after title
	lines = append(lines, separator)

	// Update notification (if set)
	if m.updateVersion != "" {
		updateMsg := fmt.Sprintf("Update available: %s (brew upgrade ghost-tab)", m.updateVersion)
		updateContent := updateStyle.Render(updateMsg)
		updatePadding := menuInnerWidth - lipgloss.Width(updateContent) - 2 // leading 2 spaces
		if updatePadding < 0 {
			updatePadding = 0
		}
		updateRow := leftBorder + "  " + updateContent + strings.Repeat(" ", updatePadding) + rightBorder
		lines = append(lines, updateRow)
	}

	// Empty line before items
	emptyRow := leftBorder + strings.Repeat(" ", menuInnerWidth) + rightBorder
	lines = append(lines, emptyRow)

	// Project items
	numProjects := len(m.projects)
	for i, proj := range m.projects {
		selected := m.selectedItem == i
		num := fmt.Sprintf("%d", i+1)

		var nameLine string
		var pathLine string

		shortPath := shortenHomePath(proj.Path)

		if selected {
			marker := brightBoldStyle.Render("\u258e")
			nameText := brightBoldStyle.Render(num + "  " + proj.Name)
			// "  ▎ 1  name" -> 2 spaces + marker + space + num + 2 spaces + name
			nameContent := "  " + marker + " " + nameText
			namePadding := menuInnerWidth - lipgloss.Width(nameContent)
			if namePadding < 0 {
				namePadding = 0
			}
			nameLine = leftBorder + nameContent + strings.Repeat(" ", namePadding) + rightBorder

			pathContent := "       " + dimPathStyle.Render(shortPath)
			pathPadding := menuInnerWidth - lipgloss.Width(pathContent)
			if pathPadding < 0 {
				pathPadding = 0
			}
			pathLine = leftBorder + pathContent + strings.Repeat(" ", pathPadding) + rightBorder
		} else {
			nameContent := "    " + num + "  " + proj.Name
			namePadding := menuInnerWidth - len([]rune(nameContent))
			if namePadding < 0 {
				namePadding = 0
			}
			nameLine = leftBorder + nameContent + strings.Repeat(" ", namePadding) + rightBorder

			pathContent := "       " + dimPathStyle.Render(shortPath)
			pathPadding := menuInnerWidth - lipgloss.Width(pathContent)
			if pathPadding < 0 {
				pathPadding = 0
			}
			pathLine = leftBorder + pathContent + strings.Repeat(" ", pathPadding) + rightBorder
		}

		lines = append(lines, nameLine)
		lines = append(lines, pathLine)
	}

	// Separator between projects and actions (only if there are projects)
	if numProjects > 0 {
		lines = append(lines, separator)
	}

	// Action items
	for i, action := range actionLabels {
		actionIdx := numProjects + i
		selected := m.selectedItem == actionIdx

		var actionLine string
		if selected {
			marker := brightBoldStyle.Render("\u258e")
			shortcutText := brightBoldStyle.Render(action.shortcut + "  " + action.label)
			content := "  " + marker + " " + shortcutText
			padding := menuInnerWidth - lipgloss.Width(content)
			if padding < 0 {
				padding = 0
			}
			actionLine = leftBorder + content + strings.Repeat(" ", padding) + rightBorder
		} else {
			shortcutText := boldStyle.Render(action.shortcut)
			content := "    " + shortcutText + "  " + action.label
			padding := menuInnerWidth - lipgloss.Width(content)
			if padding < 0 {
				padding = 0
			}
			actionLine = leftBorder + content + strings.Repeat(" ", padding) + rightBorder
		}

		lines = append(lines, actionLine)
	}

	// Separator before help
	lines = append(lines, separator)

	// Help row
	var helpText string
	if len(m.aiTools) > 1 {
		helpText = "\u2191\u2193 navigate \u2190\u2192 AI tool S settings \u23ce select"
	} else {
		helpText = "\u2191\u2193 navigate S settings \u23ce select"
	}
	helpContent := helpStyle.Render(helpText)
	helpPadding := menuInnerWidth - lipgloss.Width(helpContent) - 1 // -1 for leading space
	if helpPadding < 0 {
		helpPadding = 0
	}
	helpRow := leftBorder + " " + helpContent + strings.Repeat(" ", helpPadding) + rightBorder
	lines = append(lines, helpRow)

	// Bottom border
	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// View implements tea.Model. Renders the full box-drawing menu with optional ghost.
func (m *MainMenuModel) View() string {
	if m.quitting {
		return ""
	}

	menuBox := m.renderMenuBox()

	layout := m.CalculateLayout(m.width, m.height)

	// Determine ghost display
	ghostPosition := layout.GhostPosition
	if m.ghostDisplay == "none" {
		ghostPosition = "hidden"
	}

	switch ghostPosition {
	case "side":
		ghostLines := GhostForTool(m.CurrentAITool(), m.ghostSleeping)
		ghostStr := RenderGhost(ghostLines)
		spacer := strings.Repeat(" ", 3)
		return lipgloss.JoinHorizontal(lipgloss.Top, menuBox, spacer, ghostStr)

	case "above":
		ghostLines := GhostForTool(m.CurrentAITool(), m.ghostSleeping)
		ghostStr := RenderGhost(ghostLines)
		return lipgloss.JoinVertical(lipgloss.Center, ghostStr, "", menuBox)

	default:
		return menuBox
	}
}
