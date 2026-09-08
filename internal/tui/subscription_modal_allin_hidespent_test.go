package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackuait/wisp-deck/internal/allin"
	"github.com/jackuait/wisp-deck/internal/subusage"
)

// writeAllInUsage puts a fresh reading in the cache the picker annotates from.
func writeAllInUsage(t *testing.T, path string, usedPercent float64) {
	t.Helper()
	now := time.Now().Unix()
	if err := subusage.WriteCache(path, subusage.Snapshot{
		RateLimits: subusage.RateLimits{FiveHour: &subusage.Window{UsedPercentage: usedPercent}},
		FetchedAt:  now,
		CheckedAt:  now,
	}); err != nil {
		t.Fatal(err)
	}
}

// allInDefaultRow is the roster row spending the implicit login, which is the
// one login this fixture can write a usage cache for by name.
func allInDefaultRow(t *testing.T, rows []allin.Row) allin.Row {
	t.Helper()
	for _, row := range rows {
		if allin.SourceKey(row.Model) == "acct.default" {
			return row
		}
	}
	t.Fatalf("no roster row spends the default login: %+v", rows)
	return allin.Row{}
}

// The pane is where a user decides what /model offers, so it has to show the
// same numbers the picker was annotated with — otherwise a row vanishing from
// /model has no explanation anywhere in the UI.
func TestSubscriptionModal_allInRowsShowWhatEachSubscriptionHasLeft(t *testing.T) {
	m, rows := allInSubscriptionMenu(t)
	row := allInDefaultRow(t, rows)
	writeAllInUsage(t, allin.AccountUsageFile(m.claudeConfigsList, "default"), 77)

	m.loadAllInChecklist()

	if detail := allInDetail(t, m); !strings.Contains(detail, row.Label+" · 23% left") {
		t.Fatalf("row %q does not carry what its login has left:\n%s", row.Label, detail)
	}
}

func TestSubscriptionDetailRows_allInLeadsWithTheHideSpentToggle(t *testing.T) {
	m, rows := allInSubscriptionMenu(t)

	want := []int{subscriptionDetailAllInHideSpent}
	for i := range rows {
		want = append(want, subscriptionDetailAllInBase+i)
	}
	want = append(want, subscriptionDetailRename)

	got := m.subscriptionDetailRows()
	if len(got) != len(want) {
		t.Fatalf("detail rows = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("detail rows = %v, want %v", got, want)
		}
	}
	if m.subscriptionModal.detailCursor != subscriptionDetailAllInHideSpent {
		t.Fatalf("cursor opened on %d, want the toggle row %d",
			m.subscriptionModal.detailCursor, subscriptionDetailAllInHideSpent)
	}
}

// Absent means on, so the pane opens showing the box checked and Enter is what
// turns the filter off.
func TestSubscriptionModal_togglingHideSpentWritesTheSetting(t *testing.T) {
	m, _ := allInSubscriptionMenu(t)
	path := allin.HideExhaustedFile(m.claudeConfigsList)
	if !allin.LoadHideExhausted(path) {
		t.Fatal("setup: a machine that never touched the setting should have it on")
	}
	if detail := allInDetail(t, m); !strings.Contains(detail, "[x] "+subscriptionAllInHideSpentLabel) {
		t.Fatalf("the toggle did not render as on:\n%s", detail)
	}

	m.subscriptionModal.detailCursor = subscriptionDetailAllInHideSpent
	m.activateSubscriptionDetail()

	if m.subscriptionModal.err != nil {
		t.Fatalf("toggling the setting failed: %v", m.subscriptionModal.err)
	}
	if allin.LoadHideExhausted(path) {
		t.Fatal("Enter did not turn the filter off")
	}
	if detail := allInDetail(t, m); !strings.Contains(detail, "[ ] "+subscriptionAllInHideSpentLabel) {
		t.Fatalf("the toggle did not render as off:\n%s", detail)
	}

	m.activateSubscriptionDetail()
	if !allin.LoadHideExhausted(path) {
		t.Fatal("Enter did not turn the filter back on")
	}
}

// The setting is a property of the machine, so flipping it has to rewrite the
// picker on the spot — the same contract a hidden row already has.
func TestSubscriptionModal_togglingHideSpentRewritesThePicker(t *testing.T) {
	m, rows := allInSubscriptionMenu(t)
	spent := allInDefaultRow(t, rows)
	writeAllInUsage(t, allin.AccountUsageFile(m.claudeConfigsList, "default"), 100)

	m.subscriptionModal.detailCursor = subscriptionDetailAllInHideSpent
	m.activateSubscriptionDetail() // off: a spent row comes back
	if m.subscriptionModal.err != nil {
		t.Fatalf("turning the filter off failed: %v", m.subscriptionModal.err)
	}
	found := false
	for _, model := range allInPickerModels(t, m) {
		if model == spent.Model {
			found = true
		}
	}
	if !found {
		t.Fatalf("row %q is missing from the picker with the filter off", spent.Model)
	}

	m.activateSubscriptionDetail() // on again: it goes
	for _, model := range allInPickerModels(t, m) {
		if model == spent.Model {
			t.Fatalf("spent row %q survived the filter", spent.Model)
		}
	}
}

func TestSubscriptionModal_spaceTogglesHideSpent(t *testing.T) {
	m, _ := allInSubscriptionMenu(t)
	m.subscriptionModal.detailCursor = subscriptionDetailAllInHideSpent
	m.subscriptionModal.pane = subscriptionDetailsPane

	m = subscriptionModalKey(t, m, tea.KeyMsg{Type: tea.KeySpace})

	if allin.LoadHideExhausted(allin.HideExhaustedFile(m.claudeConfigsList)) {
		t.Fatal("space did not turn the filter off")
	}
}

func TestSubscriptionModal_clickingHideSpentTogglesIt(t *testing.T) {
	m, _ := allInSubscriptionMenu(t)

	x, y := subscriptionCardCell(t, m, subscriptionAllInHideSpentLabel)
	target := m.subscriptionModalTarget(x, y)
	if target.kind != subscriptionHitField || target.index != subscriptionDetailAllInHideSpent {
		t.Fatalf("hit test on the toggle returned %+v", target)
	}

	updated, _ := m.Update(subscriptionScreenMouse(m, x, y, tea.MouseActionPress, tea.MouseButtonLeft))
	next, ok := updated.(*MainMenuModel)
	if !ok {
		t.Fatalf("Update returned %T", updated)
	}
	m = next

	if allin.LoadHideExhausted(allin.HideExhaustedFile(m.claudeConfigsList)) {
		t.Fatal("clicking the toggle did not turn the filter off")
	}
}
