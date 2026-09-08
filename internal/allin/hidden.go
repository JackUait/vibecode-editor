package allin

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Hidden picker rows: a row listed in the hidden file is left out of the
// generated profile's picker, while staying in Roster so the Subscription
// modal can still show it and offer to bring it back. The file sits next to
// the configs list (one picker model id per line), so every surface that knows
// the list can derive it — the same shape as claude-configs.disabled.
//
// Hiding is a display preference, never a revocation. Resolve keeps routing a
// hidden id, because a session whose saved picker default was hidden after the
// fact must still be able to finish its turn.

// HiddenFile returns the hidden-rows sidecar path for a configs list file, or
// "" when no list path is known.
func HiddenFile(listFile string) string {
	if listFile == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(listFile), "claude-allin.hidden")
}

// LoadHidden reads the hidden-rows file (one picker model id per line). A
// missing or unreadable file means nothing is hidden.
func LoadHidden(path string) map[string]bool {
	hidden := map[string]bool{}
	data, err := os.ReadFile(path)
	if err != nil {
		return hidden
	}
	for _, line := range strings.Split(string(data), "\n") {
		if model := strings.TrimSpace(line); model != "" {
			hidden[model] = true
		}
	}
	return hidden
}

// ToggleHidden flips a picker row's membership in the hidden-rows file and
// rewrites it. Returns the row's new hidden state.
func ToggleHidden(path, model string) (bool, error) {
	// Stored unmarked, so a row keeps one spelling on disk however the roster
	// marks it. Two spellings of one row would hide it on one pass and show it
	// on the next.
	model = BareModel(model)
	hidden := LoadHidden(path)
	nowHidden := !hidden[model]
	if nowHidden {
		hidden[model] = true
	} else {
		delete(hidden, model)
	}

	lines := make([]string, 0, len(hidden))
	for id := range hidden {
		lines = append(lines, id)
	}
	sort.Strings(lines)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, err
	}
	content := ""
	if len(lines) > 0 {
		content = strings.Join(lines, "\n") + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return false, err
	}
	return nowHidden, nil
}
