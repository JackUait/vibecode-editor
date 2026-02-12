package bash_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// settingsJsonSnippet builds a bash snippet that sources tui.sh and settings-json.sh,
// then runs the provided bash code.
func settingsJsonSnippet(t *testing.T, body string) string {
	t.Helper()
	root := projectRoot(t)
	tuiPath := filepath.Join(root, "lib", "tui.sh")
	settingsJsonPath := filepath.Join(root, "lib", "settings-json.sh")
	return fmt.Sprintf("source %q && source %q && %s", tuiPath, settingsJsonPath, body)
}

func TestSettingsJson_add_sound_notification_hook_migrates_old_notification_idle_prompt_to_stop(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{
  "hooks": {
    "Notification": [
      {
        "matcher": "idle_prompt",
        "hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Bottle.aiff &"}]
      }
    ]
  }
}
`)

	snippet := settingsJsonSnippet(t,
		fmt.Sprintf(`add_sound_notification_hook %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "added")

	// Read the resulting file and verify migration
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	content := string(data)
	assertContains(t, content, `"Stop"`)
	assertNotContains(t, content, "idle_prompt")
}

func TestSettingsJson_add_sound_notification_hook_migration_preserves_other_notification_hooks(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{
  "hooks": {
    "Notification": [
      {
        "matcher": "idle_prompt",
        "hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Bottle.aiff &"}]
      },
      {
        "matcher": "permission_prompt",
        "hooks": [{"type": "command", "command": "echo permission"}]
      }
    ]
  }
}
`)

	snippet := settingsJsonSnippet(t,
		fmt.Sprintf(`add_sound_notification_hook %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "added")

	// Read the resulting file and verify migration preserves other hooks
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	content := string(data)
	assertContains(t, content, `"Stop"`)
	assertContains(t, content, "permission_prompt")
	assertNotContains(t, content, "idle_prompt")
}

// --- Additional coverage for functions in settings-json.sh ---

func TestSettingsJson_add_sound_notification_hook_creates_file_when_missing(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	snippet := settingsJsonSnippet(t,
		fmt.Sprintf(`add_sound_notification_hook %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "added")

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("settings.json should have been created: %v", err)
	}
	content := string(data)
	assertContains(t, content, `"Stop"`)
	assertContains(t, content, "afplay /System/Library/Sounds/Bottle.aiff &")
}

func TestSettingsJson_add_sound_notification_hook_reports_exists_when_duplicate(t *testing.T) {
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

	snippet := settingsJsonSnippet(t,
		fmt.Sprintf(`add_sound_notification_hook %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, strings.TrimSpace(out), "exists")
}

func TestSettingsJson_remove_sound_notification_hook_removes_existing_hook(t *testing.T) {
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

	snippet := settingsJsonSnippet(t,
		fmt.Sprintf(`remove_sound_notification_hook %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, strings.TrimSpace(out), "removed")

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	content := string(data)
	assertNotContains(t, content, "Bottle.aiff")
}

func TestSettingsJson_remove_sound_notification_hook_returns_not_found_for_missing_file(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "nonexistent.json")

	snippet := settingsJsonSnippet(t,
		fmt.Sprintf(`remove_sound_notification_hook %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, strings.TrimSpace(out), "not_found")
}

func TestSettingsJson_remove_sound_notification_hook_returns_not_found_when_no_match(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{
  "hooks": {
    "Stop": [
      {
        "hooks": [{"type": "command", "command": "echo something_else"}]
      }
    ]
  }
}
`)

	snippet := settingsJsonSnippet(t,
		fmt.Sprintf(`remove_sound_notification_hook %q "afplay /System/Library/Sounds/Bottle.aiff &"`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, strings.TrimSpace(out), "not_found")
}

func TestSettingsJson_merge_claude_settings_creates_file_when_missing(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	snippet := settingsJsonSnippet(t,
		fmt.Sprintf(`merge_claude_settings %q`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "Created Claude settings with status line")

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("settings.json should have been created: %v", err)
	}
	content := string(data)
	assertContains(t, content, `"statusLine"`)
	assertContains(t, content, "statusline-wrapper.sh")
}

func TestSettingsJson_merge_claude_settings_adds_status_line_to_existing(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{
  "hooks": {}
}
`)

	snippet := settingsJsonSnippet(t,
		fmt.Sprintf(`merge_claude_settings %q`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "Added status line to Claude settings")

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	content := string(data)
	assertContains(t, content, `"statusLine"`)
}

func TestSettingsJson_merge_claude_settings_skips_when_already_configured(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := writeTempFile(t, tmpDir, "settings.json", `{
  "statusLine": {
    "type": "command",
    "command": "bash ~/.claude/statusline-wrapper.sh"
  }
}
`)

	snippet := settingsJsonSnippet(t,
		fmt.Sprintf(`merge_claude_settings %q`, settingsFile))

	out, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "already configured")
}
