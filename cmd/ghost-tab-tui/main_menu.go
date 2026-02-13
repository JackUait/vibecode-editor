package main

import (
	"encoding/json"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackuait/ghost-tab/internal/models"
	"github.com/jackuait/ghost-tab/internal/tui"
	"github.com/jackuait/ghost-tab/internal/util"
	"github.com/spf13/cobra"
)

var mainMenuCmd = &cobra.Command{
	Use:   "main-menu",
	Short: "Unified home screen with ghost, projects, and AI tool cycling",
	RunE:  runMainMenu,
}

var (
	mainMenuProjectsFile string
	mainMenuAITool       string
	mainMenuAITools      string
	mainMenuAIToolFile   string
	mainMenuGhostDisplay string
	mainMenuTabTitle     string
	mainMenuUpdateVer    string
	mainMenuSoundName string
)

func init() {
	mainMenuCmd.Flags().StringVar(&mainMenuProjectsFile, "projects-file", "", "Path to projects file")
	mainMenuCmd.MarkFlagRequired("projects-file")
	mainMenuCmd.Flags().StringVar(&mainMenuAITool, "ai-tool", "claude", "Current AI tool name")
	mainMenuCmd.Flags().StringVar(&mainMenuAITools, "ai-tools", "claude", "Comma-separated available tool names")
	mainMenuCmd.Flags().StringVar(&mainMenuAIToolFile, "ai-tool-file", "", "Path to AI tool preference file for persistence")
	mainMenuCmd.Flags().StringVar(&mainMenuGhostDisplay, "ghost-display", "animated", "Ghost display mode (animated, static, none)")
	mainMenuCmd.Flags().StringVar(&mainMenuTabTitle, "tab-title", "full", "Tab title mode (full, project)")
	mainMenuCmd.Flags().StringVar(&mainMenuUpdateVer, "update-version", "", "Optional update notification version")
	mainMenuCmd.Flags().StringVar(&mainMenuSoundName, "sound-name", "", "Sound name for notifications (empty = off)")
	rootCmd.AddCommand(mainMenuCmd)
}

func runMainMenu(cmd *cobra.Command, args []string) error {
	// Ignore SIGHUP so the process survives when the terminal window closes.
	// Bubbletea will detect TTY EOF and shut down gracefully instead.
	signal.Ignore(syscall.SIGHUP)

	projects, err := models.LoadProjects(mainMenuProjectsFile)
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	aiTools := strings.Split(mainMenuAITools, ",")
	for i := range aiTools {
		aiTools[i] = strings.TrimSpace(aiTools[i])
	}

	model := tui.NewMainMenu(projects, aiTools, mainMenuAITool, mainMenuGhostDisplay)
	model.SetTabTitle(mainMenuTabTitle)
	model.SetSoundName(mainMenuSoundName)
	model.SetProjectsFile(mainMenuProjectsFile)
	if mainMenuAIToolFile != "" {
		model.SetAIToolFile(mainMenuAIToolFile)
	}

	ttyOpts, cleanup, err := util.TUITeaOptions()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	defer cleanup()

	opts := append([]tea.ProgramOption{tea.WithAltScreen(), tea.WithMouseCellMotion()}, ttyOpts...)
	p := tea.NewProgram(model, opts...)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	m := finalModel.(*tui.MainMenuModel)
	result := m.Result()

	if result == nil {
		result = &tui.MainMenuResult{
			Action: "quit",
			AITool: m.CurrentAITool(),
		}
	}

	jsonOutput, _ := json.Marshal(result)
	fmt.Println(string(jsonOutput))

	return nil
}
