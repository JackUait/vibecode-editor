package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/jackuait/wisp-deck/internal/allin"
	"github.com/jackuait/wisp-deck/internal/rolefix"
)

func init() {
	rootCmd.AddCommand(newClaudeAllInCommandWithExit(runClaudeRolefixChild, os.Exit))
}

func newClaudeAllInCommand(run claudeRolefixRunner) *cobra.Command {
	return newClaudeAllInCommandWithExit(run, func(int) {})
}

// newClaudeAllInCommandWithExit wraps one Claude launch in a loopback router
// that sends each turn to the subscription its picker row names.
//
// Nothing here may cost the user their session: an unreadable overlay, or one
// declaring no endpoint, runs the child exactly as it was going to run anyway.
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
			if run == nil {
				return errors.New("child runner is unavailable")
			}
			finish := func(err error) error {
				var code exitCodeError
				if errors.As(err, &code) {
					// Claude's own exit status is the session's; surfacing it as
					// a cobra error would print a banner and lose the code.
					if exit != nil {
						exit(int(code))
					}
					return nil
				}
				return err
			}
			upstream, err := rolefix.UpstreamFromSettings(settingsPath)
			if err != nil {
				return finish(run(argv))
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return finish(run(argv))
			}
			defer func() { _ = listener.Close() }()

			server := &http.Server{Handler: allin.NewHandler(allin.NewResolver(env), upstream)}
			go func() { _ = server.Serve(listener) }()
			defer func() { _ = server.Close() }()

			routerURL := fmt.Sprintf("http://%s", listener.Addr().String())
			if err := rolefix.PointSettingsAt(settingsPath, routerURL); err != nil {
				// The overlay still names the real endpoint, so the session is
				// no worse off than without this wrapper.
				return finish(run(argv))
			}
			return finish(run(argv))
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
