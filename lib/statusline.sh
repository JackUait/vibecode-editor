#!/bin/bash
# Statusline helper functions — pure, no side effects on source.

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
