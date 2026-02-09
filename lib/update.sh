#!/bin/bash
# Version update check — Homebrew only.

check_for_update() {
  local cache_ts now age latest
  # Only check if brew is available (Homebrew install)
  command -v brew &>/dev/null || return 0

  # Read cache if it exists
  if [ -f "$UPDATE_CACHE" ]; then
    latest="$(sed -n '1p' "$UPDATE_CACHE")"
    cache_ts="$(sed -n '2p' "$UPDATE_CACHE")"
    now="$(date +%s)"
    age=$(( now - ${cache_ts:-0} ))
    # Use cached result if less than 24 hours old
    if [ "$age" -lt 86400 ]; then
      # Verify cached version is actually newer than installed
      if [ -n "$latest" ]; then
        installed="$(brew list --versions ghost-tab 2>/dev/null | awk '{print $2}')"
        if [ "$latest" != "$installed" ]; then
          _update_version="$latest"
        fi
      fi
      return
    fi
  fi

  # Spawn background check (non-blocking)
  (
    result="$(brew outdated --verbose --formula ghost-tab 2>/dev/null)"
    mkdir -p "$(dirname "$UPDATE_CACHE")"
    if [ -n "$result" ]; then
      # Extract new version: "ghost-tab (1.0.0) < 1.1.0" -> "1.1.0"
      new_ver="$(echo "$result" | sed -n 's/.*< *//p')"
      printf '%s\n%s\n' "$new_ver" "$(date +%s)" > "$UPDATE_CACHE.tmp"
      mv "$UPDATE_CACHE.tmp" "$UPDATE_CACHE"
    else
      printf '\n%s\n' "$(date +%s)" > "$UPDATE_CACHE.tmp"
      mv "$UPDATE_CACHE.tmp" "$UPDATE_CACHE"
    fi
  ) &
  disown
}
