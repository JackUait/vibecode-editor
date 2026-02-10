#!/bin/bash
# Package installation helpers for the installer.

# Install base requirements (tmux, jq, ghostty).
ensure_base_requirements() {
  ensure_command "tmux" "brew install tmux" "" "tmux"
  ensure_command "jq" "brew install jq" "" "jq"
  ensure_command "ghostty" "brew install --cask ghostty" "" "Ghostty"
}

# Install a Homebrew formula if not already present.
ensure_brew_pkg() {
  local pkg="$1"
  if brew list "$pkg" &>/dev/null; then
    success "$pkg already installed"
  else
    info "Installing $pkg..."
    if brew install "$pkg"; then
      success "$pkg installed"
    else
      warn "Failed to install $pkg — install it manually with: brew install $pkg"
    fi
  fi
}

# Install a Homebrew cask if the .app isn't in /Applications.
ensure_cask() {
  local cask="$1" app_name="$2"
  if [ -d "/Applications/${app_name}.app" ]; then
    success "$app_name found"
  else
    info "Installing $app_name..."
    if brew install --cask "$cask"; then
      success "$app_name installed"
    else
      error "$app_name installation failed."
      info "Install manually from https://${cask}.org or run: brew install --cask $cask"
      exit 1
    fi
  fi
}

# Install a command-line tool if not already on PATH.
# Usage: ensure_command "cmd" "install_cmd" "post_msg" "display_name"
ensure_command() {
  local cmd="$1" install_cmd="$2" post_msg="$3" display_name="$4"
  if command -v "$cmd" &>/dev/null; then
    success "$display_name already installed"
  else
    info "Installing $display_name..."
    if eval "$install_cmd"; then
      success "$display_name installed"
      [ -n "$post_msg" ] && info "$post_msg"
    else
      warn "$display_name installation failed — install manually: $install_cmd"
    fi
  fi
}
