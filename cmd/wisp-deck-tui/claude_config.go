package main

import (
	"fmt"
	"os"

	"github.com/jackuait/wisp-deck/internal/allin"
	"github.com/jackuait/wisp-deck/internal/claudeconfig"
	"github.com/jackuait/wisp-deck/internal/opencodeconfig"
	"github.com/spf13/cobra"
)

var (
	ccList             string
	ccDir              string
	ccPointer          string
	ccFile             string
	ccName             string
	ccAccountsList     string
	ccAccountsDir      string
	ccDefaultLabelFile string
)

// ensureAllInFromCLI refreshes the All-In profile for add/delete, the two
// mutations reachable from the legacy "Manage Claude configs" menu
// (lib/config-tui.sh) — the Subscription modal's own add/delete call the same
// gate directly. A failure here must never fail the config mutation the user
// asked for, exactly like the sibling ensure-budget/ensure-watchdog sweeps
// bin/wisp-deck already runs with `|| true`.
func ensureAllInFromCLI() {
	_ = allin.EnsureProfileIfEligible(allin.Env{
		AccountsList:     ccAccountsList,
		AccountsDir:      ccAccountsDir,
		ConfigsList:      ccList,
		ConfigsDir:       ccDir,
		DefaultLabelFile: ccDefaultLabelFile,
	})
}

func syncOpenCode() {
	if ccList == "" || ccDir == "" {
		return
	}
	home, _ := os.UserHomeDir()
	_ = opencodeconfig.Sync(opencodeconfig.Inputs{
		ListFile:    ccList,
		ConfigsDir:  ccDir,
		PointerFile: ccPointer,
		Home:        home,
	})
}

var claudeConfigCmd = &cobra.Command{
	Use:   "claude-config",
	Short: "Create, rename, and delete Claude config files",
	Long:  "Mutation commands for Claude settings configs; the single source of truth shared by the inline TUI and the config menu",
}

var claudeConfigAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new Claude config and print its filename",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, err := claudeconfig.Add(ccList, ccDir, ccName)
		if err != nil {
			return err
		}
		syncOpenCode()
		// A bare config carries no key, so it is never ConfigReady and this
		// can never cross the threshold by itself today — wired anyway so an
		// add that DOES become a source (a future default-populated provider)
		// is covered without a second change.
		ensureAllInFromCLI()
		fmt.Fprintln(cmd.OutOrStdout(), file)
		return nil
	},
}

var claudeConfigRenameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Rename an existing Claude config",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := claudeconfig.Rename(ccList, ccFile, ccName); err != nil {
			return err
		}
		syncOpenCode()
		return nil
	},
}

var claudeConfigDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a Claude config and clear the pointer if it was active",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read before deleting: once gone, ProfileFile can no longer say the
		// removed file WAS the router profile, and refreshing right after
		// deleting it would recreate it on the same keystroke — the source
		// count that made it eligible does not change just because the
		// profile itself is what got removed.
		deletingAllIn := ccFile != "" && ccFile == allin.ProfileFile(ccList)
		if err := claudeconfig.Delete(ccList, ccDir, ccPointer, ccFile); err != nil {
			return err
		}
		syncOpenCode()
		if !deletingAllIn {
			ensureAllInFromCLI()
		}
		return nil
	},
}

// A profile written before the window was declared is exactly the one that can
// strand a session, and the installer only copies a default when the file is
// absent — so existing profiles are reachable only by an explicit sweep.
var claudeConfigEnsureBudgetCmd = &cobra.Command{
	Use:   "ensure-budget",
	Short: "Backfill each config's real context window and print how many changed",
	RunE: func(cmd *cobra.Command, args []string) error {
		changed, err := claudeconfig.EnsureContextBudgetAll(ccDir)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), changed)
		return nil
	},
}

// A user-configured endpoint makes no promise to keep its stream warm, and a
// profile already on disk is never re-copied from defaults, so the watchdog it
// was written with can only be disarmed here.
//
// Both tiers are swept together because both are launch-time env keys with the
// same repair story. They differ in scope: the byte tier is wrong only for a
// self-hosted endpoint, while the event tier is wrong for every endpoint
// wisp-deck configures.
var claudeConfigEnsureWatchdogCmd = &cobra.Command{
	Use:   "ensure-watchdog",
	Short: "Disarm the stall watchdogs on subscription configs and print how many changed",
	RunE: func(cmd *cobra.Command, args []string) error {
		changed, err := claudeconfig.EnsureByteWatchdogAll(ccDir)
		if err != nil {
			return err
		}
		streamChanged, err := claudeconfig.EnsureStreamWatchdogAll(ccDir)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), changed+streamChanged)
		return nil
	},
}

// The All-In profile's picker rows name a login or a provider config, both of
// which come and go — so unlike add/rename/delete, this rebuilds the picker on
// every call rather than mutating once. Same repair story as ensure-budget:
// the profile is never re-copied from defaults once it exists on disk.
func newEnsureAllInCommand() *cobra.Command {
	var env allin.Env
	command := &cobra.Command{
		Use:   "ensure-allin",
		Short: "Create or refresh the All-In profile's model picker",
		RunE: func(_ *cobra.Command, _ []string) error {
			// Shared with the TUI's own login/subscription add and delete, so
			// the create-vs-refresh gate cannot drift between callers.
			return allin.EnsureProfileIfEligible(env)
		},
	}
	// The flag names match claude-allin's, which reads the same two files: this
	// command already takes --accounts-list/--accounts-dir, so a bare
	// --list/--dir beside them names the configs pair only by omission.
	flags := command.Flags()
	flags.StringVar(&env.ConfigsDir, "configs-dir", "", "directory holding the profile settings files")
	flags.StringVar(&env.ConfigsList, "configs-list", "", "name:file list of subscription profiles")
	flags.StringVar(&env.AccountsList, "accounts-list", "", "name:dir list of Claude logins")
	flags.StringVar(&env.AccountsDir, "accounts-dir", "", "directory holding each login's config dir")
	flags.StringVar(&env.DefaultLabelFile, "default-label-file", "",
		"file holding the implicit Default login's custom tag (optional)")
	return command
}

func init() {
	claudeConfigAddCmd.Flags().StringVar(&ccList, "list", "", "Path to configs list (name:file)")
	claudeConfigAddCmd.Flags().StringVar(&ccDir, "dir", "", "Path to configs directory")
	claudeConfigAddCmd.Flags().StringVar(&ccName, "name", "", "Display name for the new config")
	claudeConfigAddCmd.Flags().StringVar(&ccPointer, "pointer", "", "Path to active config pointer file")
	claudeConfigAddCmd.Flags().StringVar(&ccAccountsList, "accounts-list", "", "name:dir list of Claude logins (for the All-In gate)")
	claudeConfigAddCmd.Flags().StringVar(&ccAccountsDir, "accounts-dir", "", "directory holding each login's config dir (for the All-In gate)")
	claudeConfigAddCmd.Flags().StringVar(&ccDefaultLabelFile, "default-label-file", "",
		"file holding the implicit Default login's custom tag (for the All-In gate, optional)")

	claudeConfigRenameCmd.Flags().StringVar(&ccList, "list", "", "Path to configs list (name:file)")
	claudeConfigRenameCmd.Flags().StringVar(&ccFile, "file", "", "Filename of the config to rename")
	claudeConfigRenameCmd.Flags().StringVar(&ccName, "name", "", "New display name")
	claudeConfigRenameCmd.Flags().StringVar(&ccDir, "dir", "", "Path to configs directory")
	claudeConfigRenameCmd.Flags().StringVar(&ccPointer, "pointer", "", "Path to active config pointer file")

	claudeConfigDeleteCmd.Flags().StringVar(&ccList, "list", "", "Path to configs list (name:file)")
	claudeConfigDeleteCmd.Flags().StringVar(&ccDir, "dir", "", "Path to configs directory")
	claudeConfigDeleteCmd.Flags().StringVar(&ccPointer, "pointer", "", "Path to active config pointer file")
	claudeConfigDeleteCmd.Flags().StringVar(&ccFile, "file", "", "Filename of the config to delete")
	claudeConfigDeleteCmd.Flags().StringVar(&ccAccountsList, "accounts-list", "", "name:dir list of Claude logins (for the All-In gate)")
	claudeConfigDeleteCmd.Flags().StringVar(&ccAccountsDir, "accounts-dir", "", "directory holding each login's config dir (for the All-In gate)")
	claudeConfigDeleteCmd.Flags().StringVar(&ccDefaultLabelFile, "default-label-file", "",
		"file holding the implicit Default login's custom tag (for the All-In gate, optional)")

	claudeConfigEnsureBudgetCmd.Flags().StringVar(&ccDir, "dir", "", "Path to configs directory")
	claudeConfigEnsureWatchdogCmd.Flags().StringVar(&ccDir, "dir", "", "Path to configs directory")

	claudeConfigCmd.AddCommand(claudeConfigAddCmd, claudeConfigRenameCmd, claudeConfigDeleteCmd,
		claudeConfigEnsureBudgetCmd, claudeConfigEnsureWatchdogCmd, newEnsureAllInCommand())
	rootCmd.AddCommand(claudeConfigCmd)
}
