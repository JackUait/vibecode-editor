# Homebrew Tap Package Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Convert the ghost-tab project from a curl-pipe installer into a versioned Homebrew package.

**Architecture:** Restructure the repo so `setup.sh` becomes `bin/ghost-tab` (the entry point), extract the embedded wrapper script back into `ghostty/claude-wrapper.sh`, add a Homebrew formula in `HomebrewFormula/`, and add a `VERSION` file. The script detects whether it's running from a Homebrew install or a local clone and resolves supporting files accordingly.

**Tech Stack:** Bash, Homebrew (Ruby formula), Git tags, GitHub Releases

---

### Task 1: Create VERSION file

**Files:**
- Create: `VERSION`

**Step 1: Create the file**

```
1.0.0
```

**Step 2: Commit**

```bash
git add VERSION
git commit -m "Add VERSION file (1.0.0)"
```

---

### Task 2: Move setup.sh to bin/ghost-tab and split into setup vs runtime

The current `setup.sh` does two things: (1) installs dependencies and configures the system, and (2) embeds the runtime wrapper script. We need to separate these concerns.

After this task, the repo has:
- `bin/ghost-tab` — the setup/installer script (what `setup.sh` is today, minus the embedded wrapper)
- `ghostty/claude-wrapper.sh` — the runtime wrapper (already exists as a standalone file, but out of sync with the embedded version in `setup.sh`)

**Files:**
- Move: `setup.sh` → `bin/ghost-tab`
- Modify: `bin/ghost-tab` — remove the embedded `WRAPPER_SCRIPT` variable (lines 29-680), replace with logic that copies `ghostty/claude-wrapper.sh` from the correct location
- Modify: `ghostty/claude-wrapper.sh` — sync with the embedded version from `setup.sh` (the embedded version at lines 29-680 is newer and has the full interactive menu)

**Step 1: Create bin/ directory and move setup.sh**

```bash
mkdir -p bin
git mv setup.sh bin/ghost-tab
```

**Step 2: Sync ghostty/claude-wrapper.sh with the embedded version**

Replace the contents of `ghostty/claude-wrapper.sh` with the embedded `WRAPPER_SCRIPT` from the old `setup.sh` (lines 29-680). This is the version with the full interactive menu, mouse support, path autocomplete, etc.

The embedded script is stored as a single-quoted bash string with escaped single quotes (`'\''`). When extracting it:
- Remove the `WRAPPER_SCRIPT='` prefix and trailing `'` suffix
- Replace all `'\''` with literal `'`

The result should be a standalone executable bash script starting with `#!/bin/bash`.

**Step 3: Add install-context detection to bin/ghost-tab**

Add this near the top of `bin/ghost-tab` (after the color setup, before the embedded files section):

```bash
# Determine where supporting files live
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [[ "$SCRIPT_DIR" == "$(brew --prefix 2>/dev/null)/bin" ]]; then
    SHARE_DIR="$(brew --prefix)/share/ghost-tab"
else
    SHARE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
fi
```

**Step 4: Replace the embedded WRAPPER_SCRIPT variable**

Remove the entire `WRAPPER_SCRIPT='...'` block (lines 29-680 in the original). Replace the wrapper installation section (around line 748-753) that writes `$WRAPPER_SCRIPT` to disk with a file copy:

Replace:
```bash
# ---------- Wrapper script ----------
header "Setting up wrapper script..."
mkdir -p ~/.config/ghostty
echo "$WRAPPER_SCRIPT" > ~/.config/ghostty/claude-wrapper.sh
chmod +x ~/.config/ghostty/claude-wrapper.sh
success "Created claude-wrapper.sh in ~/.config/ghostty/"
```

With:
```bash
# ---------- Wrapper script ----------
header "Setting up wrapper script..."
mkdir -p ~/.config/ghostty
cp "$SHARE_DIR/ghostty/claude-wrapper.sh" ~/.config/ghostty/claude-wrapper.sh
chmod +x ~/.config/ghostty/claude-wrapper.sh
success "Created claude-wrapper.sh in ~/.config/ghostty/"
```

**Step 5: Similarly update the Ghostty default config reference**

The `GHOSTTY_DEFAULT_CONFIG` variable (lines 682-686) references a hardcoded config. Instead, read from the shipped config file. Replace:

```bash
GHOSTTY_DEFAULT_CONFIG='keybind = cmd+shift+left=previous_tab
keybind = cmd+shift+right=next_tab
keybind = cmd+t=new_tab
macos-option-as-alt = left
command = ~/.config/ghostty/claude-wrapper.sh'
```

With:
```bash
GHOSTTY_DEFAULT_CONFIG="$(cat "$SHARE_DIR/ghostty/config")"
```

**Step 6: Verify the script runs from a local clone**

```bash
chmod +x bin/ghost-tab
# From the repo root, this should resolve SHARE_DIR to the repo root
bash -c 'echo "$(cd "$(dirname "bin/ghost-tab")/.." && pwd)"'
```

Expected: prints the repo root path.

**Step 7: Commit**

```bash
git add bin/ghost-tab ghostty/claude-wrapper.sh
git commit -m "Restructure: move setup.sh to bin/ghost-tab, extract embedded wrapper"
```

---

### Task 3: Create the Homebrew formula

**Files:**
- Create: `HomebrewFormula/ghost-tab.rb`

**Step 1: Create HomebrewFormula directory and formula**

```ruby
class GhostTab < Formula
  desc "Ghostty + tmux wrapper for AI-assisted development with Claude Code"
  homepage "https://github.com/JackUait/ghost-tab"
  url "https://github.com/JackUait/ghost-tab/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"

  depends_on "tmux"
  depends_on "lazygit"
  depends_on "broot"
  depends_on "claude-code"
  depends_on "node@22"
  depends_on :macos

  def install
    bin.install "bin/ghost-tab"
    (share/"ghost-tab").install "ghostty"
  end

  def caveats
    <<~EOS
      Ghostty terminal is required but not installed automatically.
      Install it with:
        brew install --cask ghostty

      Then run `ghost-tab` to set up your environment.
    EOS
  end

  test do
    assert_match "macOS", shell_output("#{bin}/ghost-tab --help 2>&1", 1)
  end
end
```

Note: The `sha256` will be a placeholder until the first GitHub release is created. The `license` field assumes MIT — this should be confirmed or a LICENSE file added.

**Step 2: Commit**

```bash
git add HomebrewFormula/ghost-tab.rb
git commit -m "Add Homebrew formula"
```

---

### Task 4: Update README.md

**Files:**
- Modify: `README.md`

**Step 1: Update install instructions**

Replace the Quick Start section with both Homebrew and curl-pipe options. Update all references from `vibecode-editor` to `ghost-tab`. Update the wrapper script path reference.

The new Quick Start should show:

```markdown
## Quick Start

### Homebrew (recommended)

```sh
brew tap JackUait/ghost-tab https://github.com/JackUait/ghost-tab
brew install ghost-tab
ghost-tab
```

### One-liner

```sh
curl -fsSL https://raw.githubusercontent.com/JackUait/ghost-tab/main/bin/ghost-tab | bash
```
```

Update the "Alternative: Clone and Run" section:

```markdown
```sh
git clone https://github.com/JackUait/ghost-tab.git
cd ghost-tab
./bin/ghost-tab
```
```

Update the manual setup wrapper path from `ghostty/claude-wrapper.sh` to reference the correct location.

Update all remaining `vibecode-editor` references to `ghost-tab`.

**Step 2: Commit**

```bash
git add README.md
git commit -m "Update README with Homebrew install instructions and new repo name"
```

---

### Task 5: Update vibecode-editor references in scripts

**Files:**
- Modify: `bin/ghost-tab` — update config directory references
- Modify: `ghostty/claude-wrapper.sh` — update `PROJECTS_FILE` path

The projects file is currently stored at `~/.config/vibecode-editor/projects`. We should update this to `~/.config/ghost-tab/projects` to match the new name. Both scripts reference this path.

**Step 1: Update bin/ghost-tab**

Replace all `vibecode-editor` references with `ghost-tab`:
- `PROJECTS_DIR` path (line ~797)
- Any other references

**Step 2: Update ghostty/claude-wrapper.sh**

Replace the `PROJECTS_FILE` path:
```bash
PROJECTS_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/projects"
```

**Step 3: Add backwards compatibility migration**

In `bin/ghost-tab`, before the projects setup section, add a migration that moves the old config if it exists:

```bash
# Migrate from old config location
OLD_PROJECTS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/vibecode-editor"
NEW_PROJECTS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab"
if [ -d "$OLD_PROJECTS_DIR" ] && [ ! -d "$NEW_PROJECTS_DIR" ]; then
  mv "$OLD_PROJECTS_DIR" "$NEW_PROJECTS_DIR"
  info "Migrated config from vibecode-editor to ghost-tab"
fi
```

**Step 4: Commit**

```bash
git add bin/ghost-tab ghostty/claude-wrapper.sh
git commit -m "Rename vibecode-editor references to ghost-tab"
```

---

### Task 6: Verify everything works end-to-end

**Step 1: Test local clone install**

```bash
cd /path/to/ghost-tab
./bin/ghost-tab
```

Verify: the setup flow runs, finds dependencies, copies wrapper script correctly from `ghostty/claude-wrapper.sh`.

**Step 2: Test the formula structure**

```bash
brew audit --formula HomebrewFormula/ghost-tab.rb
```

This will fail on the sha256 placeholder but should pass structural checks.

**Step 3: Commit any fixes**

If anything needed fixing, commit the changes.

---

### Task 7: Tag v1.0.0 and prepare for release

**Step 1: Tag the release**

```bash
git tag v1.0.0
```

Note: Don't push yet — the repo needs to be renamed on GitHub first.

**Step 2: Document release steps**

After the GitHub repo is renamed from `vibecode-editor` to `ghost-tab`:
1. `git remote set-url origin https://github.com/JackUait/ghost-tab.git`
2. `git push origin main --tags`
3. Create GitHub Release from v1.0.0 tag
4. Download the tarball, compute sha256: `shasum -a 256 v1.0.0.tar.gz`
5. Update `HomebrewFormula/ghost-tab.rb` with the real sha256
6. Commit and push the sha256 update
