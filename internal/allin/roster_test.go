package allin

import (
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

func TestRoster_marks_a_1M_capable_model_so_the_client_grants_the_window(t *testing.T) {
	got := models(Roster(rosterEnv(t)))
	if !has(got, "wisp/acct.default/claude-opus-5[1m]") {
		t.Fatalf("no 1m opus row in %v", got)
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

func TestRoster_marks_a_self_hosted_model_with_1M_window(t *testing.T) {
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
	if !has(got, "wisp/cfg.wide-host/qwen-1m[1m]") {
		t.Fatalf("no 1m wide-host row in %v", got)
	}
}
