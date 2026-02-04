# Homebrew Tap Package Design

## Goal

Distribute ghost-tab as a Homebrew package with semantic versioning.

## Decisions

- Package name: `ghost-tab`
- Rename GitHub repo from `vibecode-editor` to `ghost-tab`
- Starting version: `v1.0.0`
- All runtime dependencies declared as `depends_on` in the formula

## Repository Structure

Single repo: `ghost-tab` (no separate tap repo needed)

```
ghost-tab/
├── VERSION
├── README.md
├── HomebrewFormula/
│   └── ghost-tab.rb
├── bin/
│   └── ghost-tab
├── ghostty/
│   ├── claude-wrapper.sh
│   └── config
└── docs/
    ├── screenshot.png
    └── plans/
```

- `setup.sh` moves to `bin/ghost-tab`
- Embedded wrapper script extracted back into `ghostty/claude-wrapper.sh` as standalone file
- `VERSION` file at repo root contains `1.0.0`
- Formula lives in `HomebrewFormula/` inside the same repo

## Homebrew Formula

The formula:

- Downloads the tagged release tarball from `ghost-tab` repo
- Installs `bin/ghost-tab` to Homebrew's bin directory
- Installs `ghostty/` to `$(brew --prefix)/share/ghost-tab/ghostty/`
- Declares dependencies:
  - `depends_on "tmux"`
  - `depends_on "lazygit"`
  - `depends_on "broot"`
  - `depends_on "claude-code"`
  - `depends_on "node@22"`
- Note: Ghostty (`cask`) cannot be a formula dependency (Homebrew limitation). The `bin/ghost-tab` script checks for and installs Ghostty at runtime.

## Install Experience

```bash
# Homebrew
brew tap JackUait/ghost-tab https://github.com/JackUait/ghost-tab
brew install ghost-tab
ghost-tab

# Curl pipe (still supported)
curl -fsSL https://raw.githubusercontent.com/JackUait/ghost-tab/main/bin/ghost-tab | bash
```

## Script Changes

`bin/ghost-tab` detects its install context at startup:

```bash
if [[ "$(dirname "$0")" == "$(brew --prefix)/bin" ]]; then
    SHARE_DIR="$(brew --prefix)/share/ghost-tab"
else
    SHARE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
fi
```

All references to `ghostty/config` and `ghostty/claude-wrapper.sh` use `$SHARE_DIR/ghostty/...`.

## Release Workflow

1. Update `VERSION` file
2. Tag the commit: `git tag v1.0.0 && git push --tags`
3. Create GitHub Release from the tag
4. Update formula in `HomebrewFormula/ghost-tab.rb` with new tarball URL and SHA256 hash

No CI/CD automation to start. Automate later if needed.

## What Stays the Same

- All existing functionality (menu, tmux layout, process cleanup, status line)
- `ghostty/config` file
- `docs/` directory
