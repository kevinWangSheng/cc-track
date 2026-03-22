# cc-track

Local-first CLI tool for monitoring Claude Code usage. Go + SQLite, zero external dependencies.

## Commands

```bash
go build -o cc-track .          # build
go test ./...                   # test all
go vet ./...                    # lint
golangci-lint run               # full lint (if installed)
```

## Stack

- Go 1.22+, SQLite via `modernc.org/sqlite`, CLI via `github.com/spf13/cobra`
- 终端样式 via `github.com/charmbracelet/lipgloss`

## Architecture

- 无 daemon：hook spawn 短生命周期进程直接写 SQLite
- 数据：`~/.cc-track/data.db`（WAL mode + busy_timeout=5000）
- 采集：Claude Code hooks（command 类型，async: true）
- 入口：`cc-track hook`，按 `hook_event_name` 分发

## Boundaries

### Always
- error: `fmt.Errorf("pkg: %w", err)`
- JSON 字段: snake_case
- tool_input/tool_output 超 10KB 截断
- 新功能必须带测试

### Ask First
- 修改 SQLite schema
- 修改 settings.json 合并逻辑
- 添加外部依赖

### Never
- CGO
- `internal/` 层打日志（只返回 error）
- `log.Fatal` / `os.Exit`（除 `main.go`）
- mock SQLite（用 `:memory:`）

## Gotchas

- SQLite 驱动名是 `"sqlite"` 不是 `"sqlite3"`：`sql.Open("sqlite", path)`
- `modernc.org/sqlite` 导入用 `_ "modernc.org/sqlite"`，不是 `_ "github.com/mattn/go-sqlite3"`
- hook 是 async 的，PreToolUse 和 PostToolUse 可能乱序到达，必须用 tool_use_id 做 upsert
- `~/.claude/settings.json` 里可能已有其他 hooks（如 notifier），setup 必须 merge 不能覆盖
- 时间戳全部用 unix milliseconds（int64），不是 seconds

## Skills

详细规范按需加载：`/spec` `/plan` `/tasks` `/conventions` `/testing`
