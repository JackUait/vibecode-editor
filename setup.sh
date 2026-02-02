#!/bin/bash
set -e

# Colors (with fallback)
if [ -t 1 ] && [ "$(tput colors 2>/dev/null)" -ge 8 ] 2>/dev/null; then
  GREEN='\033[0;32m'
  YELLOW='\033[0;33m'
  RED='\033[0;31m'
  BLUE='\033[0;34m'
  BOLD='\033[1m'
  NC='\033[0m'
else
  GREEN='' YELLOW='' RED='' BLUE='' BOLD='' NC=''
fi

success() { echo -e "${GREEN}✓${NC} $1"; }
warn()    { echo -e "${YELLOW}!${NC} $1"; }
error()   { echo -e "${RED}✗${NC} $1"; }
info()    { echo -e "${BLUE}→${NC} $1"; }
header()  { echo -e "\n${BOLD}$1${NC}"; }

# ---------- Embedded files ----------

WRAPPER_SCRIPT='#!/bin/bash
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

TMUX_CMD="$(command -v tmux)"
LAZYGIT_CMD="$(command -v lazygit)"
BROOT_CMD="$(command -v broot)"
CLAUDE_CMD="$(command -v claude)"

# Load user projects from config file if it exists
PROJECTS_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/vibecode-editor/projects"

# Select working directory
if [ -n "$1" ] && [ -d "$1" ]; then
  cd "$1"
  shift
elif [ -z "$1" ]; then
  projects=()
  if [ -f "$PROJECTS_FILE" ]; then
    while IFS= read -r line; do
      [[ -z "$line" || "$line" == \#* ]] && continue
      projects+=("$line")
    done < "$PROJECTS_FILE"
  fi

  if [ ${#projects[@]} -gt 0 ]; then
    echo "Select project:"
    for i in "${!projects[@]}"; do
      name="${projects[$i]%%:*}"
      printf "  %d) %s\n" $((i+1)) "$name"
    done
    printf "  0) current directory\n"
    read -rn1 -p "> " choice
    echo
    if [[ "$choice" =~ ^[1-9][0-9]*$ ]] && [ "$choice" -le "${#projects[@]}" ]; then
      dir="${projects[$((choice-1))]#*:}"
      cd "$dir"
    fi
  fi
fi

export PROJECT_DIR="$(pwd)"
SESSION_NAME="dev-$(basename "$PROJECT_DIR")-$$"

# Background watcher: switch to Claude pane once it'\''s ready
(
  while true; do
    sleep 0.5
    content=$("$TMUX_CMD" capture-pane -t "$SESSION_NAME:0.1" -p 2>/dev/null)
    if echo "$content" | grep -q '\''>'\''; then
      "$TMUX_CMD" select-pane -t "$SESSION_NAME:0.1"
      break
    fi
  done
) &
WATCHER_PID=$!

# Kill all processes in the tmux session when the terminal closes
cleanup() {
  # Get PIDs of all pane shell processes in this session, then kill their entire process trees
  for pane_pid in $("$TMUX_CMD" list-panes -s -t "$SESSION_NAME" -F '\''#{pane_pid}'\'' 2>/dev/null); do
    pkill -TERM -P "$pane_pid" 2>/dev/null
  done
  kill $WATCHER_PID 2>/dev/null
  "$TMUX_CMD" kill-session -t "$SESSION_NAME" 2>/dev/null
}
trap cleanup EXIT HUP TERM INT

"$TMUX_CMD" new-session -s "$SESSION_NAME" -e "PATH=$PATH" -c "$PROJECT_DIR" \
  "$LAZYGIT_CMD; exec bash" \; \
  split-window -h -p 50 -c "$PROJECT_DIR" \
  "$CLAUDE_CMD $*; exec bash" \; \
  select-pane -t 0 \; \
  split-window -v -p 50 -c "$PROJECT_DIR" \
  "while true; do $BROOT_CMD $PROJECT_DIR; done" \; \
  split-window -v -p 30 -c "$PROJECT_DIR" \; \
  select-pane -t 3'

GHOSTTY_DEFAULT_CONFIG='keybind = cmd+shift+left=previous_tab
keybind = cmd+shift+right=next_tab
keybind = cmd+t=new_tab
macos-option-as-alt = left
command = ~/.config/ghostty/claude-wrapper.sh'

# ---------- OS check ----------
header "Checking platform..."
if [ "$(uname)" != "Darwin" ]; then
  error "This setup script only supports macOS."
  exit 1
fi
success "macOS detected"

# ---------- Homebrew ----------
header "Checking Homebrew..."
if ! command -v brew &>/dev/null; then
  info "Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  eval "$(/opt/homebrew/bin/brew shellenv 2>/dev/null || /usr/local/bin/brew shellenv 2>/dev/null)"
  success "Homebrew installed"
else
  success "Homebrew found"
fi

# ---------- Dependencies ----------
header "Installing dependencies..."
for pkg in tmux lazygit broot claude; do
  brew_pkg="$pkg"
  [ "$pkg" = "claude" ] && brew_pkg="claude-code"

  if brew list "$brew_pkg" &>/dev/null; then
    success "$pkg already installed"
  else
    info "Installing $pkg..."
    if brew install "$brew_pkg"; then
      success "$pkg installed"
      [ "$pkg" = "claude" ] && info "Run 'claude' to authenticate before opening Ghostty."
    else
      if [ "$pkg" = "claude" ]; then
        error "Claude Code installation failed."
        info "Install manually: brew install claude-code"
        info "Then run 'claude' to authenticate before re-running this script."
        exit 1
      else
        warn "Failed to install $pkg — install it manually with: brew install $brew_pkg"
      fi
    fi
  fi
done

# ---------- Ghostty ----------
header "Checking Ghostty..."
if [ -d "/Applications/Ghostty.app" ]; then
  success "Ghostty found"
else
  warn "Ghostty not found in /Applications. Config files will still be set up."
  warn "Install Ghostty from https://ghostty.org"
fi

# ---------- Wrapper script ----------
header "Setting up wrapper script..."
mkdir -p ~/.config/ghostty
echo "$WRAPPER_SCRIPT" > ~/.config/ghostty/claude-wrapper.sh
chmod +x ~/.config/ghostty/claude-wrapper.sh
success "Created claude-wrapper.sh in ~/.config/ghostty/"

# ---------- Ghostty config ----------
header "Setting up Ghostty config..."
GHOSTTY_CONFIG="$HOME/.config/ghostty/config"
WRAPPER_LINE="command = ~/.config/ghostty/claude-wrapper.sh"

if [ -f "$GHOSTTY_CONFIG" ]; then
  warn "Existing Ghostty config found at $GHOSTTY_CONFIG"
  echo ""
  echo -e "  ${BOLD}1)${NC} Merge — add the wrapper command to your existing config"
  echo -e "  ${BOLD}2)${NC} Backup & replace — save current config and use ours"
  echo ""
  read -rn1 -p "$(echo -e "${BLUE}Choose (1/2):${NC} ")" config_choice </dev/tty
  echo ""

  case "$config_choice" in
    1)
      if grep -q '^command\s*=' "$GHOSTTY_CONFIG"; then
        sed -i '' 's|^command\s*=.*|'"$WRAPPER_LINE"'|' "$GHOSTTY_CONFIG"
        success "Replaced existing command line in config"
      else
        echo "$WRAPPER_LINE" >> "$GHOSTTY_CONFIG"
        success "Appended wrapper command to config"
      fi
      ;;
    2)
      BACKUP="$GHOSTTY_CONFIG.backup.$(date +%s)"
      cp "$GHOSTTY_CONFIG" "$BACKUP"
      success "Backed up existing config to $BACKUP"
      echo "$GHOSTTY_DEFAULT_CONFIG" > "$GHOSTTY_CONFIG"
      success "Replaced config with vibecode-editor defaults"
      ;;
    *)
      warn "Invalid choice, skipping config setup"
      ;;
  esac
else
  echo "$GHOSTTY_DEFAULT_CONFIG" > "$GHOSTTY_CONFIG"
  success "Created Ghostty config at $GHOSTTY_CONFIG"
fi

# ---------- Projects ----------
header "Setting up projects..."
PROJECTS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/vibecode-editor"
PROJECTS_FILE="$PROJECTS_DIR/projects"
mkdir -p "$PROJECTS_DIR"

echo ""
read -rn1 -p "$(echo -e "${BLUE}Add a project? (y/n):${NC} ")" add_project </dev/tty
echo ""

while [[ "$add_project" =~ ^[yY]$ ]]; do
  read -rp "$(echo -e "${BLUE}Project name:${NC} ")" proj_name </dev/tty
  read -rp "$(echo -e "${BLUE}Project path:${NC} ")" proj_path </dev/tty

  # Expand ~ to $HOME for validation
  expanded_path="${proj_path/#\~/$HOME}"

  if [ -d "$expanded_path" ]; then
    echo "$proj_name:$expanded_path" >> "$PROJECTS_FILE"
    success "Added $proj_name"
  else
    warn "Path $proj_path does not exist yet — adding anyway"
    echo "$proj_name:$expanded_path" >> "$PROJECTS_FILE"
  fi

  echo ""
  read -rn1 -p "$(echo -e "${BLUE}Add another? (y/n):${NC} ")" add_project </dev/tty
  echo ""
done

if [ -f "$PROJECTS_FILE" ] && [ -s "$PROJECTS_FILE" ]; then
  success "Projects saved to $PROJECTS_FILE"
else
  info "No projects added. Add them later to $PROJECTS_FILE"
fi

# ---------- Summary ----------
header "Setup complete!"
echo ""
success "Wrapper script: ~/.config/ghostty/claude-wrapper.sh"
success "Ghostty config:  ~/.config/ghostty/config"
success "Projects file:   $PROJECTS_FILE"
echo ""
info "Open a new Ghostty window to start coding."
