//go:build darwin

package allin

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

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
