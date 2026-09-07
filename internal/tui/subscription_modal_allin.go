package tui

import "github.com/jackuait/wisp-deck/internal/allin"

// ensureAllIn refreshes the All-In profile after a mutation that can change
// the source count (a subscription key saved, a profile deleted, a login
// added or removed). It shares the exact create-vs-refresh gate the CLI's own
// ensure-allin and add/delete commands use, so the two surfaces cannot drift
// on what counts as "eligible".
//
// Errors are swallowed: this is a convenience layered on top of the mutation
// the user actually asked for, and that action must succeed even if the
// refresh fails — the same contract bin/wisp-deck's sibling sweeps
// (ensure-budget, ensure-watchdog, ensure-allin) keep with `|| true`.
func (m *MainMenuModel) ensureAllIn() {
	_ = allin.EnsureProfileIfEligible(allin.Env{
		AccountsList: m.claudeAccountsList,
		AccountsDir:  m.claudeAccountsDir,
		ConfigsList:  m.claudeConfigsList,
		ConfigsDir:   m.claudeConfigsDir,
	})
}
