package main

import "github.com/spf13/cobra"

var aiToolFlag string

var rootCmd = &cobra.Command{
	Use:   "ghost-tab-tui",
	Short: "Interactive TUI components for Ghost Tab",
	Long:  "Provides terminal UI components for Ghost Tab project selector, AI tool picker, and settings menu.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&aiToolFlag, "ai-tool", "claude", "AI tool for theming")
}
