package allin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rosterEnv(t *testing.T) Env {
	t.Helper()
	dir := t.TempDir()
	accounts := filepath.Join(dir, "claude-accounts")
	configs := filepath.Join(dir, "claude-configs")
	if err := os.MkdirAll(filepath.Join(accounts, "personal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(dir, "claude-accounts.list"), "Personal:personal\n")
	write(filepath.Join(dir, "claude-configs.list"), "Zhipu GLM:zhipu-glm.json\n")
	write(filepath.Join(configs, "zhipu-glm.json"),
		`{"env":{"ANTHROPIC_BASE_URL":"https://api.z.ai/api/anthropic","ANTHROPIC_AUTH_TOKEN":"k"}}`)
	return Env{
		AccountsList: filepath.Join(dir, "claude-accounts.list"),
		AccountsDir:  accounts,
		ConfigsList:  filepath.Join(dir, "claude-configs.list"),
		ConfigsDir:   configs,
	}
}

func models(rows []Row) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Model)
	}
	return out
}

func has(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func TestRoster_lists_the_default_login_and_every_registered_account(t *testing.T) {
	got := models(Roster(rosterEnv(t)))
	if !has(got, "wisp/acct.default/claude-opus-5") {
		t.Fatalf("no default opus row in %v", got)
	}
	if !has(got, "wisp/acct.personal/claude-opus-5") {
		t.Fatalf("no personal opus row in %v", got)
	}
}

// The suffix is the only thing that grants a row a 1M window, and nothing
// narrows the window again when the user picks a 200k row later in the same
// conversation: the transcript is already past the new endpoint's cap, and
// /compact is larger than the turn that just failed. Every row is uniformly
// 200k so no pick can wedge the session. Route keeps stripping and reporting
// the marker — 1M returns behind a size guard, not by re-adding this.
func TestRoster_never_offers_a_1m_row(t *testing.T) {
	for _, id := range models(Roster(rosterEnv(t))) {
		if strings.HasSuffix(id, "[1m]") {
			t.Fatalf("a row promises a 1M window nothing can narrow again: %s", id)
		}
	}
}

// Measured on a live pane: an unknown model id WITHOUT behavesAs is given
// thinking:{"type":"adaptive"} and effort stays available; with
// behavesAs:claude-sonnet-4-5 it becomes thinking:{"type":"enabled",budget} and
// the pane reports "Effort not supported". Absent is the more capable default.
func TestRoster_never_declares_behaves_as(t *testing.T) {
	encoded, err := json.Marshal(Roster(rosterEnv(t)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "behavesAs") {
		t.Fatalf("a row declares behavesAs, which costs it adaptive thinking: %s", encoded)
	}
}

func TestRoster_lists_a_configured_providers_models(t *testing.T) {
	got := models(Roster(rosterEnv(t)))
	if !has(got, "wisp/cfg.zhipu-glm/glm-4.7") {
		t.Fatalf("no zhipu row in %v", got)
	}
}

func TestRoster_omits_a_model_too_narrow_for_claude_code(t *testing.T) {
	for _, id := range models(Roster(rosterEnv(t))) {
		if strings.HasSuffix(id, "/glm-4.5-air") {
			t.Fatalf("131072-token model was offered: %s", id)
		}
	}
}

func TestRoster_omits_a_provider_that_is_not_served_by_an_api_key(t *testing.T) {
	env := rosterEnv(t)
	if err := os.WriteFile(filepath.Join(env.ConfigsDir, "openai-chatgpt.json"),
		[]byte(`{"env":{"WISP_DECK_SUBSCRIPTION_PROVIDER":"openai-chatgpt"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.ConfigsList,
		[]byte("Zhipu GLM:zhipu-glm.json\nOpenAI / ChatGPT:openai-chatgpt.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, id := range models(Roster(env)) {
		if strings.Contains(id, "cfg.openai-chatgpt/") {
			t.Fatalf("ChatGPT row offered with no endpoint to serve it: %s", id)
		}
	}
}

func TestRoster_omits_featherless_because_the_router_has_no_role_repair(t *testing.T) {
	env := rosterEnv(t)
	if err := os.WriteFile(filepath.Join(env.ConfigsDir, "featherless.json"),
		[]byte(`{
"name":"Featherless",
"env":{
"WISP_DECK_SUBSCRIPTION_PROVIDER":"featherless",
"ANTHROPIC_BASE_URL":"https://api.featherless.ai",
"ANTHROPIC_AUTH_TOKEN":"sk-test",
"ANTHROPIC_DEFAULT_OPUS_MODEL":"zai-org/GLM-5.3-Flash",
"ANTHROPIC_DEFAULT_SONNET_MODEL":"zai-org/GLM-5.3-Flash",
"ANTHROPIC_DEFAULT_FABLE_MODEL":"zai-org/GLM-5.3-Flash",
"ANTHROPIC_DEFAULT_HAIKU_MODEL":"zai-org/GLM-5.3-Flash",
"CLAUDE_CODE_MAX_CONTEXT_TOKENS":"262144"
}
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.ConfigsList,
		[]byte("Zhipu GLM:zhipu-glm.json\nFeatherless:featherless.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, id := range models(Roster(env)) {
		if strings.Contains(id, "cfg.featherless/") {
			t.Fatalf("Featherless row offered with no repair proxy in the router: %s", id)
		}
	}
}

func TestRoster_labels_a_row_with_its_source(t *testing.T) {
	for _, row := range Roster(rosterEnv(t)) {
		if row.Model == "wisp/acct.personal/claude-opus-5" && !strings.Contains(row.Label, "Personal") {
			t.Fatalf("label %q does not name the account", row.Label)
		}
	}
}

func TestRoster_omits_a_self_hosted_model_too_narrow_for_claude_code(t *testing.T) {
	env := rosterEnv(t)
	// Custom provider with narrow context window (below 200k minimum).
	if err := os.WriteFile(filepath.Join(env.ConfigsDir, "narrow-host.json"),
		[]byte(`{
"name":"Narrow Self-Hosted",
"env":{
"WISP_DECK_SUBSCRIPTION_PROVIDER":"custom",
"ANTHROPIC_BASE_URL":"http://localhost:8000",
"ANTHROPIC_AUTH_TOKEN":"sk-test",
"ANTHROPIC_DEFAULT_OPUS_MODEL":"qwen-32k",
"ANTHROPIC_DEFAULT_SONNET_MODEL":"qwen-32k",
"ANTHROPIC_DEFAULT_FABLE_MODEL":"qwen-32k",
"ANTHROPIC_DEFAULT_HAIKU_MODEL":"qwen-32k",
"CLAUDE_CODE_MAX_CONTEXT_TOKENS":"32768"
}
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.ConfigsList,
		[]byte("Zhipu GLM:zhipu-glm.json\nNarrow Self-Hosted:narrow-host.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, id := range models(Roster(env)) {
		if strings.Contains(id, "cfg.narrow-host/") {
			t.Fatalf("32768-token custom provider was offered: %s", id)
		}
	}
}

func TestRoster_offers_a_wide_self_hosted_model_at_the_uniform_window(t *testing.T) {
	env := rosterEnv(t)
	// Custom provider with 1M+ context window.
	if err := os.WriteFile(filepath.Join(env.ConfigsDir, "wide-host.json"),
		[]byte(`{
"name":"Wide Self-Hosted",
"env":{
"WISP_DECK_SUBSCRIPTION_PROVIDER":"custom",
"ANTHROPIC_BASE_URL":"http://localhost:8000",
"ANTHROPIC_AUTH_TOKEN":"sk-test",
"ANTHROPIC_DEFAULT_OPUS_MODEL":"qwen-1m",
"ANTHROPIC_DEFAULT_SONNET_MODEL":"qwen-1m",
"ANTHROPIC_DEFAULT_FABLE_MODEL":"qwen-1m",
"ANTHROPIC_DEFAULT_HAIKU_MODEL":"qwen-1m",
"CLAUDE_CODE_MAX_CONTEXT_TOKENS":"1000000"
}
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.ConfigsList,
		[]byte("Zhipu GLM:zhipu-glm.json\nWide Self-Hosted:wide-host.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := models(Roster(env))
	if !has(got, "wisp/cfg.wide-host/qwen-1m") {
		t.Fatalf("no wide-host row in %v", got)
	}
}

// configRows iterates the very list EnsureProfile registers the All-In profile
// in, so an unguarded roster offers rows pointing at the router that is asking
// for them: a turn on one would loop back into this same proxy.
func TestRoster_omits_the_profile_it_generates(t *testing.T) {
	env := rosterEnv(t)
	file, err := EnsureProfile(env, env.ConfigsList, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	own := "cfg." + strings.TrimSuffix(file, ".json") + "/"
	for _, id := range models(Roster(env)) {
		if strings.Contains(id, own) {
			t.Fatalf("All-In offers a row pointing at itself: %s", id)
		}
	}
}

// existingProfile adopts ANY profile named "All-In", whatever it was before, and
// Roster runs against the file as it is on disk — before EnsureProfile stamps
// the router env over it. So an adopted profile is still a routable API-key
// provider at the moment its own rows are computed, and only skipping it by
// name keeps All-In from offering a row that loops back into its own router.
func TestRoster_omits_an_adopted_profile_that_still_looks_routable(t *testing.T) {
	env := rosterEnv(t)
	if err := os.WriteFile(filepath.Join(env.ConfigsDir, "all-in.json"),
		[]byte(`{"env":{"WISP_DECK_SUBSCRIPTION_PROVIDER":"zhipu",`+
			`"ANTHROPIC_BASE_URL":"https://api.z.ai/api/anthropic",`+
			`"ANTHROPIC_AUTH_TOKEN":"k"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.ConfigsList,
		[]byte("Zhipu GLM:zhipu-glm.json\n"+ProfileName+":all-in.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, id := range models(Roster(env)) {
		if strings.Contains(id, "cfg.all-in/") {
			t.Fatalf("All-In offers a row pointing at itself: %s", id)
		}
	}
}
