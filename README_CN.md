# cc-track

本地优先的 [Claude Code](https://docs.anthropic.com/en/docs/claude-code) 用量分析工具 —— 不只告诉你花了多少，还告诉你花得值不值。

## 为什么选 cc-track？

大多数监控工具只回答**"花了多少"**（token 数、API 调用次数、费用）。cc-track 回答的是**"花得值不值"**。

| | Anthropic Console / SaaS 监控 | cc-track |
|---|---|---|
| 数据位置 | 云端 / 第三方 | **100% 本地**（`~/.cc-track/data.db`） |
| 粒度 | API 调用级 | Session → Prompt → Tool Call 全链路 |
| 视角 | "花了多少钱？" | **"花得值不值？"**（浪费检测 + ROI） |
| 接入方式 | API 代理 / SDK 集成 | 一条命令注册，零侵入 |
| 依赖 | 服务端、网络 | **无** —— 无 daemon、无网络、无外部依赖 |

**核心差异：**

- **浪费检测** —— 识别重复工具调用、过度文件读取、失败重试、Edit 反复修改、僵尸 session，告诉你 token 浪费在了哪里
- **ROI 分析** —— 将 Claude Code session 与 `git log` 产出（commits 数、代码行增减）关联，回答"这些投入产生了什么"
- **隐私优先** —— 所有数据留在本机，不发送到任何地方

## 工作原理

cc-track 将自己注册为 Claude Code hook。每当 Claude Code 启动 session、提交 prompt 或调用工具时，事件通过 stdin 传给 `cc-track hook`，写入本地 SQLite 数据库。之后你可以用 CLI 命令查询用量。

```
Claude Code 事件 → Hook（异步） → cc-track hook → SQLite
```

## 安装

```bash
go install github.com/shenghuikevin/cc-track@latest
```

或从源码构建：

```bash
git clone https://github.com/kevinWangSheng/cc-track.git
cd cc-track
go build -o cc-track .
```

## 配置

将 hooks 注册到 Claude Code 的 `~/.claude/settings.json`：

```bash
cc-track setup          # 注册 hooks（与已有 hooks 合并，不会覆盖）
cc-track setup --check  # 检查注册状态
cc-track setup --remove # 移除 hooks
```

自动注册 9 种事件：SessionStart、UserPromptSubmit、PreToolUse、PostToolUse、PostToolUseFailure、Stop、StopFailure、SubagentStop、SessionEnd。

## 使用

### 用量概览

```bash
cc-track summary                      # 今天的用量
cc-track summary --week               # 最近 7 天
cc-track summary --month              # 最近 30 天
cc-track summary --since 2025-03-01   # 自定义起始日期
```

展示：session 数、prompt 数、工具调用次数、总时长、错误率、token 明细（input/output/cache）、工具使用分布。

### Session 管理

```bash
cc-track session list                 # 最近 10 个 session
cc-track session list --limit 20      # 显示更多
cc-track session show <id>            # 详细信息 + 时间线
```

Session 详情包含完整的时间线：prompt、工具调用、停止事件按时间排列。

### 浪费检测

```bash
cc-track waste                        # 分析最近 10 个 session
cc-track waste --session <id>         # 分析指定 session
```

检测 5 种浪费模式：

- **重复调用** —— 相同工具 + 相似输入，60 秒内出现 3 次以上
- **过度读取** —— 同一文件在一个 session 内被 Read 5 次以上
- **失败重试** —— 同一工具连续失败 3 次以上
- **Edit 反复** —— 同一文件出现 A→B→A 的回退模式
- **僵尸 Session** —— 持续 30 分钟以上但几乎无操作（prompts < 3 且 tool calls < 5）

### ROI 分析

```bash
cc-track roi                          # 今天的 ROI
cc-track roi --since 2025-03-01       # 自定义日期范围
cc-track roi --repo /path/to/repo     # 指定仓库路径
```

将 session 数据与 git 产出关联：session 数、总时长、工具调用次数、commit 数、代码行增减。同一仓库自动去重，不会重复计算。

### 数据导出

```bash
cc-track export                       # JSON 导出（全部数据）
cc-track export --format csv          # CSV 导出
cc-track export --since 2025-03-01    # 按日期过滤
```

### JSON 输出

所有命令支持 `--json` 获取机器可读输出：

```bash
cc-track summary --json
cc-track session list --json
```

## 采集数据

| 数据 | 内容 |
|------|------|
| Sessions | 起止时间、项目名、git 分支、模型、工作目录 |
| Prompts | 用户消息文本、时间戳 |
| Tool Calls | 工具名、输入/输出（超 10KB 截断）、成功/失败、耗时 |
| Tokens | input、output、cache-read、cache-creation（从 transcript 解析） |
| Stop Events | 停止原因、错误信息 |

所有数据存储在本地 `~/.cc-track/data.db`，不会发送到任何地方。

## 技术栈

- [Go](https://go.dev/) 1.22+
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — 纯 Go SQLite，无需 CGO
- [spf13/cobra](https://github.com/spf13/cobra) — CLI 框架
- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) — 终端样式

## 许可证

MIT
