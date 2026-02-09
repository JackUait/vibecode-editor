#!/bin/bash
# Process management helpers — no side effects on source.

# Recursively kills a process tree (depth-first: children first, then parent).
kill_tree() {
  local pid=$1
  local sig=${2:-TERM}
  for child in $(pgrep -P "$pid" 2>/dev/null); do
    kill_tree "$child" "$sig"
  done
  kill -"$sig" "$pid" 2>/dev/null
  return 0
}
