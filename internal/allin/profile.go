package allin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// ProfileName is the display name of the generated profile. It is matched
// verbatim in the configs list, so renaming it orphans the existing one.
const ProfileName = "All-In"

// EnsureProfile creates the All-In profile when absent and rewrites its picker
// rows every call — logins and providers come and go, and a stale roster offers
// models the machine can no longer reach.
//
// Only modelPicker is written. Every other key in the file is the user's, and
// the launch overlay copies the whole object.
func EnsureProfile(env Env, listFile, configsDir string) (string, error) {
	file := existingProfile(listFile)
	if file == "" {
		created, err := claudeconfig.Add(listFile, configsDir, ProfileName)
		if err != nil {
			return "", err
		}
		file = created
	}

	path := filepath.Join(configsDir, file)
	settings := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	rows := Roster(env)
	options := make([]Row, 0, len(rows))
	options = append(options, rows...)
	settings["modelPicker"] = map[string]any{
		// Every row names its account explicitly, so the built-in lineup would
		// only add rows whose credential is ambiguous.
		"replaceBuiltInOptions": true,
		"options":               options,
	}

	encoded, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	// Published by rename: a reader must never see half a settings file.
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(encoded, '\n'), 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(temporary, path); err != nil {
		return "", err
	}
	return file, nil
}

func existingProfile(listFile string) string {
	for _, config := range claudeconfig.Load(listFile) {
		if strings.EqualFold(config.Name, ProfileName) {
			return config.File
		}
	}
	return ""
}
