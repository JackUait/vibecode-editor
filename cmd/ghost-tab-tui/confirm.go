package main

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/jackuait/ghost-tab/internal/tui"
	"github.com/jackuait/ghost-tab/internal/util"
)

var confirmCmd = &cobra.Command{
	Use:   "confirm [message]",
	Short: "Show confirmation dialog",
	Long:  "Shows yes/no confirmation dialog and returns result as JSON",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfirm,
}

func init() {
	rootCmd.AddCommand(confirmCmd)
}

func runConfirm(cmd *cobra.Command, args []string) error {
	tui.ApplyTheme(tui.ThemeForTool(aiToolFlag))

	message := args[0]

	model := tui.NewConfirmDialog(message)

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

	m := finalModel.(tui.ConfirmDialogModel)

	result := map[string]interface{}{
		"confirmed": m.Confirmed,
	}

	jsonOutput, _ := json.Marshal(result)
	fmt.Println(string(jsonOutput))

	return nil
}
