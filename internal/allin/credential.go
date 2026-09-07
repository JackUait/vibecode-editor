package allin

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// anthropicUpstream is where a Claude login's turn goes. A login has no
// endpoint of its own — only a credential.
const anthropicUpstream = "https://api.anthropic.com"

// KeychainService names the Keychain entry Claude Code writes for one config
// dir. The suffix is sha256 of the directory path, first eight hex characters;
// the default login (no CLAUDE_CONFIG_DIR) has no suffix. Only hashing, so it
// carries no build tag — keychain_darwin.go and keychain_other.go each keep
// their own keychainToken, the part that actually needs one.
func KeychainService(configDir string) string {
	const base = "Claude Code-credentials"
	if configDir == "" {
		return base
	}
	sum := sha256.Sum256([]byte(configDir))
	return base + "-" + hex.EncodeToString(sum[:])[:8]
}

// ErrStaleAccount marks a login whose token could not be read. It is
// deterministic, so the router must surface it as 400: Claude Code retries a
// 401 about eleven times before giving up.
var ErrStaleAccount = errors.New("allin: account credential unavailable")

// Credential is the endpoint and auth header one target needs.
type Credential struct {
	BaseURL string
	Header  string
	Value   string
	// NeedsRepair marks a RemoteCatalog target (Featherless): proxy.go must
	// serve it through internal/rolefix's handler instead of the plain reverse
	// proxy, or it 400s on Claude Code's role:"system" messages and silently
	// stops parsing tool calls once a request carries `thinking`.
	NeedsRepair bool
}

// Resolver answers what a parsed picker row should be sent with.
type Resolver interface {
	Resolve(Target) (Credential, error)
}

// FileResolver reads the same files the account switcher owns. Token is a seam
// so tests never touch the real Keychain.
type FileResolver struct {
	Env   Env
	Token func(configDir string) (string, error)
}

func NewResolver(env Env) *FileResolver {
	return &FileResolver{Env: env, Token: keychainToken}
}

func (r *FileResolver) Resolve(target Target) (Credential, error) {
	if err := validSource(target.Source); err != nil {
		return Credential{}, err
	}
	switch target.Kind {
	case KindAccount:
		configDir := ""
		if target.Source != "default" {
			configDir = filepath.Join(r.Env.AccountsDir, target.Source)
		}
		token, err := r.Token(configDir)
		if err != nil || token == "" {
			return Credential{}, fmt.Errorf("%w: %s", ErrStaleAccount, target.Source)
		}
		return Credential{BaseURL: anthropicUpstream, Header: "Authorization", Value: "Bearer " + token}, nil
	case KindConfig:
		file := target.Source + ".json"
		provider, err := routableProfile(r.Env, file)
		if err != nil {
			return Credential{}, err
		}
		key := claudeconfig.ReadAPIKey(r.Env.ConfigsDir, file)
		base := claudeconfig.ReadBaseURL(r.Env.ConfigsDir, file)
		if key == "" || base == "" {
			return Credential{}, fmt.Errorf("allin: profile %q is not ready", target.Source)
		}
		return Credential{
			BaseURL:     base,
			Header:      "Authorization",
			Value:       "Bearer " + key,
			NeedsRepair: provider.RemoteCatalog,
		}, nil
	}
	return Credential{}, errors.New("allin: session target needs no credential")
}

// routableProfile refuses the one provider configRows deliberately leaves out
// of the roster: anything but AuthAPIKey has no key to swap in. A ChatGPT
// profile is served by a bridge process, and the All-In profile itself is in
// this same configs list, so a row can name the router that is asking for it.
// The id comes off the wire — hand-typed, or saved as a picker default by an
// older build — so this exclusion is advice until it is enforced here too.
//
// RemoteCatalog (Featherless) is NOT refused: the caller (Resolve) reads the
// returned provider's RemoteCatalog bit and marks the credential NeedsRepair,
// so proxy.go routes it through internal/rolefix's handler instead of
// refusing it outright. UserConfigured is also not refused, for a different
// reason — a self-hosted endpoint speaks the Anthropic API directly and needs
// no repair at all. SuppliesOwnModel() is true for both, so keying either
// decision off it instead of RemoteCatalog would wrongly repair (or refuse) a
// self-hosted profile too.
func routableProfile(env Env, file string) (claudeconfig.Provider, error) {
	name := ""
	for _, config := range claudeconfig.Load(env.ConfigsList) {
		if config.File == file {
			name = config.Name
			break
		}
	}
	provider := claudeconfig.ProviderForConfig(env.ConfigsDir,
		claudeconfig.Config{Name: name, File: file})
	if provider.Auth != claudeconfig.AuthAPIKey {
		return provider, fmt.Errorf("allin: profile %q is served by %s, which this router cannot address",
			strings.TrimSuffix(file, ".json"), provider.Name)
	}
	return provider, nil
}

// validSource keeps a model id from naming a path. The id comes off the wire,
// so a "../" source would otherwise read any file the user can read.
func validSource(source string) error {
	if source == "" {
		return errors.New("allin: empty source")
	}
	if strings.ContainsAny(source, "/\\") || strings.Contains(source, "..") {
		return fmt.Errorf("allin: source %q is not a single name", source)
	}
	return nil
}
