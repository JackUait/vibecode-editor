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

# Returns total RSS in KB for a process and all its descendants.
# Usage: get_tree_rss_kb 12345  =>  "92160"
get_tree_rss_kb() {
  local root_pid="$1"
  local total=0
  local queue=("$root_pid")

  while [ ${#queue[@]} -gt 0 ]; do
    local pid="${queue[0]}"
    queue=("${queue[@]:1}")

    local rss
    rss=$(ps -o rss= -p "$pid" 2>/dev/null | tr -d ' ')
    if [ -n "$rss" ] && [ "$rss" -gt 0 ] 2>/dev/null; then
      total=$((total + rss))
    fi

    local children
    children=$(pgrep -P "$pid" 2>/dev/null) || true
    if [ -n "$children" ]; then
      while IFS= read -r child; do
        queue+=("$child")
      done <<< "$children"
    fi
  done

  echo "$total"
}

# Extracts the current_dir value from a JSON string.
# Uses sed — no jq dependency.
parse_cwd_from_json() {
  echo "$1" | sed -n 's/.*"current_dir":"\([^"]*\)".*/\1/p'
}
