package main

import (
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/jackuait/wisp-deck/internal/allin"
)

func init() {
	rootCmd.AddCommand(newClaudeAllInCommandWithExit(runClaudeRolefixChild, os.Exit))
}

func newClaudeAllInCommand(run claudeRolefixRunner) *cobra.Command {
	return newClaudeAllInCommandWithExit(run, func(int) {})
}

// newClaudeAllInCommandWithExit wraps one Claude launch in a loopback router
// that sends each turn to the subscription its picker row names. The launch
// and exit-code contract itself is runLoopbackWrappedLaunch, shared with
// claude-rolefix so the two never drift apart.
func newClaudeAllInCommandWithExit(run claudeRolefixRunner, exit func(int)) *cobra.Command {
	var settingsPath string
	var env allin.Env
	command := &cobra.Command{
		Use:          "claude-allin --settings PATH -- COMMAND [ARG...]",
		Short:        "Route one Claude launch across every configured subscription",
		Hidden:       true,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, argv []string) error {
			newHandler := func(upstream string) http.Handler {
				return allin.NewHandler(allin.NewResolver(env), upstream)
			}
			return runLoopbackWrappedLaunch(settingsPath, argv, run, exit, newHandler)
		},
	}
	flags := command.Flags()
	flags.StringVar(&settingsPath, "settings", "", "launch settings overlay to point at the router")
	flags.StringVar(&env.AccountsList, "accounts-list", "", "name:dir list of Claude logins")
	flags.StringVar(&env.AccountsDir, "accounts-dir", "", "directory holding each login's config dir")
	flags.StringVar(&env.ConfigsList, "configs-list", "", "name:file list of subscription profiles")
	flags.StringVar(&env.ConfigsDir, "configs-dir", "", "directory holding the profile settings files")
	return command
}
