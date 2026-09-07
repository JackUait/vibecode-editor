package allin

import (
	"errors"
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
	resolver.Token = func(configDir string) (string, error) { return "oat-personal", nil }
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
