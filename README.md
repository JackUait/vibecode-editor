# vibecode-editor

A Ghostty + tmux wrapper that launches a four-pane dev session with Claude Code, lazygit, broot, and a spare terminal. Automatically cleans up all processes when the window is closed — no zombie Claude processes.

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

- [Ghostty](https://ghostty.org)
- [tmux](https://github.com/tmux/tmux)
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code)
- [lazygit](https://github.com/jesseduffield/lazygit)
- [broot](https://dystroy.org/broot/)

Install on macOS:

```sh
brew install tmux lazygit broot
```

## Setup

1. Clone this repo:

```sh
git clone https://github.com/JackUait/vibecode-editor.git
```

2. Copy the wrapper script to your Ghostty config:

```sh
mkdir -p ~/.config/ghostty
cp vibecode-editor/ghostty/claude-wrapper.sh ~/.config/ghostty/claude-wrapper.sh
chmod +x ~/.config/ghostty/claude-wrapper.sh
```

3. Edit the `projects` array in `claude-wrapper.sh` to list your own project directories:

```bash
projects=(
    "my-app:/path/to/my-app"
    "another-project:/path/to/another-project"
)
```

4. Configure Ghostty to use the wrapper. Add to `~/.config/ghostty/config`:

```
command = /Users/YOUR_USERNAME/.config/ghostty/claude-wrapper.sh
```

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
