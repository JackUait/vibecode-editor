package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackuait/wisp-deck/internal/allin"
	"github.com/jackuait/wisp-deck/internal/claudeconfig"
	"github.com/jackuait/wisp-deck/internal/models"
)

// newBareSubscriptionMenu builds a MainMenuModel wired to real, but empty,
// config and account files — a genuine single-source machine (Default alone)
// — so each test controls the source count precisely from a known baseline.
// newSubscriptionModalMenu (subscription_modal_test.go) pre-seeds a ready
// provider, which already crosses the two-source line before any mutation.
func newBareSubscriptionMenu(t *testing.T) *MainMenuModel {
	t.Helper()
	dir := t.TempDir()
	list := filepath.Join(dir, "claude-configs.list")
	configsDir := filepath.Join(dir, "claude-configs")
	pointer := filepath.Join(dir, "claude-config")
	accountsList := filepath.Join(dir, "claude-accounts.list")
	accountsDir := filepath.Join(dir, "claude-accounts")

	m := NewMainMenu(
		[]models.Project{{Name: "p", Path: "/p"}},
		[]string{"claude", "opencode"},
		"claude",
		"none",
	)
	m.SetSize(100, 36)
	m.SetClaudeConfigFile(pointer)
	m.SetClaudeConfigPaths(list, configsDir)
	m.SetClaudeConfigs(LoadClaudeConfigsList(list))
	m.SetActiveClaudeConfig("")
	m.SetClaudeAccountFile(filepath.Join(dir, "claude-account"))
	m.SetClaudeAccountPaths(accountsList, accountsDir)
	m.SetClaudeDefaultLabelFile(filepath.Join(dir, "claude-account-default-label"))
	m.SetClaudeAccounts(nil)
	m.SetActiveTab(TabSettings)
	m.settingsSelected = rowSubscription
	return m
}

// This is the gap the feature closes: a machine that adds its second Claude
// login through the modal (never touching bin/wisp-deck's installer sweep)
// must get the All-In profile without a relaunch of setup.
func TestAddSubscriptionLogin_crossing_two_sources_creates_the_allin_profile(t *testing.T) {
	m := newBareSubscriptionMenu(t)
	if file := allin.ProfileFile(m.claudeConfigsList); file != "" {
		t.Fatalf("setup: All-In profile already exists on a fresh machine: %s", file)
	}

	m.addSubscriptionLogin("Work")

	if m.subscriptionModal.err != nil {
		t.Fatalf("addSubscriptionLogin failed: %v", m.subscriptionModal.err)
	}
	file := allin.ProfileFile(m.claudeConfigsList)
	if file == "" {
		t.Fatal("adding a second login did not create the All-In profile")
	}

	// The profile must be visible and selectable in the SAME open modal,
	// without a relaunch — m.claudeConfigs has to be reloaded after the
	// write, not left holding the pre-creation list.
	found := false
	for _, p := range m.subscriptionProfiles() {
		if p.File == file {
			found = true
		}
	}
	if !found {
		t.Fatalf("newly created All-In profile is not visible in the open modal: %+v", m.subscriptionProfiles())
	}

	// The cursor must land on the login just added ("Work", claudeAccounts[0],
	// so subscriptionModalLoginIndex() == 1). subscriptionLoginRowStart() is
	// len(subscriptionProfiles())+1, which grows by one the moment the All-In
	// config row is created — computing the cursor BEFORE that reload leaves
	// it one row short, landing on Default instead.
	if got := m.subscriptionModalLoginIndex(); got != 1 {
		t.Fatalf("cursor landed on login index %d, want 1 (Work)", got)
	}
}

// A machine whose two-plus sources were connected before this feature
// existed — or in a session that never mutates anything — never reaches any
// mutation call site, so it never got the profile any other way. Opening the
// Subscriptions modal is a read, not a mutation, and is the point that closes
// the gap.
func TestOpenSubscriptionModal_creates_the_allin_profile_for_a_user_who_mutates_nothing(t *testing.T) {
	m := newBareSubscriptionMenu(t)
	// Two logins connected entirely outside this session — never through
	// addSubscriptionLogin or any other mutation call site that would already
	// have run the gate.
	if err := os.WriteFile(m.claudeAccountsList, []byte("Work:work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(m.claudeAccountsDir, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	m.SetClaudeAccounts(LoadClaudeAccountsList(m.claudeAccountsList))

	if allin.ProfileFile(m.claudeConfigsList) != "" {
		t.Fatal("setup: All-In profile already exists")
	}

	m.openSubscriptionModal()

	file := allin.ProfileFile(m.claudeConfigsList)
	if file == "" {
		t.Fatal("opening the Subscriptions modal did not create the All-In profile for an existing two-source machine")
	}
	found := false
	for _, p := range m.subscriptionProfiles() {
		if p.File == file {
			found = true
		}
	}
	if !found {
		t.Fatalf("newly created All-In profile is not visible in the just-opened modal: %+v", m.subscriptionProfiles())
	}
}

// Removing a login is a source disappearing; an existing All-In profile must
// still be rebuilt so it stops naming a login that is gone.
func TestDeleteSubscriptionLogin_refreshes_an_existing_allin_profile(t *testing.T) {
	m := newBareSubscriptionMenu(t)
	m.addSubscriptionLogin("Work") // two sources: creates the profile
	file := allin.ProfileFile(m.claudeConfigsList)
	if file == "" {
		t.Fatal("setup: All-In profile was not created")
	}
	m.addSubscriptionLogin("Personal") // three sources, so deleting one still leaves two

	m.subscriptionModal.profileCursor = m.subscriptionLoginRowStart() + 1 // "Work"
	m.deleteSubscriptionLogin()

	if m.subscriptionModal.err != nil {
		t.Fatalf("deleteSubscriptionLogin failed: %v", m.subscriptionModal.err)
	}
	data, err := os.ReadFile(filepath.Join(m.claudeConfigsDir, file))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "acct.work/") {
		t.Fatalf("removed login's rows survived the refresh: %s", data)
	}
}

// Deleting the All-In profile itself must not bring it straight back, even
// though the machine still has two other sources to route between.
func TestDeleteSubscriptionProfile_never_recreates_the_profile_it_just_deleted(t *testing.T) {
	m := newBareSubscriptionMenu(t)
	m.addSubscriptionLogin("Work")
	m.addSubscriptionLogin("Personal") // three sources: Default, Work, Personal
	file := allin.ProfileFile(m.claudeConfigsList)
	if file == "" {
		t.Fatal("setup: All-In profile was not created")
	}
	m.SetClaudeConfigs(LoadClaudeConfigsList(m.claudeConfigsList))

	// Select the All-In profile row and delete it.
	profiles := m.subscriptionProfiles()
	idx := -1
	for i, p := range profiles {
		if p.File == file {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("All-In profile not listed among subscription profiles: %+v", profiles)
	}
	m.subscriptionModal.profileCursor = idx

	m.deleteSubscriptionProfile()

	if got := allin.ProfileFile(m.claudeConfigsList); got != "" {
		t.Fatalf("All-In profile came back after its own delete: %s", got)
	}
}

// saveSubscriptionDraft's WriteAPIKey call is only ONE of the ways it can
// make a profile ready — a self-hosted (SuppliesOwnModel) profile that
// already has a key becomes ready purely from its model/window fields, with
// draft.keyEdited == false. The hook must not be gated on keyEdited.
func TestSaveSubscriptionDraft_creates_allin_when_a_model_only_save_completes_readiness(t *testing.T) {
	m := newBareSubscriptionMenu(t) // Default alone: one source
	file, err := claudeconfig.AddForProvider(m.claudeConfigsList, m.claudeConfigsDir, "My Endpoint", "custom")
	if err != nil {
		t.Fatal(err)
	}
	if err := claudeconfig.WriteAPIKey(m.claudeConfigsDir, file, "sk-test"); err != nil {
		t.Fatal(err)
	}
	m.SetClaudeConfigs(LoadClaudeConfigsList(m.claudeConfigsList))
	if allin.ProfileFile(m.claudeConfigsList) != "" {
		t.Fatal("setup: profile should not exist before the endpoint/model are saved")
	}

	profiles := m.subscriptionProfiles()
	idx := -1
	for i, p := range profiles {
		if p.File == file {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("custom profile not listed: %+v", profiles)
	}
	m.subscriptionModal.profileCursor = idx
	m.loadSubscriptionDraft(m.subscriptionModalProfile())
	m.subscriptionModal.draft.endpoint = "https://example.test/v1"
	m.subscriptionModal.draft.model = "my-model"
	// Must clear minRosterContext (200000, allin/roster.go) or the roster
	// omits the row for being too narrow, independent of readiness.
	m.subscriptionModal.draft.window = "262144"
	m.subscriptionModal.draft.customEdited = true
	m.subscriptionModal.draft.dirty = true

	m.saveSubscriptionDraft()

	if m.subscriptionModal.err != nil {
		t.Fatalf("saveSubscriptionDraft failed: %v", m.subscriptionModal.err)
	}
	allInFile := allin.ProfileFile(m.claudeConfigsList)
	if allInFile == "" {
		t.Fatal("a model-only save that completed readiness did not create the All-In profile")
	}

	// The new profile must be visible in the same open modal without a
	// relaunch — m.claudeConfigs has to be reloaded after ensureAllIn writes.
	found := false
	for _, p := range m.subscriptionProfiles() {
		if p.File == allInFile {
			found = true
		}
	}
	if !found {
		t.Fatalf("newly created All-In profile is not visible in the open modal: %+v", m.subscriptionProfiles())
	}
}

// Renaming a login changes the display label roster.go's accountRows embeds
// in every one of its rows for that login. A stale label survives in an
// already-existing All-In profile until something refreshes it.
func TestRenameSubscriptionLogin_refreshes_an_existing_allin_profile(t *testing.T) {
	m := newBareSubscriptionMenu(t)
	m.addSubscriptionLogin("Work") // two sources: creates the profile
	file := allin.ProfileFile(m.claudeConfigsList)
	if file == "" {
		t.Fatal("setup: All-In profile was not created")
	}

	m.subscriptionModal.profileCursor = m.subscriptionLoginRowStart() + 1 // "Work"
	m.renameSubscriptionLogin("Office")

	if m.subscriptionModal.err != nil {
		t.Fatalf("renameSubscriptionLogin failed: %v", m.subscriptionModal.err)
	}
	data, err := os.ReadFile(filepath.Join(m.claudeConfigsDir, file))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "Work ·") {
		t.Fatalf("stale login label survived the rename: %s", data)
	}
	if !strings.Contains(string(data), "Office ·") {
		t.Fatalf("renamed login label missing from the refreshed profile: %s", data)
	}
}

// Renaming a subscription profile changes the display name roster.go's
// configRows embeds in every one of its rows for that profile. A stale name
// survives in an already-existing All-In profile until something refreshes
// it.
func TestRenameSubscriptionProfile_refreshes_an_existing_allin_profile(t *testing.T) {
	m := newBareSubscriptionMenu(t) // Default alone: one source
	file, err := claudeconfig.AddForProvider(m.claudeConfigsList, m.claudeConfigsDir, "Zhipu GLM", "zhipu")
	if err != nil {
		t.Fatal(err)
	}
	if err := claudeconfig.WriteAPIKey(m.claudeConfigsDir, file, "sk-test"); err != nil {
		t.Fatal(err)
	}
	m.SetClaudeConfigs(LoadClaudeConfigsList(m.claudeConfigsList))
	m.ensureAllIn() // two sources now (Default + Zhipu GLM): creates
	allInFile := allin.ProfileFile(m.claudeConfigsList)
	if allInFile == "" {
		t.Fatal("setup: All-In profile was not created")
	}
	m.SetClaudeConfigs(LoadClaudeConfigsList(m.claudeConfigsList)) // pick up the new all-in row too

	profiles := m.subscriptionProfiles()
	idx := -1
	for i, p := range profiles {
		if p.File == file {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("zhipu profile not listed: %+v", profiles)
	}
	m.subscriptionModal.profileCursor = idx

	m.renameSubscriptionProfile("Zhipu Renamed")

	if m.subscriptionModal.err != nil {
		t.Fatalf("renameSubscriptionProfile failed: %v", m.subscriptionModal.err)
	}
	data, err := os.ReadFile(filepath.Join(m.claudeConfigsDir, allInFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "Zhipu GLM ·") {
		t.Fatalf("stale profile name survived the rename: %s", data)
	}
	if !strings.Contains(string(data), "Zhipu Renamed ·") {
		t.Fatalf("renamed profile name missing from the refreshed profile: %s", data)
	}
}

// Disabling a subscription changes what All-In may route to, exactly like a
// rename or a delete — every other mutation site refreshes immediately, and
// this one must too, rather than waiting on the next unrelated mutation or
// modal reopen to drop the disabled provider's rows.
func TestToggleSubscriptionProfileDisabled_refreshes_an_existing_allin_profile(t *testing.T) {
	m := newBareSubscriptionMenu(t) // Default alone: one source
	file, err := claudeconfig.AddForProvider(m.claudeConfigsList, m.claudeConfigsDir, "Zhipu GLM", "zhipu")
	if err != nil {
		t.Fatal(err)
	}
	if err := claudeconfig.WriteAPIKey(m.claudeConfigsDir, file, "sk-test"); err != nil {
		t.Fatal(err)
	}
	m.SetClaudeConfigs(LoadClaudeConfigsList(m.claudeConfigsList))
	m.ensureAllIn() // two sources now (Default + Zhipu GLM): creates
	allInFile := allin.ProfileFile(m.claudeConfigsList)
	if allInFile == "" {
		t.Fatal("setup: All-In profile was not created")
	}
	m.SetClaudeConfigs(LoadClaudeConfigsList(m.claudeConfigsList)) // pick up the new all-in row too

	m.openSubscriptionModal()
	idx := -1
	for i, p := range m.subscriptionProfiles() {
		if p.File == file {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("zhipu profile not listed: %+v", m.subscriptionProfiles())
	}
	m.selectSubscriptionProfile(idx)
	m = subscriptionModalKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if m.subscriptionModal.err != nil {
		t.Fatalf("disabling the profile failed: %v", m.subscriptionModal.err)
	}
	data, err := os.ReadFile(filepath.Join(m.claudeConfigsDir, allInFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "cfg.zhipu-glm/") {
		t.Fatalf("disabling the subscription did not refresh the All-In profile: %s", data)
	}
}
