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

# Resolve script directory (where repo files live)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

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
for pkg in tmux lazygit broot; do
  if brew list "$pkg" &>/dev/null; then
    success "$pkg already installed"
  else
    info "Installing $pkg..."
    if brew install "$pkg"; then
      success "$pkg installed"
    else
      warn "Failed to install $pkg — install it manually with: brew install $pkg"
    fi
  fi
done

# ---------- Claude Code ----------
header "Checking Claude Code..."
if command -v claude &>/dev/null; then
  success "Claude Code found"
else
  warn "Claude Code not found in PATH."
  info "Installing Claude Code..."
  curl -fsSL https://claude.ai/install.sh | sh
  export PATH="$HOME/.local/bin:$PATH"
  if command -v claude &>/dev/null; then
    success "Claude Code installed"
    info "Run 'claude' to authenticate before opening Ghostty."
  else
    error "Claude Code installation failed."
    info "Install manually: curl -fsSL https://claude.ai/install.sh | sh"
    info "Then run 'claude' to authenticate before re-running this script."
    exit 1
  fi
fi

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
cp "$SCRIPT_DIR/ghostty/claude-wrapper.sh" ~/.config/ghostty/claude-wrapper.sh
chmod +x ~/.config/ghostty/claude-wrapper.sh
success "Copied claude-wrapper.sh to ~/.config/ghostty/"

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
  read -rn1 -p "$(echo -e "${BLUE}Choose (1/2):${NC} ")" config_choice
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
      cp "$SCRIPT_DIR/ghostty/config" "$GHOSTTY_CONFIG"
      success "Replaced config with vibecode-editor defaults"
      ;;
    *)
      warn "Invalid choice, skipping config setup"
      ;;
  esac
else
  cp "$SCRIPT_DIR/ghostty/config" "$GHOSTTY_CONFIG"
  success "Created Ghostty config at $GHOSTTY_CONFIG"
fi

# ---------- Projects ----------
header "Setting up projects..."
PROJECTS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/vibecode-editor"
PROJECTS_FILE="$PROJECTS_DIR/projects"
mkdir -p "$PROJECTS_DIR"

echo ""
read -rn1 -p "$(echo -e "${BLUE}Add a project? (y/n):${NC} ")" add_project
echo ""

while [[ "$add_project" =~ ^[yY]$ ]]; do
  read -rp "$(echo -e "${BLUE}Project name:${NC} ")" proj_name
  read -rp "$(echo -e "${BLUE}Project path:${NC} ")" proj_path

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
  read -rn1 -p "$(echo -e "${BLUE}Add another? (y/n):${NC} ")" add_project
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
