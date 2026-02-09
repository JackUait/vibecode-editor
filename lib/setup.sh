#!/bin/bash
# Setup helper functions — pure, no side effects on source.

# Determines the SHARE_DIR (where supporting files live).
# Usage: resolve_share_dir "$SCRIPT_DIR" "$BREW_PREFIX"
# When script is in $BREW_PREFIX/bin, returns $BREW_PREFIX/share/ghost-tab.
# Otherwise, returns the parent of the script directory.
resolve_share_dir() {
  local script_dir="$1"
  local brew_prefix="$2"
  if [[ -n "$brew_prefix" && "$script_dir" == "$brew_prefix/bin" ]]; then
    echo "$brew_prefix/share/ghost-tab"
  else
    (cd "$script_dir/.." && pwd)
  fi
}
