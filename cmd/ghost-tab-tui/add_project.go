package main

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/jackuait/ghost-tab/internal/tui"
	"github.com/jackuait/ghost-tab/internal/util"
)

var addProjectCmd = &cobra.Command{
	Use:   "add-project",
	Short: "Interactive project input form",
	Long:  "Prompts for project name and path with autocomplete, returns as JSON",
	RunE:  runAddProject,
}

func init() {
	rootCmd.AddCommand(addProjectCmd)
}

func runAddProject(cmd *cobra.Command, args []string) error {
	tui.ApplyTheme(tui.ThemeForTool(aiToolFlag))

	model := tui.NewProjectInput()

	ttyOpts, cleanup, err := util.TUITeaOptions()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	defer cleanup()

	p := tea.NewProgram(model, ttyOpts...)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	m := finalModel.(tui.ProjectInputModel)

	var result map[string]interface{}
	if m.Confirmed() {
		result = map[string]interface{}{
			"name":      m.Name(),
			"path":      m.Path(),
			"confirmed": true,
		}
	} else {
		result = map[string]interface{}{
			"confirmed": false,
		}
	}

	jsonOutput, _ := json.Marshal(result)
	fmt.Println(string(jsonOutput))

	return nil
}
