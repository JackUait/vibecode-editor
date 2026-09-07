package allin

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeychainService_names_the_default_login_without_a_suffix(t *testing.T) {
	if got := KeychainService(""); got != "Claude Code-credentials" {
		t.Fatalf("got %q", got)
	}
}

func TestKeychainService_derives_the_suffix_from_the_config_dir(t *testing.T) {
	// sha256("/Users/jackuait/.config/wisp-deck/claude-accounts/personal")[:8]
	got := KeychainService("/Users/jackuait/.config/wisp-deck/claude-accounts/personal")
	if got != "Claude Code-credentials-7646b36d" {
		t.Fatalf("got %q", got)
	}
}

func TestResolve_hands_an_account_row_its_own_oauth_token(t *testing.T) {
	env := rosterEnv(t)
	resolver := NewResolver(env)
	var gotConfigDir string
	resolver.Token = func(configDir string) (string, error) {
		gotConfigDir = configDir
		return "oat-personal", nil
	}
	got, err := resolver.Resolve(Target{Kind: KindAccount, Source: "personal", Model: "claude-opus-5"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Header != "Authorization" || got.Value != "Bearer oat-personal" {
		t.Fatalf("got %+v", got)
	}
	if got.BaseURL != anthropicUpstream {
		t.Fatalf("base %q", got.BaseURL)
	}
	if want := filepath.Join(env.AccountsDir, "personal"); gotConfigDir != want {
		t.Fatalf("configDir = %q, want %q", gotConfigDir, want)
	}
}

func TestResolve_reads_the_default_login_with_no_config_dir(t *testing.T) {
	env := rosterEnv(t)
	resolver := NewResolver(env)
	var gotConfigDir string
	resolver.Token = func(configDir string) (string, error) {
		gotConfigDir = configDir
		return "oat-default", nil
	}
	if _, err := resolver.Resolve(Target{Kind: KindAccount, Source: "default", Model: "claude-opus-5"}); err != nil {
		t.Fatal(err)
	}
	// The default login has no CLAUDE_CONFIG_DIR, so its Keychain entry is
	// unsuffixed — Token must see an empty configDir, not a joined path.
	if gotConfigDir != "" {
		t.Fatalf("configDir = %q, want empty", gotConfigDir)
	}
}

func TestResolve_hands_a_config_row_its_profile_key_and_endpoint(t *testing.T) {
	env := rosterEnv(t)
	got, err := NewResolver(env).Resolve(Target{Kind: KindConfig, Source: "zhipu-glm", Model: "glm-4.7"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != "Bearer k" || got.BaseURL != "https://api.z.ai/api/anthropic" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolve_reports_a_stale_account_by_name(t *testing.T) {
	env := rosterEnv(t)
	resolver := NewResolver(env)
	resolver.Token = func(string) (string, error) { return "", errors.New("not found") }
	_, err := resolver.Resolve(Target{Kind: KindAccount, Source: "personal"})
	if !errors.Is(err, ErrStaleAccount) {
		t.Fatalf("err = %v", err)
	}
}

func TestResolve_refuses_a_source_that_escapes_its_directory(t *testing.T) {
	env := rosterEnv(t)
	if _, err := NewResolver(env).Resolve(Target{Kind: KindConfig, Source: "../../etc/passwd"}); err == nil {
		t.Fatal("traversal accepted")
	}
}

// writeProfile registers one profile in the roster env and writes its settings.
func writeProfile(t *testing.T, env Env, name, file, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(env.ConfigsDir, file), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	existing, err := os.ReadFile(env.ConfigsList)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.ConfigsList,
		append(existing, []byte(name+":"+file+"\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
}

// configRows will not offer these, for reasons that are about the ROUTER, not
// about the profile: it does no role/thinking repair, so a Featherless row 400s
// or silently stops calling tools, and a ChatGPT profile is served by a bridge
// process that has no key to hand over. Resolve enforced neither, so a
// hand-typed id — or a picker default saved from an older roster — walked
// straight past the exclusion the roster exists to apply.
func TestResolve_refuses_a_provider_the_roster_would_not_offer(t *testing.T) {
	for _, tc := range []struct{ name, file, body string }{
		{"Featherless", "featherless.json", `{"env":{
"WISP_DECK_SUBSCRIPTION_PROVIDER":"featherless",
"ANTHROPIC_BASE_URL":"https://api.featherless.ai",
"ANTHROPIC_AUTH_TOKEN":"sk-test"}}`},
		{"OpenAI / ChatGPT", "openai-chatgpt.json", `{"env":{
"WISP_DECK_SUBSCRIPTION_PROVIDER":"openai-chatgpt",
"ANTHROPIC_BASE_URL":"https://api.openai.com",
"ANTHROPIC_AUTH_TOKEN":"sk-test"}}`},
	} {
		t.Run(tc.file, func(t *testing.T) {
			env := rosterEnv(t)
			writeProfile(t, env, tc.name, tc.file, tc.body)
			source := strings.TrimSuffix(tc.file, ".json")
			_, err := NewResolver(env).Resolve(Target{Kind: KindConfig, Source: source, Model: "m"})
			if err == nil {
				t.Fatalf("%s resolved, so an unrepaired turn goes to it", tc.name)
			}
			if !strings.Contains(err.Error(), source) {
				t.Fatalf("error does not name the profile: %v", err)
			}
		})
	}
}

// The All-In profile is in the same configs list, and its own rows are the ones
// being typed — so it can name itself. Routing into the router that is asking
// would loop a turn back through this handler.
func TestResolve_refuses_the_router_profile_itself(t *testing.T) {
	env := rosterEnv(t)
	if _, err := EnsureProfile(env, env.ConfigsList, env.ConfigsDir); err != nil {
		t.Fatal(err)
	}
	if _, err := NewResolver(env).Resolve(
		Target{Kind: KindConfig, Source: "all-in", Model: "m"}); err == nil {
		t.Fatal("All-In resolved as a routing destination")
	}
}

// The refusal must not have swallowed the providers the roster does offer.
func TestResolve_still_serves_a_provider_the_roster_offers(t *testing.T) {
	env := rosterEnv(t)
	if _, err := NewResolver(env).Resolve(
		Target{Kind: KindConfig, Source: "zhipu-glm", Model: "glm-4.7"}); err != nil {
		t.Fatalf("an ordinary gateway stopped resolving: %v", err)
	}
}

// A self-hosted profile speaks the Anthropic API directly and needs no repair,
// so the refusal above must key on RemoteCatalog, never on SuppliesOwnModel —
// both are true for Featherless and only the first names the repair gap.
func TestResolve_still_serves_a_self_hosted_profile(t *testing.T) {
	env := rosterEnv(t)
	writeProfile(t, env, "Self Hosted", "self-hosted.json", `{"env":{
"WISP_DECK_SUBSCRIPTION_PROVIDER":"custom",
"ANTHROPIC_BASE_URL":"http://localhost:8000",
"ANTHROPIC_AUTH_TOKEN":"sk-test"}}`)
	if _, err := NewResolver(env).Resolve(
		Target{Kind: KindConfig, Source: "self-hosted", Model: "qwen"}); err != nil {
		t.Fatalf("a self-hosted profile stopped resolving: %v", err)
	}
}
