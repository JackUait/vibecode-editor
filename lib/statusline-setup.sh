#!/bin/bash
# Statusline setup — install ccstatusline, copy configs and scripts.
# Depends on: tui.sh (success, warn, info), settings-json.sh (merge_claude_settings)

# Check whether npm is available. Extracted for testability.
_has_npm() { command -v npm &>/dev/null; }

# Install and configure the Claude Code status line.
# Usage: setup_statusline <share_dir> <claude_settings_path> <home_dir>
setup_statusline() {
  local share_dir="$1" claude_settings_path="$2" home_dir="$3"

  # Check for npm, install Node.js LTS if needed
  if ! _has_npm; then
    info "Installing Node.js LTS..."
    if brew install node@22 &>/dev/null; then
      export PATH="/opt/homebrew/opt/node@22/bin:$PATH"
      success "Node.js LTS installed"
    else
      warn "Node.js installation failed — skipping status line setup"
      return 0
    fi
  fi

  if ! _has_npm; then
    return 0
  fi

  # Install ccstatusline
  if npm list -g ccstatusline &>/dev/null; then
    success "ccstatusline already installed"
  else
    info "Installing ccstatusline..."
    if npm install -g ccstatusline &>/dev/null; then
      success "ccstatusline installed"
    else
      warn "Failed to install ccstatusline — skipping status line setup"
      return 0
    fi
  fi

  if npm list -g ccstatusline &>/dev/null; then
    # Create ccstatusline config
    mkdir -p "$home_dir/.config/ccstatusline"
    cp "$share_dir/templates/ccstatusline-settings.json" "$home_dir/.config/ccstatusline/settings.json"
    success "Created ccstatusline config"

    # Create statusline scripts
    mkdir -p "$home_dir/.claude"
    cp "$share_dir/templates/statusline-command.sh" "$home_dir/.claude/statusline-command.sh"
    cp "$share_dir/templates/statusline-wrapper.sh" "$home_dir/.claude/statusline-wrapper.sh"
    cp "$share_dir/lib/statusline.sh" "$home_dir/.claude/statusline-helpers.sh"
    chmod +x "$home_dir/.claude/statusline-command.sh"
    chmod +x "$home_dir/.claude/statusline-wrapper.sh"
    success "Created statusline scripts"

    # Update Claude settings.json
    merge_claude_settings "$claude_settings_path"
  fi
}
