# Ghost Tab

A **`Ghostty`** + **`tmux`** wrapper that launches a four-pane dev session with **`Claude Code`**, **`lazygit`**, **`broot`**, and a spare terminal. Automatically cleans up all processes when the window is closed — no zombie **`Claude Code`** processes.

![vibecode-editor screenshot](docs/screenshot.png)

---

## Quick Start

> [!NOTE]
> **One command to install everything:**

```sh
curl -fsSL https://raw.githubusercontent.com/JackUait/vibecode-editor/main/setup.sh | bash
```

That's it. The script installs all dependencies, sets up **`Ghostty`**, and walks you through adding your projects. Then open a new **`Ghostty`** window.

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
  · Add new project
  · Delete a project
  · Open once
──────────────────────────────────────
  ↑↓ navigate  ⏎ select
```

- **Arrow keys** or **mouse click** to navigate
- **Number keys** (1-9) to jump directly to a project
- **Enter** to select
- **Path autocomplete** when adding projects (with Tab completion)

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

## What the Setup Script Does

1. Installs **`Homebrew`** (if needed)
2. Installs **`tmux`**, **`lazygit`**, **`broot`**, and **`Claude Code`** via **`Homebrew`**
3. Installs **`Ghostty`** via **`Homebrew`** cask (if needed)
4. Sets up the **`Ghostty`** config (with merge/replace option if you have an existing one)
5. Walks you through adding your **project directories**
6. Sets up **Claude Code status line** showing git info and context usage (requires **`Node.js`**)

<details>
<summary><strong>Alternative: Clone and Run</strong></summary>

```sh
git clone https://github.com/JackUait/vibecode-editor.git
cd vibecode-editor
./setup.sh
```

</details>

<details>
<summary><strong>Alternative: Manual Setup</strong></summary>

If you prefer to set things up by hand:

1. Copy `ghostty/claude-wrapper.sh` to `~/.config/ghostty/` and make it executable
2. Add `command = ~/.config/ghostty/claude-wrapper.sh` to `~/.config/ghostty/config`
3. Add your projects to `~/.config/vibecode-editor/projects`, one per line in `name:path` format:

```
my-app:/path/to/my-app
another-project:/path/to/another-project
```

Lines starting with `#` are ignored. You can also add/delete projects directly from the interactive menu.

</details>

---

## Status Line

If **`Node.js`** is installed, the setup script configures a custom **Claude Code** status line:

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
