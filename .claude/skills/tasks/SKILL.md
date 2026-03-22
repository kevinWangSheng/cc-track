---
name: tasks
description: cc-track 实施任务清单，Pipeline 模式——按依赖顺序分阶段执行，每阶段有 gate 条件。开始开发或查看进度时使用。
---

# cc-track 实施 Pipeline

每个 Phase 有明确的 Gate 条件。**未通过 Gate 不得进入下一 Phase。**

## Phase 1: 基础骨架 + 数据采集

### Step 1.1 项目初始化
- [ ] `go mod init github.com/shenghuikevin/cc-track`
- [ ] 添加依赖：cobra, modernc.org/sqlite, lipgloss
- [ ] `main.go` + `cmd/root.go` + `cmd/version.go`

**验证**：`go build -o cc-track . && ./cc-track version` 输出版本号

### Step 1.2 配置和路径
- [ ] `internal/config/paths.go`：DataDir(), DBPath()
- [ ] 自动创建 `~/.cc-track/`

**验证**：单元测试 paths_test.go 通过

### Step 1.3 SQLite 存储层
- [ ] `internal/store/schema.go`：建 4 张表 + indexes
- [ ] `internal/store/db.go`：Open(), Migrate(), Close()
- [ ] `internal/store/sessions.go`
- [ ] `internal/store/toolcalls.go`
- [ ] `internal/store/prompts.go`

加载 schema 详情：读 [/spec references/data-model.md]

**验证**：`go test ./internal/store/...` 全部通过（:memory: DB）

### Step 1.4 Hook 事件处理
- [ ] `internal/hook/events.go`：所有事件结构体
- [ ] `internal/hook/handler.go`：ReadAndDispatch
- [ ] `internal/hook/testdata/*.json`：9 个 fixture 文件

加载事件结构体详情：读 [/plan references/hook-events.md]

**验证**：`go test ./internal/hook/...` 全部通过

### Step 1.5 Hook + Setup 命令
- [ ] `cmd/hook.go`
- [ ] `cmd/setup.go`（含 --check, --remove）

**验证**：`echo '<json>' | ./cc-track hook` 写入 DB

### Gate 1 → Phase 2

```bash
go test ./...                           # 全部通过
go vet ./...                            # 无警告
./cc-track setup --check                # hooks 已注册
# 在 Claude Code 中执行几条命令后：
sqlite3 ~/.cc-track/data.db "SELECT count(*) FROM sessions"    # > 0
sqlite3 ~/.cc-track/data.db "SELECT count(*) FROM tool_calls"  # > 0
```

全部通过才能进入 Phase 2。

---

## Phase 2: 查询 + 展示

### Step 2.1 输出格式化
- [ ] `internal/output/table.go`
- [ ] `internal/output/json.go`
- [ ] root.go 加全局 `--json` flag

### Step 2.2 Summary
- [ ] `internal/store/queries.go`：聚合查询
- [ ] `internal/analysis/summary.go`
- [ ] `cmd/summary.go`

### Step 2.3 Session
- [ ] `cmd/session.go`：list + show 子命令
- [ ] session ID 前缀匹配

### Step 2.4 Export
- [ ] `cmd/export.go`

### Gate 2 → Phase 3

```bash
go test ./...
./cc-track summary --week              # 输出格式正确
./cc-track session list                # 有数据
./cc-track session show <id>           # timeline 正确
./cc-track summary --json | jq .       # 合法 JSON
```

---

## Phase 3: 分析引擎

### Step 3.1 浪费检测
- [ ] `internal/analysis/waste.go`：5 种检测算法

加载检测规则详情：读 [/testing references/waste-checklist.md]

- [ ] `cmd/waste.go`

### Step 3.2 ROI
- [ ] `internal/analysis/roi.go`：git log/diff 集成
- [ ] `cmd/roi.go`

### Gate 3 → Phase 4

```bash
go test ./...
./cc-track waste                       # 输出检测结果
./cc-track roi                         # 输出 ROI 数据
```

---

## Phase 4: 分发

- [ ] Makefile
- [ ] goreleaser 配置
- [ ] GitHub Actions CI
- [ ] README.md
- [ ] tag v0.1.0

### Final Gate

```bash
make test && make lint
goreleaser check
go install github.com/shenghuikevin/cc-track@latest  # 成功
```
