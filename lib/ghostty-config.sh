#!/bin/bash
# Ghostty config file helpers.

# Merge a command line into an existing Ghostty config.
# If a "command = ..." line exists, replace it; otherwise append.
merge_ghostty_config() {
  local config_path="$1" wrapper_line="$2"
  if grep -q '^command[[:space:]]*=' "$config_path"; then
    sed -i '' 's|^command[[:space:]]*=.*|'"$wrapper_line"'|' "$config_path"
    success "Replaced existing command line in config"
  else
    echo "$wrapper_line" >> "$config_path"
    success "Appended wrapper command to config"
  fi
}

# Backup an existing config and replace it with the source config.
backup_replace_ghostty_config() {
  local config_path="$1" source_config="$2"
  local backup
  backup="${config_path}.backup.$(date +%s)"
  cp "$config_path" "$backup"
  success "Backed up existing config to $backup"
  cp "$source_config" "$config_path"
  success "Replaced config with ghost-tab defaults"
}
