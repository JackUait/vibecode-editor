#!/bin/bash
# Statusline helper functions — pure, no side effects on source.

# Converts kilobytes to human-readable memory string.
# Usage: format_memory 512000  =>  "500M"
# Usage: format_memory 1572864 =>  "1.5G"
format_memory() {
  local mem_kb="$1"
  local mem_mb=$((mem_kb / 1024))
  if [ "$mem_mb" -ge 1024 ]; then
    local mem_gb
    mem_gb=$(echo "scale=1; $mem_mb / 1024" | bc)
    echo "${mem_gb}G"
  else
    echo "${mem_mb}M"
  fi
}

# Extracts the current_dir value from a JSON string.
# Uses sed — no jq dependency.
parse_cwd_from_json() {
  echo "$1" | sed -n 's/.*"current_dir":"\([^"]*\)".*/\1/p'
}
