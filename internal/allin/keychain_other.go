//go:build !darwin

package allin

import "errors"

func keychainToken(string) (string, error) {
	return "", errors.New("allin: Keychain is only readable on darwin")
}
