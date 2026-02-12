package bash_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// notificationSnippet builds a bash snippet that sources tui.sh, settings-json.sh,
// and notification-setup.sh, then runs the provided bash code.
func notificationSnippet(t *testing.T, body string) string {
	t.Helper()
	root := projectRoot(t)
	tuiPath := filepath.Join(root, "lib", "tui.sh")
	settingsJsonPath := filepath.Join(root, "lib", "settings-json.sh")
	notifPath := filepath.Join(root, "lib", "notification-setup.sh")
	return fmt.Sprintf("source %q && source %q && source %q && %s",
		tuiPath, settingsJsonPath, notifPath, body)
}

// updateSnippet builds a bash snippet that sources update.sh then runs the provided bash code.
func updateSnippet(t *testing.T, body string) string {
	t.Helper()
	root := projectRoot(t)
	updatePath := filepath.Join(root, "lib", "update.sh")
	return fmt.Sprintf("source %q && %s", updatePath, body)
}

// ==================== notification-setup.sh tests ====================

// --- setup_sound_notification ---

func TestNotification_setup_sound_notification_adds_Stop_hook_to_empty_settings(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{}`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`setup_sound_notification %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "configured")

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	content := string(data)
	assertContains(t, content, `"Stop"`)
	assertNotContains(t, content, "idle_prompt")
}

func TestNotification_setup_sound_notification_reports_already_exists(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{
  "hooks": {
    "Stop": [
      {
        "hooks": [{"type": "command", "command": "afplay sound &"}]
      }
    ]
  }
}
`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`setup_sound_notification %q "afplay sound &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "already configured")
}

func TestNotification_setup_sound_notification_creates_file_when_missing(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "new-settings.json")

	snippet := notificationSnippet(t,
		fmt.Sprintf(`setup_sound_notification %q "afplay sound &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "configured")

	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		t.Fatalf("expected settings file to be created at %s", settingsFile)
	}
}

// --- is_sound_enabled ---

func TestNotification_is_sound_enabled_returns_true_when_features_file_missing(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentDir := filepath.Join(tmpDir, "nonexistent")

	snippet := notificationSnippet(t,
		fmt.Sprintf(`is_sound_enabled "claude" %q`, nonexistentDir))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "true" {
		t.Errorf("expected 'true', got %q", strings.TrimSpace(out))
	}
}

func TestNotification_is_sound_enabled_returns_true_when_sound_key_missing(t *testing.T) {
	tmpDir := t.TempDir()
	writeTempFile(t, tmpDir, "claude-features.json", `{}`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`is_sound_enabled "claude" %q`, tmpDir))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "true" {
		t.Errorf("expected 'true', got %q", strings.TrimSpace(out))
	}
}

func TestNotification_is_sound_enabled_returns_true_when_sound_is_true(t *testing.T) {
	tmpDir := t.TempDir()
	writeTempFile(t, tmpDir, "claude-features.json", `{"sound": true}`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`is_sound_enabled "claude" %q`, tmpDir))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "true" {
		t.Errorf("expected 'true', got %q", strings.TrimSpace(out))
	}
}

func TestNotification_is_sound_enabled_returns_false_when_sound_is_false(t *testing.T) {
	tmpDir := t.TempDir()
	writeTempFile(t, tmpDir, "claude-features.json", `{"sound": false}`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`is_sound_enabled "claude" %q`, tmpDir))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "false" {
		t.Errorf("expected 'false', got %q", strings.TrimSpace(out))
	}
}

// --- remove_sound_notification ---

func TestNotification_remove_sound_notification_removes_Stop_hook(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{
  "hooks": {
    "Stop": [
      {
        "hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Bottle.aiff &"}]
      }
    ]
  }
}
`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`remove_sound_notification %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "removed")

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	assertNotContains(t, string(data), "afplay")
}

func TestNotification_remove_sound_notification_noop_when_hook_not_present(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{}`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`remove_sound_notification %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "not_found")
}

func TestNotification_remove_sound_notification_removes_matching_keeps_others(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{
  "hooks": {
    "Stop": [
      {
        "hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Bottle.aiff &"}]
      },
      {
        "hooks": [{"type": "command", "command": "other-command"}]
      }
    ]
  }
}
`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`remove_sound_notification %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "removed")

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	content := string(data)
	assertNotContains(t, content, "afplay")
	assertContains(t, content, "other-command")
}

// --- set_sound_feature_flag ---

func TestNotification_set_sound_feature_flag_creates_file_with_sound_true(t *testing.T) {
	tmpDir := t.TempDir()

	snippet := notificationSnippet(t,
		fmt.Sprintf(`set_sound_feature_flag "claude" %q true`, tmpDir))

	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)

	featuresFile := filepath.Join(tmpDir, "claude-features.json")
	if _, err := os.Stat(featuresFile); os.IsNotExist(err) {
		t.Fatalf("expected features file to be created at %s", featuresFile)
	}

	// Verify sound is true using python3
	verifySnippet := fmt.Sprintf(
		`python3 -c "import json; print(json.load(open('%s'))['sound'])"`, featuresFile)
	out, code := runBashSnippet(t, verifySnippet, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "True" {
		t.Errorf("expected 'True', got %q", strings.TrimSpace(out))
	}
}

func TestNotification_set_sound_feature_flag_sets_sound_false_in_existing_file(t *testing.T) {
	tmpDir := t.TempDir()
	writeTempFile(t, tmpDir, "claude-features.json", `{"sound": true}`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`set_sound_feature_flag "claude" %q false`, tmpDir))

	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)

	featuresFile := filepath.Join(tmpDir, "claude-features.json")
	verifySnippet := fmt.Sprintf(
		`python3 -c "import json; print(json.load(open('%s'))['sound'])"`, featuresFile)
	out, code := runBashSnippet(t, verifySnippet, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "False" {
		t.Errorf("expected 'False', got %q", strings.TrimSpace(out))
	}
}

func TestNotification_set_sound_feature_flag_preserves_other_keys(t *testing.T) {
	tmpDir := t.TempDir()
	writeTempFile(t, tmpDir, "claude-features.json", `{"sound": false, "other": 42}`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`set_sound_feature_flag "claude" %q true`, tmpDir))

	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)

	featuresFile := filepath.Join(tmpDir, "claude-features.json")
	verifySnippet := fmt.Sprintf(
		`python3 -c "import json; d=json.load(open('%s')); print(d['sound'], d['other'])"`, featuresFile)
	out, code := runBashSnippet(t, verifySnippet, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "True 42" {
		t.Errorf("expected 'True 42', got %q", strings.TrimSpace(out))
	}
}

// --- toggle_sound_notification ---

func TestNotification_toggle_sound_notification_enables_for_claude(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	os.MkdirAll(configDir, 0755)
	writeTempFile(t, configDir, "claude-features.json", `{"sound": false}`)
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{}`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`toggle_sound_notification "claude" %q %q`, configDir, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "enabled")

	// Verify feature flag was set
	featuresFile := filepath.Join(configDir, "claude-features.json")
	verifySnippet := fmt.Sprintf(
		`python3 -c "import json; print(json.load(open('%s'))['sound'])"`, featuresFile)
	flagOut, flagCode := runBashSnippet(t, verifySnippet, nil)
	assertExitCode(t, flagCode, 0)
	if strings.TrimSpace(flagOut) != "True" {
		t.Errorf("expected feature flag 'True', got %q", strings.TrimSpace(flagOut))
	}

	// Verify hook was added
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	assertContains(t, string(data), "Stop")
}

func TestNotification_toggle_sound_notification_disables_for_claude(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	os.MkdirAll(configDir, 0755)
	writeTempFile(t, configDir, "claude-features.json", `{"sound": true}`)
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{
  "hooks": {
    "Stop": [
      {
        "hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Bottle.aiff &"}]
      }
    ]
  }
}
`)

	snippet := notificationSnippet(t,
		fmt.Sprintf(`toggle_sound_notification "claude" %q %q`, configDir, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "disabled")

	// Verify feature flag was set to false
	featuresFile := filepath.Join(configDir, "claude-features.json")
	verifySnippet := fmt.Sprintf(
		`python3 -c "import json; print(json.load(open('%s'))['sound'])"`, featuresFile)
	flagOut, flagCode := runBashSnippet(t, verifySnippet, nil)
	assertExitCode(t, flagCode, 0)
	if strings.TrimSpace(flagOut) != "False" {
		t.Errorf("expected feature flag 'False', got %q", strings.TrimSpace(flagOut))
	}

	// Verify hook was removed
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	assertNotContains(t, string(data), "afplay")
}

// ==================== update.sh tests ====================

// --- check_for_update: no brew ---

func TestUpdate_check_for_update_returns_early_when_brew_not_found(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	// Use PATH=/nonexistent so command -v brew fails
	snippet := updateSnippet(t, fmt.Sprintf(
		`UPDATE_CACHE=%q; _update_version=""; PATH="/nonexistent" check_for_update; echo "_update_version=$_update_version"`,
		cacheFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "_update_version=")
	// _update_version should be empty (just "_update_version=\n" or "_update_version=")
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "_update_version=") {
			val := strings.TrimPrefix(line, "_update_version=")
			if val != "" {
				t.Errorf("expected empty _update_version, got %q", val)
			}
		}
	}
}

// --- check_for_update: fresh cache ---

func TestUpdate_check_for_update_reads_fresh_cache_with_newer_version(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	now := fmt.Sprintf("%d", time.Now().Unix())
	writeTempFile(t, tmpDir, ".update-check", fmt.Sprintf("2.0.0\n%s\n", now))

	// Mock brew to return installed version 1.0.0
	brewBody := `
if [ "$1" = "list" ] && [ "$2" = "--versions" ]; then
  echo "ghost-tab 1.0.0"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update; echo "_update_version=$_update_version"`, cacheFile))

	out, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
	assertContains(t, out, "_update_version=2.0.0")
}

func TestUpdate_check_for_update_ignores_cache_when_versions_match(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	now := fmt.Sprintf("%d", time.Now().Unix())
	writeTempFile(t, tmpDir, ".update-check", fmt.Sprintf("1.0.0\n%s\n", now))

	brewBody := `
if [ "$1" = "list" ] && [ "$2" = "--versions" ]; then
  echo "ghost-tab 1.0.0"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update; echo "_update_version=$_update_version"`, cacheFile))

	out, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
	// _update_version should be empty
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "_update_version=") {
			val := strings.TrimPrefix(line, "_update_version=")
			if val != "" {
				t.Errorf("expected empty _update_version, got %q", val)
			}
		}
	}
}

// --- check_for_update: empty cache version ---

func TestUpdate_check_for_update_handles_empty_version_in_cache(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	now := fmt.Sprintf("%d", time.Now().Unix())
	writeTempFile(t, tmpDir, ".update-check", fmt.Sprintf("\n%s\n", now))

	brewBody := `exit 0`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update; echo "_update_version=$_update_version"`, cacheFile))

	out, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "_update_version=") {
			val := strings.TrimPrefix(line, "_update_version=")
			if val != "" {
				t.Errorf("expected empty _update_version, got %q", val)
			}
		}
	}
}

// --- Network failure and timeout scenarios ---

func TestUpdate_check_for_update_handles_brew_outdated_network_timeout(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  echo "Error: Operation timed out" >&2
  exit 1
fi
if [[ "$*" == *"list --versions"* ]]; then
  echo "ghost-tab 1.0.0"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_brew_command_hanging(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  sleep 5 &
  exit 1
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_connection_refused(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  echo "Error: Failed to connect to github.com: Connection refused" >&2
  exit 7
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_DNS_resolution_failure(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  echo "Error: Could not resolve host: github.com" >&2
  exit 6
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_HTTP_404(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  echo "Error: 404: Not Found" >&2
  exit 1
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_HTTP_503(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  echo "Error: 503 Service Unavailable" >&2
  exit 1
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

// --- Invalid version string scenarios ---

func TestUpdate_check_for_update_handles_malformed_outdated_output(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  echo "CORRUPT@#\$%DATA"
  exit 0
fi
if [[ "$*" == *"list --versions"* ]]; then
  echo "ghost-tab 1.0.0"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_unparseable_installed_version(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	now := fmt.Sprintf("%d", time.Now().Unix())
	writeTempFile(t, tmpDir, ".update-check", fmt.Sprintf("2.0.0\n%s\n", now))

	brewBody := `
if [[ "$*" == *"list --versions"* ]]; then
  echo "ghost-tab INVALID_VERSION_STRING"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_empty_brew_list_output(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	now := fmt.Sprintf("%d", time.Now().Unix())
	writeTempFile(t, tmpDir, ".update-check", fmt.Sprintf("2.0.0\n%s\n", now))

	brewBody := `
if [[ "$*" == *"list --versions"* ]]; then
  echo ""
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_outdated_output_missing_version_number(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  echo "ghost-tab (1.0.0) <"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_outdated_returning_multiple_packages(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  echo "other-pkg (1.0.0) < 2.0.0"
  echo "ghost-tab (1.0.0) < 1.5.0"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_version_with_special_characters(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  echo "ghost-tab (1.0.0) < 2.0.0-beta+build.123"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

// --- Malformed cache file scenarios ---

func TestUpdate_check_for_update_handles_corrupted_cache_file(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	writeTempFile(t, tmpDir, ".update-check", "2.0.0\nNOT_A_NUMBER\n")

	brewBody := `
if [[ "$*" == *"list --versions"* ]]; then
  echo "ghost-tab 1.0.0"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_cache_with_only_one_line(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	writeTempFile(t, tmpDir, ".update-check", "2.0.0\n")

	brewBody := `
if [[ "$*" == *"list --versions"* ]]; then
  echo "ghost-tab 1.0.0"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_cache_with_future_timestamp(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	writeTempFile(t, tmpDir, ".update-check", "2.0.0\n9999999999\n")

	brewBody := `exit 0`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_cache_with_negative_timestamp(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	writeTempFile(t, tmpDir, ".update-check", "2.0.0\n-1000\n")

	brewBody := `exit 0`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_cache_directory_creation_failure(t *testing.T) {
	tmpDir := t.TempDir()
	readonlyDir := filepath.Join(tmpDir, "readonly")
	os.MkdirAll(readonlyDir, 0755)
	os.Chmod(readonlyDir, 0444)
	cacheFile := filepath.Join(readonlyDir, ".update-check")

	brewBody := `
if [[ "$*" == *"outdated"* ]]; then
  echo "ghost-tab (1.0.0) < 2.0.0"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	// Should not crash even if cache write fails
	assertExitCode(t, code, 0)

	// Restore permissions for cleanup
	os.Chmod(readonlyDir, 0755)
}

func TestUpdate_check_for_update_handles_cache_with_non_ASCII_characters(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	now := fmt.Sprintf("%d", time.Now().Unix())
	// Write "2.0.0\xe2\x9c\x93\n<timestamp>\n" (UTF-8 checkmark in the version line)
	content := fmt.Sprintf("2.0.0\xe2\x9c\x93\n%s\n", now)
	writeTempFile(t, tmpDir, ".update-check", content)

	brewBody := `
if [[ "$*" == *"list --versions"* ]]; then
  echo "ghost-tab 1.0.0"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

// --- Command not found scenarios ---

func TestUpdate_check_for_update_returns_early_when_brew_not_in_PATH(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	// Build env with PATH=/nonexistent so brew cannot be found
	env := buildEnv(t, nil,
		"PATH=/nonexistent",
		fmt.Sprintf("UPDATE_CACHE=%s", cacheFile),
	)

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update; echo "_update_version=$_update_version"`, cacheFile))

	out, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "_update_version=") {
			val := strings.TrimPrefix(line, "_update_version=")
			if val != "" {
				t.Errorf("expected empty _update_version, got %q", val)
			}
		}
	}
}

func TestUpdate_check_for_update_handles_brew_returning_127(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")

	brewBody := `exit 127`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	assertExitCode(t, code, 0)
}

func TestUpdate_check_for_update_handles_corrupted_cache_that_sed_cannot_parse(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".update-check")
	now := fmt.Sprintf("%d", time.Now().Unix())
	// Write binary data that sed might fail on
	content := fmt.Sprintf("\xff\xfe2.0.0\n%s\n", now)
	writeTempFile(t, tmpDir, ".update-check", content)

	brewBody := `
if [[ "$*" == *"list --versions"* ]]; then
  echo "ghost-tab 1.0.0"
  exit 0
fi
exit 0
`
	binDir := mockCommand(t, tmpDir, "brew", brewBody)
	env := buildEnv(t, []string{binDir}, fmt.Sprintf("UPDATE_CACHE=%s", cacheFile))

	snippet := updateSnippet(t,
		fmt.Sprintf(`UPDATE_CACHE=%q; _update_version=""; check_for_update`, cacheFile))

	_, code := runBashSnippet(t, snippet, env)
	// Should handle gracefully even with corrupt cache
	assertExitCode(t, code, 0)
}
