package allin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHiddenFile_sits_beside_the_configs_list(t *testing.T) {
	got := HiddenFile("/cfg/wisp-deck/claude-configs.list")
	want := "/cfg/wisp-deck/claude-allin.hidden"
	if got != want {
		t.Fatalf("HiddenFile = %q, want %q", got, want)
	}
	if HiddenFile("") != "" {
		t.Fatalf("an unknown list must yield no path, got %q", HiddenFile(""))
	}
}

func TestLoadHidden_reads_nothing_when_the_file_is_absent(t *testing.T) {
	hidden := LoadHidden(filepath.Join(t.TempDir(), "claude-allin.hidden"))
	if len(hidden) != 0 {
		t.Fatalf("a missing file must hide nothing, got %v", hidden)
	}
}

// A model id carries slashes and dots of its own, so the file is read as whole
// lines with only surrounding space trimmed.
func TestLoadHidden_keeps_a_row_id_intact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claude-allin.hidden")
	body := "\n  wisp/cfg.featherless/TurboVadim/Qwen3.8-27B-OBLITERATED  \n\nwisp/acct.default/claude-opus-5\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	hidden := LoadHidden(path)
	if len(hidden) != 2 {
		t.Fatalf("blank lines must not become entries: %v", hidden)
	}
	if !hidden["wisp/cfg.featherless/TurboVadim/Qwen3.8-27B-OBLITERATED"] {
		t.Fatalf("a slashed model id did not survive the read: %v", hidden)
	}
}

func TestToggleHidden_hides_then_shows_a_row(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claude-allin.hidden")
	row := "wisp/acct.default/claude-haiku-4-5-20251001"

	nowHidden, err := ToggleHidden(path, row)
	if err != nil {
		t.Fatal(err)
	}
	if !nowHidden {
		t.Fatal("first toggle must hide the row")
	}
	if !LoadHidden(path)[row] {
		t.Fatal("the hidden row did not survive a reload")
	}

	nowHidden, err = ToggleHidden(path, row)
	if err != nil {
		t.Fatal(err)
	}
	if nowHidden {
		t.Fatal("second toggle must show the row again")
	}
	if LoadHidden(path)[row] {
		t.Fatal("the row is still hidden after being toggled back")
	}
}

// The sidecar lives in the config root, which a fresh machine may not have
// created yet.
func TestToggleHidden_creates_the_directory_it_writes_into(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "claude-allin.hidden")
	if _, err := ToggleHidden(path, "wisp/acct.default/claude-opus-5"); err != nil {
		t.Fatal(err)
	}
	if !LoadHidden(path)["wisp/acct.default/claude-opus-5"] {
		t.Fatal("the row was not written")
	}
}

func TestEnsureProfile_omits_a_hidden_row(t *testing.T) {
	env := rosterEnv(t)
	rows := Roster(env)
	if len(rows) < 2 {
		t.Fatalf("setup: need at least two rows, got %d", len(rows))
	}
	gone, kept := rows[0].Model, rows[1].Model
	if _, err := ToggleHidden(HiddenFile(env.ConfigsList), gone); err != nil {
		t.Fatal(err)
	}

	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	file, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	options := pickerModels(t, filepath.Join(env.ConfigsDir, file))
	if has(options, gone) {
		t.Fatalf("hidden row %q was still written: %v", gone, options)
	}
	if !has(options, kept) {
		t.Fatalf("hiding one row dropped %q too: %v", kept, options)
	}
	if len(options) != len(rows)-1 {
		t.Fatalf("wrote %d options for %d rows with one hidden", len(options), len(rows))
	}
}

// replaceBuiltInOptions leaves no built-in row to fall back on, so an empty
// options list is a picker the user cannot pick anything from, inside a session
// that has no other way to change model. The UI refuses to hide the last row;
// a hand-edited file still reaches here.
func TestEnsureProfile_writes_every_row_when_the_hidden_set_would_empty_the_picker(t *testing.T) {
	env := rosterEnv(t)
	rows := Roster(env)
	for _, row := range rows {
		if _, err := ToggleHidden(HiddenFile(env.ConfigsList), row.Model); err != nil {
			t.Fatal(err)
		}
	}

	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	file, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	if options := pickerModels(t, filepath.Join(env.ConfigsDir, file)); len(options) != len(rows) {
		t.Fatalf("hiding every row left %d options, want the full %d", len(options), len(rows))
	}
}

// Roster is the full offer set: the modal's checklist renders from it, so a row
// the user hid must still appear there to be un-hidden. The filter belongs to
// EnsureProfile alone.
func TestRoster_still_reports_a_hidden_row(t *testing.T) {
	env := rosterEnv(t)
	rows := Roster(env)
	if len(rows) == 0 {
		t.Fatal("setup: no rows")
	}
	if _, err := ToggleHidden(HiddenFile(env.ConfigsList), rows[0].Model); err != nil {
		t.Fatal(err)
	}
	if after := Roster(env); len(after) != len(rows) {
		t.Fatalf("hiding a row changed Roster: %d rows, want %d", len(after), len(rows))
	}
}

// SourceCount gates whether the profile is created at all. Hiding is a display
// preference and must not answer "how many credentials does this machine have".
func TestSourceCount_ignores_hidden_rows(t *testing.T) {
	env := rosterEnv(t)
	before := SourceCount(env)
	for _, row := range Roster(env) {
		if _, err := ToggleHidden(HiddenFile(env.ConfigsList), row.Model); err != nil {
			t.Fatal(err)
		}
	}
	if after := SourceCount(env); after != before {
		t.Fatalf("hiding every row moved SourceCount from %d to %d", before, after)
	}
}

func pickerModels(t *testing.T, path string) []string {
	t.Helper()
	options, _ := readPicker(t, path)["options"].([]any)
	out := make([]string, 0, len(options))
	for _, option := range options {
		row, _ := option.(map[string]any)
		model, _ := row["model"].(string)
		out = append(out, model)
	}
	return out
}
