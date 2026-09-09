package allin

import (
	"encoding/json"
	"testing"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// A GLM row is the failing case this exists for: the session's tool set travels
// unchanged through the router, and z.ai 400s the whole turn on one tool schema
// it cannot validate. The profile's own deny rule cannot cover it — All-In runs
// on the router profile, whose picker also carries Claude rows that want the
// tool — so the drop has to happen per request, here.
func TestResolve_marks_a_zhipu_target_for_the_tool_drop(t *testing.T) {
	env := rosterEnv(t)
	got, err := NewResolver(env).Resolve(Target{Kind: KindConfig, Source: "zhipu-glm", Model: "glm-5.3"})
	if err != nil {
		t.Fatal(err)
	}
	zhipu, ok := claudeconfig.ProviderByKey("zhipu")
	if !ok {
		t.Fatal("catalog is missing the zhipu provider")
	}
	want := zhipu.UnsupportedTools
	if len(got.DropTools) != len(want) || (len(want) > 0 && got.DropTools[0] != want[0]) {
		t.Errorf("DropTools = %v, want %v", got.DropTools, want)
	}
}

func TestDropTools_removes_only_the_named_tools(t *testing.T) {
	payload := map[string]any{
		"model": "glm-5.3",
		"tools": []any{
			map[string]any{"name": "Bash"},
			map[string]any{"name": "Artifact"},
			map[string]any{"name": "Read"},
		},
	}
	dropTools(payload, []string{"Artifact"})
	tools, _ := payload["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("tools = %v, want two left", tools)
	}
	for _, tool := range tools {
		if entry, _ := tool.(map[string]any); entry["name"] == "Artifact" {
			t.Errorf("Artifact survived the drop: %v", tools)
		}
	}
}

// An empty list is the Claude-row case, which runs on every turn: it must not
// touch the body at all.
func TestDropTools_leaves_a_body_alone_when_nothing_is_named(t *testing.T) {
	payload := map[string]any{"tools": []any{map[string]any{"name": "Artifact"}}}
	before, _ := json.Marshal(payload)
	dropTools(payload, nil)
	after, _ := json.Marshal(payload)
	if string(before) != string(after) {
		t.Errorf("body changed: %s -> %s", before, after)
	}
}

// Claude Code sends no `tools` key at all on some turns (the title generator),
// and a drop that invented an empty array there would change what the endpoint
// is asked for.
func TestDropTools_does_not_invent_a_tools_key(t *testing.T) {
	payload := map[string]any{"model": "glm-5.3"}
	dropTools(payload, []string{"Artifact"})
	if _, present := payload["tools"]; present {
		t.Errorf("drop added a tools key: %v", payload)
	}
}

func TestRewriteModel_drops_the_tools_the_target_cannot_take(t *testing.T) {
	payload := map[string]any{
		"model": "wisp/cfg.zhipu-glm/glm-5.3",
		"tools": []any{map[string]any{"name": "Artifact"}, map[string]any{"name": "Bash"}},
	}
	out, err := rewriteModel(payload, "glm-5.3", []string{"Artifact"})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Model string `json:"model"`
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got.Model != "glm-5.3" {
		t.Errorf("model = %q", got.Model)
	}
	if len(got.Tools) != 1 || got.Tools[0].Name != "Bash" {
		t.Errorf("tools = %v, want only Bash", got.Tools)
	}
}
