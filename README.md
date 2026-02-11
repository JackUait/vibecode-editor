# Ghost Tab

A **`Ghostty`** + **`tmux`** wrapper that launches a four-pane dev session with **`Claude Code`**, **`lazygit`**, **`broot`**, and a spare terminal. Automatically cleans up all processes when the window is closed — no zombie **`Claude Code`** processes.

![ghost-tab screenshot](docs/screenshot.png)

---

## Quick Start

> [!NOTE]
> **Homebrew (recommended):**

```sh
brew install JackUait/ghost-tab/ghost-tab
ghost-tab
```

That's it. Homebrew installs all dependencies. Then run `ghost-tab` to set up **`Ghostty`** and add your projects.

> [!IMPORTANT]
> **Only requirement:** **`macOS`**. Everything (**`Homebrew`**, **`Ghostty`**, **`tmux`**, **`lazygit`**, **`broot`**, **`Claude Code`**) is installed automatically.

---

## Usage

**Step 1.** Open a new **`Ghostty`** window (`Cmd+N`)

**Step 2.** Use the interactive project selector:

```
⬡  Ghost Tab
──────────────────────────────────────

 1❯ my-app
    ~/Projects/my-app
  2 another-project
    ~/Projects/another-project
──────────────────────────────────────
  A Add new project
  D Delete a project
  O Open once
  P Plain terminal
──────────────────────────────────────
  ↑↓ navigate  ⏎ select
```

- **Arrow keys** or **mouse click** to navigate
- **Number keys** (1-9) to jump directly to a project
- **Letter keys** — **A** add, **D** delete, **O** open once, **P** plain terminal
- **Enter** to select
- **Path autocomplete** when adding projects (with Tab completion)
- **Plain terminal** opens a bare shell with no tmux overhead

**Step 3.** The four-pane **`tmux`** session launches automatically with **`Claude Code`** already focused — start typing your prompt right away.

> [!TIP]
> You can also open a specific project directly from the terminal:
> ```sh
> ~/.config/ghostty/claude-wrapper.sh /path/to/project
> ```

---

## Hotkeys

| Shortcut | Action |
|---|---|
| `Cmd+T` | New tab |
| `Cmd+Shift+Left` | Previous tab |
| `Cmd+Shift+Right` | Next tab |
| `Left Option` | Acts as `Alt` instead of typing special characters |

---

## What `ghost-tab` Does

1. Installs **`Homebrew`** (if needed)
2. Installs **`tmux`**, **`lazygit`**, and **`broot`** via **`Homebrew`**
3. Installs **`Claude Code`** via native installer (auto-updates)
4. Installs **`Ghostty`** via **`Homebrew`** cask (if needed)
5. Sets up the **`Ghostty`** config (with merge/replace option if you have an existing one)
6. Walks you through adding your **project directories**
7. Installs **`Node.js`** LTS (if needed) and sets up **Claude Code status line** showing git info and context usage

<details>
<summary><strong>Alternative: Clone and Run</strong></summary>

```sh
git clone https://github.com/JackUait/ghost-tab.git
cd ghost-tab
./bin/ghost-tab
```

</details>

<details>
<summary><strong>Alternative: One-liner</strong></summary>

> [!CAUTION]
> The one-liner requires cloning the repo first. It cannot run standalone via curl pipe.

```sh
git clone https://github.com/JackUait/ghost-tab.git && cd ghost-tab && ./bin/ghost-tab
```

</details>

<details>
<summary><strong>Alternative: Manual Setup</strong></summary>

If you prefer to set things up by hand:

1. Copy `ghostty/claude-wrapper.sh` to `~/.config/ghostty/` and make it executable
2. Add `command = ~/.config/ghostty/claude-wrapper.sh` to `~/.config/ghostty/config`
3. Add your projects to `~/.config/ghost-tab/projects`, one per line in `name:path` format:

```
my-app:/path/to/my-app
another-project:/path/to/another-project
```

Lines starting with `#` are ignored. You can also add/delete projects directly from the interactive menu.

</details>

---

## Status Line

The `ghost-tab` command configures a custom **Claude Code** status line based on [Matt Pocock's guide](https://www.aihero.dev/creating-the-perfect-claude-code-status-line):

```
my-project | main | S: 0 | U: 2 | A: 1 | 23.5%
```

- **Repository name** — current project
- **Branch** — current git branch
- **S** — staged files count
- **U** — unstaged files count
- **A** — untracked (added) files count
- **Context %** — how much of Claude's context window is used

> [!TIP]
> Monitor context usage to know when to start a new conversation. Lower is better.

---

## Process Cleanup

> [!CAUTION]
> When you close the **`Ghostty`** window, **all processes are force-terminated** — make sure your work is saved.

The wrapper automatically:

1. **Recursively kills** the full process tree of every **`tmux`** pane (including deeply nested subprocesses spawned by **`Claude Code`**, **`lazygit`**, etc.)
2. **Force-kills** (`SIGKILL`) any processes that ignored the initial `SIGTERM` after a brief grace period
3. **Destroys** the **`tmux`** session
4. **Self-destructs** the session via `destroy-unattached` if the **`tmux`** client disconnects without triggering cleanup

This prevents zombie **`Claude Code`** processes from accumulating.

---

## Architecture

Ghost Tab uses a **hybrid architecture**:

**Layer 1: Go TUI Binary (`ghost-tab-tui`)**
- Interactive terminal UI components built with Bubbletea
- Project selector, AI tool selector, settings menu, input forms
- Outputs structured JSON for bash consumption
- Binary: `~/.local/bin/ghost-tab-tui`

**Layer 2: Bash Orchestration (`ghost-tab`)**
- Entry point and session orchestration
- Process management, config file operations
- Calls ghost-tab-tui for interactive parts
- Parses JSON responses with jq
- Script: `~/.local/bin/ghost-tab`

**Dependencies:**
- Go 1.21+ (for building)
- jq (for JSON parsing)
- tmux (session management)
- Ghostty (terminal emulator)

**Communication:**
```bash
# Bash calls Go with subcommand
result=$(ghost-tab-tui select-project --projects-file ~/.config/ghost-tab/projects)

# Go returns JSON
{"name": "my-project", "path": "/home/user/code/my-project", "selected": true}

# Bash parses with jq
project_name=$(echo "$result" | jq -r '.name')
```
