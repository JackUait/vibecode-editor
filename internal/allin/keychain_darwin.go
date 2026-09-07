//go:build darwin

package allin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// KeychainService names the Keychain entry Claude Code writes for one config
// dir. The suffix is sha256 of the directory path, first eight hex characters;
// the default login (no CLAUDE_CONFIG_DIR) has no suffix.
func KeychainService(configDir string) string {
	const base = "Claude Code-credentials"
	if configDir == "" {
		return base
	}
	sum := sha256.Sum256([]byte(configDir))
	return base + "-" + hex.EncodeToString(sum[:])[:8]
}

func keychainToken(configDir string) (string, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", KeychainService(configDir), "-w").Output()
	if err != nil {
		return "", err
	}
	var blob struct {
		OAuth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &blob); err != nil {
		return "", fmt.Errorf("allin: parse keychain blob: %w", err)
	}
	return blob.OAuth.AccessToken, nil
}
