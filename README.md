# cc-track

Local-first CLI tool for monitoring [Claude Code](https://docs.anthropic.com/en/docs/claude-code) usage. Built with Go + SQLite, zero external dependencies, no daemon.

## How It Works

cc-track registers itself as a Claude Code hook. Every time Claude Code starts a session, submits a prompt, or calls a tool, the event is piped to `cc-track hook` which writes it to a local SQLite database (`~/.cc-track/data.db`). You can then query your usage with simple CLI commands.

```
Claude Code Event → Hook (async) → cc-track hook → SQLite
```

## Install

```bash
go install github.com/shenghuikevin/cc-track@latest
```

Or build from source:

```bash
git clone https://github.com/kevinWangSheng/cc-track.git
cd cc-track
go build -o cc-track .
```

## Setup

Register hooks in Claude Code's `~/.claude/settings.json`:

```bash
cc-track setup          # register hooks (merges with existing ones)
cc-track setup --check  # verify registration
cc-track setup --remove # unregister hooks
```

This adds hooks for 9 event types: SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, PostToolUseFailure, Stop, StopFailure, SubagentStop, SessionEnd.

## Usage

### Summary

```bash
cc-track summary          # today's usage
cc-track summary --week   # last 7 days
cc-track summary --month  # last 30 days
cc-track summary --since 2025-03-01  # custom range
```

Shows: session count, prompts, tool calls, duration, error rate, token breakdown (input/output/cache), and tool usage distribution.

### Sessions

```bash
cc-track session list             # recent 10 sessions
cc-track session list --limit 20  # more sessions
cc-track session show <id>        # detailed session with timeline
```

Session detail includes full chronological timeline of prompts, tool calls, and stop events.

### Export

```bash
cc-track export                        # JSON export (all data)
cc-track export --format csv           # CSV export
cc-track export --since 2025-03-01     # filter by date
```

### JSON Output

All commands support `--json` for machine-readable output:

```bash
cc-track summary --json
cc-track session list --json
```

## What Gets Tracked

| Data | Source |
|------|--------|
| Sessions | start/end time, project, git branch, model, CWD |
| Prompts | user message text, timestamp |
| Tool Calls | tool name, input/output (truncated to 10KB), success/failure, duration |
| Tokens | input, output, cache-read, cache-creation (parsed from transcript) |
| Stop Events | stop reason, errors |

All data stays local in `~/.cc-track/data.db`. Nothing is sent anywhere.

## Stack

- [Go](https://go.dev/) 1.22+
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — pure-Go SQLite, no CGO
- [spf13/cobra](https://github.com/spf13/cobra) — CLI framework
- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling

## License

MIT
