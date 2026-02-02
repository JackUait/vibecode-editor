# vibecode-editor

A Ghostty + tmux wrapper that launches a four-pane dev session with Claude Code, lazygit, broot, and a spare terminal. Automatically cleans up all processes when the window is closed — no zombie Claude processes.

![vibecode-editor screenshot](docs/screenshot.png)

## Layout

```
┌──────────────┬──────────────┐
│   lazygit    │  Claude Code │
├──────────────┤              │
│    broot     │              │
├──────────────┤              │
│   terminal   │              │
└──────────────┴──────────────┘
```

## Prerequisites

- macOS
- [Ghostty](https://ghostty.org)
Everything else (tmux, lazygit, broot, Claude Code) is installed automatically by the setup script.

## Setup

```sh
curl -fsSL https://raw.githubusercontent.com/JackUait/vibecode-editor/main/setup.sh | bash
```

Or clone and run locally:

```sh
git clone https://github.com/JackUait/vibecode-editor.git
cd vibecode-editor
./setup.sh
```

The setup script will:

1. Install Homebrew (if needed)
2. Install tmux, lazygit, broot, and Claude Code
4. Set up the Ghostty config (with merge/replace option if you have an existing one)
5. Walk you through adding your project directories

### Manual setup

If you prefer to set things up manually:

1. Copy `ghostty/claude-wrapper.sh` to `~/.config/ghostty/` and make it executable
2. Add `command = ~/.config/ghostty/claude-wrapper.sh` to `~/.config/ghostty/config`
3. Add your projects to `~/.config/vibecode-editor/projects`, one per line in `name:path` format:

```
my-app:/path/to/my-app
another-project:/path/to/another-project
```

Lines starting with `#` are ignored. If the file doesn't exist or is empty, the wrapper opens in the current directory.

## Usage

Open a new Ghostty window. You'll see a project picker:

```
Select project:
  1) my-app
  2) another-project
  0) current directory
>
```

Pick a project and the four-pane tmux session launches with Claude Code auto-focused.

To open a specific directory directly:

```sh
~/.config/ghostty/claude-wrapper.sh /path/to/project
```

## Process cleanup

When you close the Ghostty window, the wrapper automatically:

1. Kills all child processes in every tmux pane (including Claude Code and any subprocesses it spawned)
2. Destroys the tmux session

This prevents zombie Claude Code processes from accumulating.
