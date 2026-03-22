# cc-track

Local-first CLI tool for analyzing [Claude Code](https://docs.anthropic.com/en/docs/claude-code) usage — not just how much you spent, but whether it was worth it.

## Why cc-track?

Most monitoring tools tell you **how much** you spent (tokens, API calls, dollars). cc-track tells you **how well** you spent it.

| | Anthropic Console / SaaS Observability | cc-track |
|---|---|---|
| Data location | Cloud / third-party | **100% local** (`~/.cc-track/data.db`) |
| Granularity | API call level | Session → Prompt → Tool Call full chain |
| Perspective | "How much did I spend?" | **"Was it worth it?"** (waste + ROI) |
| Setup | API proxy / SDK integration | One command, zero intrusion |
| Dependencies | Server, network | **None** — no daemon, no network |

**Key differentiators:**

- **Waste Detection** — identifies duplicate tool calls, excessive file reads, failed retries, edit reverts, and zombie sessions. Tells you where tokens are being wasted.
- **ROI Analysis** — correlates Claude Code sessions with `git log` output (commits, lines added/removed) to answer "what did this investment produce?"
- **Privacy First** — all data stays on your machine. Nothing is sent anywhere, ever.

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

### Waste Detection

```bash
cc-track waste                    # analyze recent 10 sessions
cc-track waste --session <id>     # analyze a specific session
```

Detects 5 patterns:
- **Duplicate Calls** — same tool + similar input, 3+ times within 60s
- **Excessive Reads** — same file read 5+ times in one session
- **Failed Retries** — same tool fails consecutively 3+ times
- **Edit Reverts** — A→B→A pattern on the same file
- **Zombie Sessions** — >30 min duration but almost no activity

### ROI Analysis

```bash
cc-track roi                      # today's ROI
cc-track roi --since 2025-03-01   # custom date range
cc-track roi --repo /path/to/repo # override repo path
```

Correlates session data with git output: sessions, duration, tool calls, commits, lines added/removed. Automatically deduplicates when multiple sessions point to the same repo.

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
