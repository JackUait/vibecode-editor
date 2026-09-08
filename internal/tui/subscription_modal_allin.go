package tui

import "github.com/jackuait/wisp-deck/internal/allin"

// ensureAllIn refreshes the All-In profile after something that can change the
// source count or the roster it offers (a subscription key saved, a profile
// deleted or disabled, a login added or removed) — and also from
// openSubscriptionModal, a read rather than a mutation, so a machine whose
// subscriptions were connected before this feature existed (or a session that
// never mutates anything) still gets the profile the moment the user looks at
// Subscriptions. It shares the exact create-vs-refresh gate the CLI's own
// ensure-allin and add/delete commands use, so the surfaces cannot drift on
// what counts as "eligible".
//
// Errors are swallowed: this is a convenience layered on top of the mutation
// the user actually asked for, and that action must succeed even if the
// refresh fails — the same contract bin/wisp-deck's sibling sweeps
// (ensure-budget, ensure-watchdog, ensure-allin) keep with `|| true`.
func (m *MainMenuModel) ensureAllIn() {
	_ = allin.EnsureProfileIfEligible(allin.Env{
		AccountsList:     m.claudeAccountsList,
		AccountsDir:      m.claudeAccountsDir,
		ConfigsList:      m.claudeConfigsList,
		ConfigsDir:       m.claudeConfigsDir,
		DefaultLabelFile: m.claudeDefaultLabelFile,
	})
}
