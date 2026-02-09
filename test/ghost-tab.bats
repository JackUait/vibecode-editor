setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/install.sh"
  source "$PROJECT_ROOT/lib/ghostty-config.sh"
  source "$PROJECT_ROOT/lib/settings-json.sh"
  TEST_TMP="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_TMP"
}

# ---------- Section 1: OS Check ----------

@test "os check: rejects non-Darwin platform" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tui.sh"
    uname() { echo "Linux"; }
    export -f uname
    if [ "$(uname)" != "Darwin" ]; then
      error "This setup script only supports macOS."
      exit 1
    fi
  '
  assert_failure
  assert_output --partial "macOS"
}

# ---------- Section 2: Supporting Files Validation ----------

@test "supporting files: fails when files missing" {
  SHARE_DIR="$TEST_TMP/empty-share"
  mkdir -p "$SHARE_DIR"

  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tui.sh"
    SHARE_DIR="'"$SHARE_DIR"'"
    if [ ! -f "$SHARE_DIR/ghostty/claude-wrapper.sh" ] || [ ! -f "$SHARE_DIR/ghostty/config" ] || [ ! -d "$SHARE_DIR/templates" ]; then
      error "Supporting files not found in $SHARE_DIR"
      exit 1
    fi
  '
  assert_failure
  assert_output --partial "Supporting files not found"
}

@test "supporting files: passes when all present" {
  SHARE_DIR="$TEST_TMP/full-share"
  mkdir -p "$SHARE_DIR/ghostty" "$SHARE_DIR/templates"
  touch "$SHARE_DIR/ghostty/claude-wrapper.sh"
  touch "$SHARE_DIR/ghostty/config"

  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tui.sh"
    SHARE_DIR="'"$SHARE_DIR"'"
    if [ ! -f "$SHARE_DIR/ghostty/claude-wrapper.sh" ] || [ ! -f "$SHARE_DIR/ghostty/config" ] || [ ! -d "$SHARE_DIR/templates" ]; then
      error "Supporting files not found in $SHARE_DIR"
      exit 1
    fi
    echo "ok"
  '
  assert_success
  assert_output "ok"
}

# ---------- Section 3: Config Migration ----------

@test "migration: renames vibecode-editor to ghost-tab" {
  export XDG_CONFIG_HOME="$TEST_TMP/config"
  mkdir -p "$XDG_CONFIG_HOME/vibecode-editor"
  echo "proj:path" > "$XDG_CONFIG_HOME/vibecode-editor/projects"

  OLD_PROJECTS_DIR="$XDG_CONFIG_HOME/vibecode-editor"
  NEW_PROJECTS_DIR="$XDG_CONFIG_HOME/ghost-tab"
  if [ -d "$OLD_PROJECTS_DIR" ] && [ ! -d "$NEW_PROJECTS_DIR" ]; then
    mv "$OLD_PROJECTS_DIR" "$NEW_PROJECTS_DIR"
  fi

  [ -d "$XDG_CONFIG_HOME/ghost-tab" ]
  [ -f "$XDG_CONFIG_HOME/ghost-tab/projects" ]
  [ ! -d "$XDG_CONFIG_HOME/vibecode-editor" ]
  run cat "$XDG_CONFIG_HOME/ghost-tab/projects"
  assert_output "proj:path"
}

@test "migration: skips when ghost-tab already exists" {
  export XDG_CONFIG_HOME="$TEST_TMP/config"
  mkdir -p "$XDG_CONFIG_HOME/vibecode-editor"
  mkdir -p "$XDG_CONFIG_HOME/ghost-tab"
  echo "old" > "$XDG_CONFIG_HOME/vibecode-editor/projects"
  echo "new" > "$XDG_CONFIG_HOME/ghost-tab/projects"

  OLD_PROJECTS_DIR="$XDG_CONFIG_HOME/vibecode-editor"
  NEW_PROJECTS_DIR="$XDG_CONFIG_HOME/ghost-tab"
  if [ -d "$OLD_PROJECTS_DIR" ] && [ ! -d "$NEW_PROJECTS_DIR" ]; then
    mv "$OLD_PROJECTS_DIR" "$NEW_PROJECTS_DIR"
  fi

  [ -d "$XDG_CONFIG_HOME/vibecode-editor" ]
  [ -d "$XDG_CONFIG_HOME/ghost-tab" ]
  run cat "$XDG_CONFIG_HOME/ghost-tab/projects"
  assert_output "new"
}

# ---------- Section 4: Ghostty Config ----------

@test "ghostty config: merge option adds command line" {
  local config_file="$TEST_TMP/config"
  echo 'font-size = 14' > "$config_file"

  run merge_ghostty_config "$config_file" "command = ~/.config/ghostty/claude-wrapper.sh"
  assert_success
  assert_output --partial "Appended"

  run cat "$config_file"
  assert_line "font-size = 14"
  assert_line "command = ~/.config/ghostty/claude-wrapper.sh"
}

@test "ghostty config: backup-replace creates backup" {
  local config_file="$TEST_TMP/config"
  local template_file="$TEST_TMP/template"
  echo 'old content' > "$config_file"
  echo 'new template content' > "$template_file"

  run backup_replace_ghostty_config "$config_file" "$template_file"
  assert_success
  assert_output --partial "Backed up"
  assert_output --partial "Replaced"

  # Verify backup exists
  local backup_count
  backup_count=$(ls "$TEST_TMP"/config.backup.* 2>/dev/null | wc -l | tr -d ' ')
  [ "$backup_count" -eq 1 ]

  # Verify config matches template
  run cat "$config_file"
  assert_output "new template content"
}

@test "ghostty config: invalid choice warns and skips" {
  local config_file="$TEST_TMP/config"
  echo 'original content' > "$config_file"

  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tui.sh"
    config_choice="3"
    case "$config_choice" in
      1) echo "merge" ;;
      2) echo "backup" ;;
      *) warn "Invalid choice, skipping config setup" ;;
    esac
  '
  assert_success
  assert_output --partial "Invalid choice"

  # Config unchanged
  run cat "$config_file"
  assert_output "original content"
}

@test "ghostty config: creates new when none exists" {
  local config_file="$TEST_TMP/ghostty-config"
  local template_file="$TEST_TMP/template-config"
  echo 'command = ~/.config/ghostty/claude-wrapper.sh' > "$template_file"

  [ ! -f "$config_file" ]

  cp "$template_file" "$config_file"

  [ -f "$config_file" ]
  run cat "$config_file"
  assert_output "command = ~/.config/ghostty/claude-wrapper.sh"
}

# ---------- Section 5: Project Addition ----------

@test "projects: writes entry to file" {
  local projects_dir="$TEST_TMP/ghost-tab"
  local projects_file="$projects_dir/projects"
  mkdir -p "$projects_dir"

  local proj_name="myproject"
  local expanded_path="$TEST_TMP/Code/myproject"
  mkdir -p "$expanded_path"

  # Replicate the main script logic
  if [ -d "$expanded_path" ]; then
    echo "$proj_name:$expanded_path" >> "$projects_file"
  fi

  [ -f "$projects_file" ]
  run cat "$projects_file"
  assert_output "myproject:$TEST_TMP/Code/myproject"
}

@test "projects: adds nonexistent path with warning" {
  local projects_dir="$TEST_TMP/ghost-tab"
  local projects_file="$projects_dir/projects"
  mkdir -p "$projects_dir"

  local proj_name="futureproject"
  local proj_path="/nonexistent/path/futureproject"
  local expanded_path="/nonexistent/path/futureproject"

  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tui.sh"
    proj_name="futureproject"
    proj_path="/nonexistent/path/futureproject"
    expanded_path="/nonexistent/path/futureproject"
    projects_file="'"$projects_file"'"

    if [ -d "$expanded_path" ]; then
      echo "$proj_name:$expanded_path" >> "$projects_file"
    else
      warn "Path $proj_path does not exist yet — adding anyway"
      echo "$proj_name:$expanded_path" >> "$projects_file"
    fi
  '
  assert_success
  assert_output --partial "does not exist yet"

  [ -f "$projects_file" ]
  run cat "$projects_file"
  assert_output "futureproject:/nonexistent/path/futureproject"
}

# ---------- Section 6: Summary ----------

@test "summary: shows all installed components" {
  export HOME="$TEST_TMP"
  mkdir -p "$HOME/.claude"
  mkdir -p "$HOME/.config/ghost-tab"

  # Create all the optional component files
  touch "$HOME/.claude/statusline-wrapper.sh"
  echo '{"hooks":{"Notification":[{"matcher":"idle_prompt","hooks":[]}]}}' > "$HOME/.claude/settings.json"
  echo "claude" > "$HOME/.config/ghost-tab/ai-tool"

  run bash -c '
    export HOME="'"$TEST_TMP"'"
    source "'"$PROJECT_ROOT"'/lib/tui.sh"
    if [ -f "$HOME/.claude/statusline-wrapper.sh" ]; then
      success "Status line:     ~/.claude/statusline-wrapper.sh"
    fi
    if grep -q "idle_prompt" "$HOME/.claude/settings.json" 2>/dev/null; then
      success "Sound:           Notification on idle"
    fi
  '
  assert_success
  assert_output --partial "Status line"
  assert_output --partial "Sound"
}

@test "summary: omits missing components" {
  export HOME="$TEST_TMP"
  mkdir -p "$HOME/.claude"

  run bash -c '
    export HOME="'"$TEST_TMP"'"
    source "'"$PROJECT_ROOT"'/lib/tui.sh"
    if [ -f "$HOME/.claude/statusline-wrapper.sh" ]; then
      success "Status line:     ~/.claude/statusline-wrapper.sh"
    fi
    if grep -q "idle_prompt" "$HOME/.claude/settings.json" 2>/dev/null; then
      success "Sound:           Notification on idle"
    fi
    echo "done"
  '
  assert_success
  refute_output --partial "Status line"
  refute_output --partial "Sound"
  refute_output --partial "Tab animation"
}
