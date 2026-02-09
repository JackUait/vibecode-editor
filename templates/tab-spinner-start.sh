#!/bin/bash

# Get project name from PWD or tmux session
PROJECT="${PWD##*/}"
if command -v tmux &>/dev/null && tmux list-sessions &>/dev/null; then
  SESSION_NAME=$(tmux display-message -p '#S' 2>/dev/null)
  [ -n "$SESSION_NAME" ] && PROJECT="$SESSION_NAME"
fi

# PID file for this project
PID_FILE="/tmp/ghost-tab-spinner-${PROJECT}.pid"

# Exit if already running
if [ -f "$PID_FILE" ]; then
  OLD_PID=$(cat "$PID_FILE")
  if kill -0 "$OLD_PID" 2>/dev/null; then
    exit 0
  fi
  rm -f "$PID_FILE"
fi

# Spinner frames
FRAMES=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)

# Run animation in background
(
  echo $$ > "$PID_FILE"
  i=0
  while true; do
    printf '\033]0;%s %s\007' "${FRAMES[$i]}" "$PROJECT"
    i=$(( (i + 1) % ${#FRAMES[@]} ))
    sleep 0.1
  done
) &

disown
