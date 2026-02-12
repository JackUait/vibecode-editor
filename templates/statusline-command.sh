#!/bin/bash
input=$(cat)
cwd=$(echo "$input" | sed -n 's/.*"current_dir":"\([^"]*\)".*/\1/p')

if git -C "$cwd" rev-parse --git-dir > /dev/null 2>&1; then
  repo_name=$(basename "$cwd")
  branch=$(git -C "$cwd" --no-optional-locks rev-parse --abbrev-ref HEAD 2>/dev/null)

  # Session line diff: compare working tree against baseline SHA
  baseline_file="${GHOST_TAB_BASELINE_FILE:-}"
  if [ -n "$baseline_file" ] && [ -f "$baseline_file" ]; then
    baseline_sha=$(head -1 "$baseline_file" 2>/dev/null)
    if [ -n "$baseline_sha" ]; then
      diff_stats=$(git -C "$cwd" --no-optional-locks diff "$baseline_sha" --numstat 2>/dev/null \
        | awk '{a+=$1; d+=$2} END {print a+0, d+0}')
      added=$(echo "$diff_stats" | cut -d' ' -f1)
      deleted=$(echo "$diff_stats" | cut -d' ' -f2)
    fi
  fi

  if [ -n "${added:-}" ]; then
    printf '\033[01;36m%s\033[00m | \033[01;32m%s\033[00m | \033[01;32m+%s\033[00m / \033[01;31m-%s\033[00m' \
      "$repo_name" "$branch" "$added" "$deleted"
  else
    printf '\033[01;36m%s\033[00m | \033[01;32m%s\033[00m' \
      "$repo_name" "$branch"
  fi
else
  printf '\033[01;36m%s\033[00m' "$(basename "$cwd")"
fi
