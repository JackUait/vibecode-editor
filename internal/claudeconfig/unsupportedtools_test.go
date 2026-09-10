package claudeconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestZhipuProvider_declaresTheToolItsEndpointRejects(t *testing.T) {
	provider, ok := ProviderByKey("zhipu")
	if !ok {
		t.Fatal("catalog is missing the zhipu provider")
	}
	if got := provider.UnsupportedTools; len(got) != 1 || got[0] != "Artifact" {
		t.Errorf("zhipu UnsupportedTools = %v, want [Artifact]", got)
	}
	// Every other gateway measured takes the schema without complaint, and
	// denying a tool nobody's endpoint refuses only removes a working
	// capability. A key here must be backed by a live 400, not by a guess:
	// deepseek's is in deepseek_catalog_test.go.
	measured := map[string]bool{"zhipu": true, "deepseek": true}
	for _, other := range Providers {
		if measured[other.Key] {
			continue
		}
		if len(other.UnsupportedTools) != 0 {
			t.Errorf("provider %q denies %v; no live 400 was measured for it",
				other.Key, other.UnsupportedTools)
		}
	}
}

func writeProfile(t *testing.T, dir, name string, settings map[string]any) string {
	t.Helper()
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, name)
}

func TestEnsureUnsupportedTools_denies_the_tool_on_a_zhipu_profile(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "zhipu-glm.json", map[string]any{
		"env": map[string]any{"WISP_DECK_SUBSCRIPTION_PROVIDER": "zhipu"},
	})

	changed, err := EnsureUnsupportedTools(dir, "zhipu-glm.json")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("sweep reported no change on a profile that denied nothing")
	}
	deny := readDenyList(filepath.Join(dir, "zhipu-glm.json"))
	if !contains(deny, "Artifact") {
		t.Errorf("deny = %v, want it to carry Artifact", deny)
	}

	// Every launch path may run the sweep, so a second pass must write nothing.
	changed, err = EnsureUnsupportedTools(dir, "zhipu-glm.json")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("second sweep rewrote a profile that was already correct")
	}
}

func TestEnsureUnsupportedTools_keeps_rules_it_does_not_own(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "zhipu-glm.json", map[string]any{
		"env":         map[string]any{"WISP_DECK_SUBSCRIPTION_PROVIDER": "zhipu"},
		"permissions": map[string]any{"deny": []any{"Read(//**/*.png)", "Bash(rm:*)"}},
	})
	if _, err := EnsureUnsupportedTools(dir, "zhipu-glm.json"); err != nil {
		t.Fatal(err)
	}
	deny := readDenyList(filepath.Join(dir, "zhipu-glm.json"))
	for _, want := range []string{"Read(//**/*.png)", "Bash(rm:*)", "Artifact"} {
		if !contains(deny, want) {
			t.Errorf("deny = %v, want it to carry %q", deny, want)
		}
	}
}

func TestEnsureUnsupportedTools_leaves_a_provider_that_denies_nothing(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "kimi.json", map[string]any{
		"env": map[string]any{"WISP_DECK_SUBSCRIPTION_PROVIDER": "moonshot"},
	})
	changed, err := EnsureUnsupportedTools(dir, "kimi.json")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("sweep touched a profile whose provider denies no tool")
	}
	if deny := readDenyList(filepath.Join(dir, "kimi.json")); len(deny) != 0 {
		t.Errorf("deny = %v, want none", deny)
	}
}

// An unmarked profile resolves by name, exactly like every other reader here —
// and an unmatched name falls through to Providers[0], which IS zhipu.
func TestEnsureUnsupportedTools_resolves_an_unmarked_profile_by_name(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "zhipu-glm.json", map[string]any{"env": map[string]any{}})
	if _, err := EnsureUnsupportedTools(dir, "zhipu-glm.json"); err != nil {
		t.Fatal(err)
	}
	if deny := readDenyList(filepath.Join(dir, "zhipu-glm.json")); !contains(deny, "Artifact") {
		t.Errorf("deny = %v, want it to carry Artifact", deny)
	}
}

func TestEnsureUnsupportedToolsAll_skips_what_it_cannot_parse(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "zhipu-glm.json", map[string]any{
		"env": map[string]any{"WISP_DECK_SUBSCRIPTION_PROVIDER": "zhipu"},
	})
	os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{not json"), 0600)

	changed, err := EnsureUnsupportedToolsAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Errorf("changed = %d, want 1", changed)
	}
}

// A shipped default is copied verbatim on a fresh install, so it has to carry
// the denial its provider declares or every new GLM profile 400s on the first
// interactive turn.
func TestShippedDefaults_denyTheToolsTheirProviderCannotSend(t *testing.T) {
	dir := filepath.Join("..", "..", "defaults", "claude-configs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read defaults: %v", err)
	}
	seen := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		provider, ok := ProviderByKey(ReadProviderMarker(dir, entry.Name()))
		if !ok {
			provider = ProviderForName(entry.Name())
		}
		if len(provider.UnsupportedTools) == 0 {
			continue
		}
		seen++
		deny := readDenyList(filepath.Join(dir, entry.Name()))
		for _, tool := range provider.UnsupportedTools {
			if !contains(deny, tool) {
				t.Errorf("%s: deny = %v, want it to carry %q", entry.Name(), deny, tool)
			}
		}
	}
	if seen == 0 {
		t.Error("no shipped default named a provider that denies a tool — the guard checked nothing")
	}
}

func contains(list []string, want string) bool {
	for _, got := range list {
		if strings.EqualFold(got, want) {
			return true
		}
	}
	return false
}

// Both features write `permissions.deny`, and the image toggle rebuilds that
// list from scratch — so toggling images must not carry the tool denial away
// with it, in either direction.
func TestImagesToggle_keeps_the_tool_denial(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "zhipu-glm.json", map[string]any{
		"env": map[string]any{"WISP_DECK_SUBSCRIPTION_PROVIDER": "zhipu"},
	})
	if _, err := EnsureUnsupportedTools(dir, "zhipu-glm.json"); err != nil {
		t.Fatal(err)
	}
	for _, blocked := range []bool{true, false} {
		if err := WriteImagesBlocked(dir, "zhipu-glm.json", blocked); err != nil {
			t.Fatal(err)
		}
		deny := readDenyList(filepath.Join(dir, "zhipu-glm.json"))
		if !contains(deny, "Artifact") {
			t.Fatalf("images blocked=%v dropped the tool denial: %v", blocked, deny)
		}
	}
}

// providerFor answers Providers[0] — which IS zhipu — for a name matching no
// alias, so a bare name fallback would deny Artifact on an unmarked profile
// that has nothing to do with GLM, removing a working tool from it. Only a real
// alias match may stand in for a missing marker. Same trap the byte watchdog
// records for its own provider read.
func TestEnsureUnsupportedTools_does_not_deny_on_the_zhipu_name_fallback(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "myllm.json", map[string]any{"env": map[string]any{}})
	changed, err := EnsureUnsupportedTools(dir, "myllm.json")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("sweep stamped a profile that matched no provider alias")
	}
	if deny := readDenyList(filepath.Join(dir, "myllm.json")); len(deny) != 0 {
		t.Errorf("deny = %v, want none", deny)
	}
}
