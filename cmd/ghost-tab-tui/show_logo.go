package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/jackuait/ghost-tab/internal/tui"
)

var showLogoCmd = &cobra.Command{
	Use:   "show-logo",
	Short: "Display animated Ghost Tab logo",
	Long:  "Shows an animated ASCII art logo (no JSON output)",
	RunE:  runShowLogo,
}

func init() {
	rootCmd.AddCommand(showLogoCmd)
}

func runShowLogo(cmd *cobra.Command, args []string) error {
	model := tui.NewLogo()
	p := tea.NewProgram(model)

	_, err := p.Run()
	return err
}
