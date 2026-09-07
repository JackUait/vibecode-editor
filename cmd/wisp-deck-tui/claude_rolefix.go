package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/jackuait/wisp-deck/internal/rolefix"
)

type claudeRolefixRunner func(argv []string) error

func init() {
	rootCmd.AddCommand(newClaudeRolefixCommandWithExit(runClaudeRolefixChild, os.Exit))
}

// exitCodeError carries a child's exit status back to the command without
// becoming a cobra error, which would print a banner into the pane.
type exitCodeError int

func (e exitCodeError) Error() string { return fmt.Sprintf("exit status %d", int(e)) }

// newClaudeRolefixCommand is the test seam: it swallows the child's exit code
// instead of ending the process.
func newClaudeRolefixCommand(run claudeRolefixRunner) *cobra.Command {
	return newClaudeRolefixCommandWithExit(run, func(int) {})
}

// newClaudeRolefixCommandWithExit wraps one Claude launch in a loopback proxy
// that repairs the message roles a strict Anthropic endpoint rejects, and points
// the session's settings overlay at it.
func newClaudeRolefixCommandWithExit(run claudeRolefixRunner, exit func(int)) *cobra.Command {
	var settingsPath string
	command := &cobra.Command{
		Use:          "claude-rolefix --settings PATH -- COMMAND [ARG...]",
		Short:        "Route one Claude launch through the message-role repair proxy",
		Hidden:       true,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, argv []string) error {
			return runLoopbackWrappedLaunch(settingsPath, argv, run, exit, rolefix.NewHandler, nil)
		},
	}
	command.Flags().StringVar(&settingsPath, "settings", "",
		"Path to the launch settings overlay to point at the proxy")
	return command
}

// runLoopbackWrappedLaunch is the launch shape claude-rolefix and claude-allin
// both need: point the settings overlay at a fresh loopback listener serving
// newHandler(upstream), then run the child and propagate its exit code.
//
// Nothing here may cost the user their session: an overlay that cannot be
// read, declares no endpoint, or already points somewhere local, a listener
// that cannot bind, or a failed overlay rewrite, all fall through to running
// the child exactly as it was going to run anyway.
//
// cleanup (nil for claude-rolefix, the ChatGPT bridge's Close for claude-allin)
// runs on EVERY route out — including the fall-through ones that leave the
// launch unwrapped — and it runs BEFORE the exit code is propagated. exit is os.Exit in production, which runs
// no deferred function — so a `defer cleanup()` here would leak a 220MB Codex
// app-server on the one route that actually happens.
func runLoopbackWrappedLaunch(settingsPath string, argv []string, run claudeRolefixRunner, exit func(int), newHandler func(upstream string) http.Handler, cleanup func()) error {
	if run == nil {
		if cleanup != nil {
			cleanup()
		}
		return errors.New("child runner is unavailable")
	}
	finish := func(err error) error {
		if cleanup != nil {
			cleanup()
		}
		var code exitCodeError
		if errors.As(err, &code) {
			// The child's own exit status is the session's; surfacing it as
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

	server := &http.Server{Handler: newHandler(upstream)}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Close() }()

	proxyURL := fmt.Sprintf("http://%s", listener.Addr().String())
	if err := rolefix.PointSettingsAt(settingsPath, proxyURL); err != nil {
		// The overlay still names the real endpoint, so the session is
		// no worse off than without this wrapper.
		return finish(run(argv))
	}
	return finish(run(argv))
}

func runClaudeRolefixChild(argv []string) error {
	child := exec.Command(argv[0], argv[1:]...)
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := child.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitCodeError(exitErr.ExitCode())
	}
	return err
}
