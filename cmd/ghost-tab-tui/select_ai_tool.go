package main

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/jackuait/ghost-tab/internal/models"
	"github.com/jackuait/ghost-tab/internal/tui"
)

var selectAIToolCmd = &cobra.Command{
	Use:   "select-ai-tool",
	Short: "Interactive AI tool selector",
	Long:  "Shows available AI tools and returns selected tool as JSON",
	RunE:  runSelectAITool,
}

func init() {
	rootCmd.AddCommand(selectAIToolCmd)
}

func runSelectAITool(cmd *cobra.Command, args []string) error {
	tools := models.DetectAITools()

	model := tui.NewAIToolSelector(tools)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	m := finalModel.(tui.AIToolSelectorModel)
	selected := m.Selected()

	var result map[string]interface{}
	if selected != nil {
		result = map[string]interface{}{
			"tool":     selected.Name,
			"command":  selected.Command,
			"selected": true,
		}
	} else {
		result = map[string]interface{}{"selected": false}
	}

	jsonOutput, _ := json.Marshal(result)
	fmt.Println(string(jsonOutput))

	return nil
}
