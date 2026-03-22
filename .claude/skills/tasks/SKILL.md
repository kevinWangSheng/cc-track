---
name: tasks
description: cc-track 实施任务清单，Pipeline 模式——按依赖顺序分阶段执行，每阶段有 gate 条件。开始开发或查看进度时使用。
---

# cc-track 实施 Pipeline

每个 Phase 有明确的 Gate 条件。**未通过 Gate 不得进入下一 Phase。**

## Phase 1: 基础骨架 + 数据采集 ✅

### Step 1.1 项目初始化
- [x] `go mod init github.com/shenghuikevin/cc-track`
- [x] 添加依赖：cobra, modernc.org/sqlite, lipgloss
- [x] `main.go` + `cmd/root.go` + `cmd/version.go`

### Step 1.2 配置和路径
- [x] `internal/config/paths.go`：DataDir(), DBPath()
- [x] 自动创建 `~/.cc-track/`

### Step 1.3 SQLite 存储层
- [x] `internal/store/schema.go`：建 4 张表 + indexes
- [x] `internal/store/db.go`：Open(), Migrate(), Close()
- [x] `internal/store/sessions.go`
- [x] `internal/store/toolcalls.go`
- [x] `internal/store/prompts.go`

### Step 1.4 Hook 事件处理
- [x] `internal/hook/events.go`：所有事件结构体
- [x] `internal/hook/handler.go`：ReadAndDispatch
- [x] `internal/hook/testdata/*.json`：9 个 fixture 文件

### Step 1.5 Hook + Setup 命令
- [x] `cmd/hook.go`
- [x] `cmd/setup.go`（含 --check, --remove）

### Gate 1 ✅

---

## Phase 2: 查询 + 展示 ✅

### Step 2.1 输出格式化
- [x] `internal/output/table.go`
- [x] `internal/output/json.go`
- [x] root.go 加全局 `--json` flag

### Step 2.2 Summary
- [x] `internal/store/queries.go`：聚合查询
- [x] `cmd/summary.go`

### Step 2.3 Session
- [x] `cmd/session.go`：list + show 子命令
- [x] session ID 前缀匹配

### Step 2.4 Export
- [x] `cmd/export.go`

### Gate 2 ✅

---

## Phase 3: 分析引擎 ✅

### Step 3.1 浪费检测
- [x] `internal/analysis/waste.go`：5 种检测算法
- [x] `cmd/waste.go`

### Step 3.2 ROI
- [x] `internal/analysis/roi.go`：git log/diff 集成
- [x] `cmd/roi.go`

### Gate 3 ✅

---

## Phase 4: 分发 ✅

- [x] Makefile
- [x] goreleaser 配置
- [x] GitHub Actions CI
- [x] README.md（英文 + 中文）
- [x] tag v0.1.0

### Final Gate ✅

---

## Phase 5: 费用估算

### Step 5.1 模型定价表
- [ ] `internal/analysis/pricing.go`：各模型的 token 单价（input/output/cache）
- [ ] 支持 Claude Opus、Sonnet、Haiku 系列
- [ ] 单元测试

### Step 5.2 费用计算
- [ ] `internal/analysis/cost.go`：基于 session token 数据 × 模型单价计算费用
- [ ] 按 session / 按时间段汇总
- [ ] 单元测试

### Step 5.3 集成到 summary 和 session
- [ ] `cc-track summary` 增加费用行（Estimated Cost: $X.XX）
- [ ] `cc-track session list` 增加费用列
- [ ] `cc-track session show` 增加费用明细

### Gate 5

```bash
go test ./...
cc-track summary --week              # 显示费用估算
cc-track session list                # 有费用列
cc-track summary --json | jq .cost   # JSON 中包含 cost 字段
```

---

## Phase 6: 趋势对比

### Step 6.1 趋势查询
- [ ] `internal/store/trend_queries.go`：按天/周聚合数据
- [ ] 返回时间序列：日期、sessions、tokens、cost、waste 数

### Step 6.2 trend 命令
- [ ] `cmd/trend.go`
- [ ] `cc-track trend`：默认最近 7 天，按天展示
- [ ] `cc-track trend --month`：最近 30 天
- [ ] 终端 sparkline 或 ASCII 柱状图可视化

### Step 6.3 waste 趋势
- [ ] 在 trend 中展示 waste 率变化
- [ ] 对比本周 vs 上周

### Gate 6

```bash
go test ./...
cc-track trend                       # 每日趋势输出
cc-track trend --month               # 30 天趋势
cc-track trend --json | jq .         # 合法 JSON
```

---

## Phase 7: 智能建议

### Step 7.1 建议引擎
- [ ] `internal/analysis/suggest.go`：基于 waste 检测结果生成改进建议
- [ ] 每种 waste 类型对应具体可操作的建议文本
- [ ] 建议附带严重程度（info / warning / critical）

### Step 7.2 集成到 waste 命令
- [ ] `cc-track waste` 输出中每条 finding 后附带建议
- [ ] `cc-track waste --summary`：只输出建议汇总，不展示逐条 finding

### Gate 7

```bash
go test ./...
cc-track waste                       # finding 后有建议
cc-track waste --summary             # 汇总建议
```

---

## Phase 8: 生态完善

### Step 8.1 Homebrew
- [ ] `Formula/cc-track.rb` 或 homebrew tap 仓库
- [ ] `brew install kevinWangSheng/tap/cc-track` 可用

### Step 8.2 报告生成
- [ ] `cc-track report --format md`：生成 Markdown 周报/月报
- [ ] 包含：费用摘要、趋势图、waste 分析、ROI、top sessions
- [ ] `cc-track report --format html`：HTML 版本

### Step 8.3 实时监控（可选）
- [ ] `cc-track watch`：实时监听 DB 变更，终端 dashboard 展示当前 session
- [ ] 显示实时 token 消耗、工具调用流

### Gate 8

```bash
brew install kevinWangSheng/tap/cc-track  # Homebrew 安装成功
cc-track report --format md               # 生成 Markdown 报告
```
