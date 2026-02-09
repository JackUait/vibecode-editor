#!/bin/bash

# Get project name from PWD or tmux session
PROJECT="${PWD##*/}"
if command -v tmux &>/dev/null && tmux list-sessions &>/dev/null; then
  SESSION_NAME=$(tmux display-message -p '#S' 2>/dev/null)
  [ -n "$SESSION_NAME" ] && PROJECT="$SESSION_NAME"
fi

# PID file for this project
PID_FILE="/tmp/ghost-tab-spinner-${PROJECT}.pid"

# Kill spinner if running
if [ -f "$PID_FILE" ]; then
  PID=$(cat "$PID_FILE")
  kill "$PID" 2>/dev/null
  rm -f "$PID_FILE"
fi

# Restore normal tab title
printf '\033]0;%s\007' "$PROJECT"
