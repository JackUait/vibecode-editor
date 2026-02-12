package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackuait/ghost-tab/internal/models"
)

// bobTickMsg is sent on each bob animation tick.
type bobTickMsg struct{}

// sleepTickMsg is sent on each sleep timer tick.
type sleepTickMsg struct{}

// bobOffsets is the 14-step sine-wave offset pattern for ghost bobbing.
var bobOffsets = []int{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0}

// BobOffsets returns a copy of the bob animation offset pattern.
func BobOffsets() []int {
	out := make([]int, len(bobOffsets))
	copy(out, bobOffsets)
	return out
}

// MainMenuResult represents the JSON output when the main menu exits.
type MainMenuResult struct {
	Action       string `json:"action"`
	Name         string `json:"name,omitempty"`
	Path         string `json:"path,omitempty"`
	AITool       string `json:"ai_tool"`
	GhostDisplay string `json:"ghost_display,omitempty"`
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
	projects            []models.Project
	aiTools             []string
	selectedAI          int
	selectedItem        int
	ghostDisplay        string
	ghostSleeping       bool
	bobStep             int
	sleepTimer          int
	width               int
	height              int
	theme               AIToolTheme
	quitting            bool
	result              *MainMenuResult
	updateVersion       string
	settingsMode        bool
	settingsSelected    int
	initialGhostDisplay string
	ghostDisplayChanged bool
	zzz                 *ZzzAnimation
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
		projects:            projects,
		aiTools:             aiTools,
		selectedAI:          selectedAI,
		selectedItem:        0,
		ghostDisplay:        ghostDisplay,
		initialGhostDisplay: ghostDisplay,
		theme:               ThemeForTool(currentAI),
		zzz:                 NewZzzAnimation(),
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

// InSettingsMode returns true if the menu is currently showing the settings panel.
func (m *MainMenuModel) InSettingsMode() bool {
	return m.settingsMode
}

// EnterSettings switches the menu to settings mode.
func (m *MainMenuModel) EnterSettings() {
	m.settingsMode = true
	m.settingsSelected = 0
}

// ExitSettings returns from settings mode to the main menu.
func (m *MainMenuModel) ExitSettings() {
	m.settingsMode = false
}

// CycleGhostDisplay cycles through ghost display modes: animated -> static -> none -> animated.
func (m *MainMenuModel) CycleGhostDisplay() {
	switch m.ghostDisplay {
	case "animated":
		m.ghostDisplay = "static"
	case "static":
		m.ghostDisplay = "none"
	default:
		m.ghostDisplay = "animated"
	}
	m.ghostDisplayChanged = m.ghostDisplay != m.initialGhostDisplay
}

// SetSleepTimer sets the sleep inactivity timer to the given number of seconds.
func (m *MainMenuModel) SetSleepTimer(seconds int) { m.sleepTimer = seconds }

// ShouldSleep returns true if the sleep timer has reached the threshold (120s).
func (m *MainMenuModel) ShouldSleep() bool { return m.sleepTimer >= 120 }

// IsSleeping returns true if the ghost is currently in sleeping state.
func (m *MainMenuModel) IsSleeping() bool { return m.ghostSleeping }

// Wake resets the ghost to awake state and clears the sleep timer.
func (m *MainMenuModel) Wake() {
	m.ghostSleeping = false
	m.sleepTimer = 0
	if m.zzz != nil {
		m.zzz.Reset()
	}
}

// BobStep returns the current bob animation step index.
func (m *MainMenuModel) BobStep() int { return m.bobStep }

// ZzzFrame returns the current Zzz animation frame index.
func (m *MainMenuModel) ZzzFrame() int {
	if m.zzz == nil {
		return 0
	}
	return m.zzz.Frame()
}

// NewBobTickMsg creates a bobTickMsg for testing.
func NewBobTickMsg() tea.Msg { return bobTickMsg{} }

// NewSleepTickMsg creates a sleepTickMsg for testing.
func NewSleepTickMsg() tea.Msg { return sleepTickMsg{} }

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

// MapRowToItem maps a terminal Y coordinate (0-indexed click row within the
// menu box) to a selectable item index. Returns -1 if the click is not on a
// valid item row.
func (m *MainMenuModel) MapRowToItem(clickY int) int {
	// Menu box row layout:
	// Row 0: top border
	// Row 1: title row
	// Row 2: separator
	// (optional) update notification row
	// Row 3/4: empty row
	// Then items start
	startRow := 4
	if m.updateVersion != "" {
		startRow++ // update notification takes a row
	}

	numProjects := len(m.projects)

	// Check project items (2 rows each: name line + path line)
	for i := 0; i < numProjects; i++ {
		projectRow := startRow + (i * 2)
		if clickY == projectRow || clickY == projectRow+1 {
			return i
		}
	}

	// After projects: separator row (if projects exist)
	actionStart := startRow + (numProjects * 2)
	if numProjects > 0 {
		actionStart++ // separator between projects and actions
	}

	// Action items (1 row each)
	for i := 0; i < len(actionNames); i++ {
		if clickY == actionStart+i {
			return numProjects + i
		}
	}

	return -1
}

// ghostDisplayForResult returns the ghost display value to include in the result,
// or empty string if unchanged.
func (m *MainMenuModel) ghostDisplayForResult() string {
	if m.ghostDisplayChanged {
		return m.ghostDisplay
	}
	return ""
}

// selectCurrent produces a result for the currently selected item.
func (m *MainMenuModel) selectCurrent() {
	idx := m.selectedItem
	numProjects := len(m.projects)

	if idx < numProjects {
		m.result = &MainMenuResult{
			Action:       "select-project",
			Name:         m.projects[idx].Name,
			Path:         m.projects[idx].Path,
			AITool:       m.CurrentAITool(),
			GhostDisplay: m.ghostDisplayForResult(),
		}
	} else {
		actionIdx := idx - numProjects
		if actionIdx < len(actionNames) {
			m.result = &MainMenuResult{
				Action:       actionNames[actionIdx],
				AITool:       m.CurrentAITool(),
				GhostDisplay: m.ghostDisplayForResult(),
			}
		}
	}
	m.quitting = true
}

// setActionResult produces a result for the given action name.
func (m *MainMenuModel) setActionResult(action string) {
	m.result = &MainMenuResult{
		Action:       action,
		AITool:       m.CurrentAITool(),
		GhostDisplay: m.ghostDisplayForResult(),
	}
	m.quitting = true
}

// bobTickCmd returns a command that sends a bobTickMsg after 180ms.
func (m *MainMenuModel) bobTickCmd() tea.Cmd {
	return tea.Tick(180*time.Millisecond, func(t time.Time) tea.Msg {
		return bobTickMsg{}
	})
}

// sleepTickCmd returns a command that sends a sleepTickMsg after 1 second.
func (m *MainMenuModel) sleepTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return sleepTickMsg{}
	})
}

// Init implements tea.Model. Starts animation ticks when in animated mode.
func (m *MainMenuModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.ghostDisplay == "animated" {
		cmds = append(cmds, m.bobTickCmd())
		cmds = append(cmds, m.sleepTickCmd())
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model. Handles key bindings, window resize, and animation ticks.
func (m *MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case bobTickMsg:
		if m.ghostDisplay == "animated" {
			m.bobStep = (m.bobStep + 1) % len(BobOffsets())
			if m.ghostSleeping && m.zzz != nil {
				m.zzz.Tick()
			}
			return m, m.bobTickCmd()
		}
		return m, nil

	case sleepTickMsg:
		if m.ghostDisplay == "animated" && !m.ghostSleeping {
			m.sleepTimer++
			if m.sleepTimer >= 120 {
				m.ghostSleeping = true
			}
			return m, m.sleepTickCmd()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.MouseMsg:
		// Reset sleep state on any mouse activity
		m.Wake()

		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			item := m.MapRowToItem(msg.Y)
			if item >= 0 {
				if m.selectedItem == item {
					// Already selected, activate (double-click-like behavior)
					m.selectCurrent()
					return m, tea.Quit
				}
				m.selectedItem = item
			}
		}
		return m, nil

	case tea.KeyMsg:
		// Reset sleep state on any keypress
		m.Wake()

		// Settings mode intercepts all key handling
		if m.settingsMode {
			return m.updateSettings(msg)
		}

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
		m.settingsMode = true
		m.settingsSelected = 0
		return m, nil
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		n := int(r - '0')
		m.JumpTo(n)
		return m, nil
	}
	return m, nil
}

// updateSettings handles key events while in settings mode.
func (m *MainMenuModel) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.settingsMode = false
		return m, nil
	case tea.KeyCtrlC:
		m.settingsMode = false
		m.setActionResult("quit")
		return m, tea.Quit
	case tea.KeyEnter:
		// Activate current settings item
		if m.settingsSelected == 0 {
			m.CycleGhostDisplay()
		}
		return m, nil
	case tea.KeyUp:
		// Future extensibility: navigate settings items
		return m, nil
	case tea.KeyDown:
		return m, nil
	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'b', 'B':
				m.settingsMode = false
				return m, nil
			case 'a', 'A':
				m.CycleGhostDisplay()
				return m, nil
			case 'j':
				return m, nil
			case 'k':
				return m, nil
			}
		}
	}
	return m, nil
}

// ghostDisplayLabel returns a capitalized display label for the ghost display mode.
func ghostDisplayLabel(mode string) string {
	switch mode {
	case "animated":
		return "Animated"
	case "static":
		return "Static"
	case "none":
		return "None"
	default:
		return mode
	}
}

// renderSettingsBox builds the settings panel box string.
func (m *MainMenuModel) renderSettingsBox() string {
	dimStyle := lipgloss.NewStyle().Foreground(m.theme.Dim)
	primaryStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
	brightBoldStyle := lipgloss.NewStyle().Foreground(m.theme.Bright).Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// State color depends on ghost display mode
	var stateColor lipgloss.Color
	switch m.ghostDisplay {
	case "animated":
		stateColor = lipgloss.Color("114") // green
	case "static":
		stateColor = lipgloss.Color("220") // yellow
	default:
		stateColor = lipgloss.Color("241") // gray
	}
	stateStyle := lipgloss.NewStyle().Foreground(stateColor)

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
	title := primaryStyle.Render("\u2b21  Settings")
	titlePadding := menuInnerWidth - lipgloss.Width(title) - 1
	if titlePadding < 0 {
		titlePadding = 0
	}
	titleRow := leftBorder + " " + title + strings.Repeat(" ", titlePadding) + rightBorder
	lines = append(lines, titleRow)

	// Separator after title
	lines = append(lines, separator)

	// Empty row
	emptyRow := leftBorder + strings.Repeat(" ", menuInnerWidth) + rightBorder
	lines = append(lines, emptyRow)

	// Ghost Display item
	label := "Ghost Display"
	stateText := "[" + ghostDisplayLabel(m.ghostDisplay) + "]"
	if m.settingsSelected == 0 {
		marker := brightBoldStyle.Render("\u258e")
		labelText := brightBoldStyle.Render(label)
		stateRendered := stateStyle.Render(stateText)
		content := "  " + marker + " " + labelText + "    " + stateRendered
		padding := menuInnerWidth - lipgloss.Width(content)
		if padding < 0 {
			padding = 0
		}
		itemRow := leftBorder + content + strings.Repeat(" ", padding) + rightBorder
		lines = append(lines, itemRow)
	} else {
		stateRendered := stateStyle.Render(stateText)
		content := "    " + label + "    " + stateRendered
		padding := menuInnerWidth - lipgloss.Width(content)
		if padding < 0 {
			padding = 0
		}
		itemRow := leftBorder + content + strings.Repeat(" ", padding) + rightBorder
		lines = append(lines, itemRow)
	}

	// Empty row
	lines = append(lines, emptyRow)

	// Separator before help
	lines = append(lines, separator)

	// Help row
	helpText := "A cycle  B back  Esc close"
	helpContent := helpStyle.Render(helpText)
	helpPadding := menuInnerWidth - lipgloss.Width(helpContent) - 1
	if helpPadding < 0 {
		helpPadding = 0
	}
	helpRow := leftBorder + " " + helpContent + strings.Repeat(" ", helpPadding) + rightBorder
	lines = append(lines, helpRow)

	// Bottom border
	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
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

	var menuBox string
	if m.settingsMode {
		menuBox = m.renderSettingsBox()
	} else {
		menuBox = m.renderMenuBox()
	}

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
		if m.ghostDisplay == "animated" && bobOffsets[m.bobStep] == 1 {
			ghostStr = "\n" + ghostStr
		}
		if m.ghostSleeping && m.zzz != nil {
			zzzColor := AnsiFromThemeColor(m.theme.SleepAccent)
			ghostStr += "\n" + m.zzz.ViewColored(zzzColor)
		}
		spacer := strings.Repeat(" ", 3)
		return lipgloss.JoinHorizontal(lipgloss.Top, menuBox, spacer, ghostStr)

	case "above":
		ghostLines := GhostForTool(m.CurrentAITool(), m.ghostSleeping)
		ghostStr := RenderGhost(ghostLines)
		if m.ghostDisplay == "animated" && bobOffsets[m.bobStep] == 1 {
			ghostStr = "\n" + ghostStr
		}
		if m.ghostSleeping && m.zzz != nil {
			zzzColor := AnsiFromThemeColor(m.theme.SleepAccent)
			ghostStr += "\n" + m.zzz.ViewColored(zzzColor)
		}
		return lipgloss.JoinVertical(lipgloss.Center, ghostStr, "", menuBox)

	default:
		return menuBox
	}
}
