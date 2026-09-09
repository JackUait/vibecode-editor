package claudeconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// EnsureUnsupportedTools denies, on a config already on disk, every tool its
// provider's endpoint cannot be sent. Claude Code advertises the whole tool set
// on every turn, so one schema the endpoint refuses kills every turn on the
// profile; a `permissions.deny` entry is what drops the tool from the request
// (verified against a live pane, which then answered normally).
//
// It reports whether the file changed. The installer copies a default profile
// only when the file is absent, so a profile written before its provider
// declared a denial is reachable only by this sweep.
//
// Rules it does not own are kept: the image denials next door are written the
// same way, and a deny rule the user wrote by hand is theirs.
func EnsureUnsupportedTools(configsDir, file string) (bool, error) {
	path := filepath.Join(configsDir, file)
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return false, err
	}

	// The marker is the truth. Its stand-in must be a real alias match, never
	// providerFor's Providers[0] catch-all: that is zhipu, so an unmarked
	// profile for any other endpoint would have a working tool denied.
	provider, ok := providerByKey(ReadProviderMarker(configsDir, file))
	if !ok {
		provider, ok = providerMatching(strings.TrimSuffix(file, ".json"))
	}
	if !ok || len(provider.UnsupportedTools) == 0 {
		return false, nil
	}

	deny := readDenyList(path)
	missing := make([]any, 0, len(provider.UnsupportedTools))
	for _, tool := range provider.UnsupportedTools {
		if !deniesTool(deny, tool) {
			missing = append(missing, tool)
		}
	}
	if len(missing) == 0 {
		return false, nil
	}

	kept := make([]any, 0, len(deny)+len(missing))
	for _, rule := range deny {
		kept = append(kept, rule)
	}
	kept = append(kept, missing...)

	permissions, _ := settings["permissions"].(map[string]any)
	if permissions == nil {
		permissions = make(map[string]any)
	}
	permissions["deny"] = kept
	settings["permissions"] = permissions

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, err
	}
	if err := writeSecure(path, append(out, '\n')); err != nil {
		return false, err
	}
	return true, nil
}

// EnsureUnsupportedToolsAll sweeps every profile in configsDir and reports how
// many changed. A profile it cannot read or parse is skipped rather than
// failing the sweep: one hand-edited file must not leave every other profile
// unrepaired.
func EnsureUnsupportedToolsAll(configsDir string) (int, error) {
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		did, err := EnsureUnsupportedTools(configsDir, entry.Name())
		if err == nil && did {
			changed++
		}
	}
	return changed, nil
}

// deniesTool reports whether a deny list already covers a bare tool name. A
// rule may also be scoped (`Artifact(read:*)`), which denies only part of the
// tool, so only the bare name counts as covering it.
func deniesTool(deny []string, tool string) bool {
	for _, rule := range deny {
		if strings.EqualFold(strings.TrimSpace(rule), tool) {
			return true
		}
	}
	return false
}
