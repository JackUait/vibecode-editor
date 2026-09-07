//go:build !darwin

package allin

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

func KeychainService(configDir string) string {
	const base = "Claude Code-credentials"
	if configDir == "" {
		return base
	}
	sum := sha256.Sum256([]byte(configDir))
	return base + "-" + hex.EncodeToString(sum[:])[:8]
}

func keychainToken(string) (string, error) {
	return "", errors.New("allin: Keychain is only readable on darwin")
}
