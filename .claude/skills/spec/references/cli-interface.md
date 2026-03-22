# CLI 接口定义

## cc-track setup

注册 hooks 到 `~/.claude/settings.json`。

```
cc-track setup          # 注册 hooks
cc-track setup --check  # 检查状态
cc-track setup --remove # 移除 hooks
```

GIVEN 用户运行 `cc-track setup`
WHEN `~/.claude/settings.json` 中没有 cc-track hooks
THEN 追加 9 个 hook 条目，保留已有 hooks

GIVEN hooks 已存在
THEN 输出 "hooks already registered" 并退出

## cc-track hook

被 Claude Code hooks 调用，读 stdin JSON 写 DB。用户不直接调用。

GIVEN Claude Code 触发 SessionStart
WHEN `cc-track hook` 收到 stdin JSON
THEN 解析 hook_event_name，INSERT sessions，project 从 `git rev-parse --show-toplevel` 派生

GIVEN Claude Code 触发 PreToolUse
THEN INSERT tool_calls 部分行（tool_name, tool_input, started_at）

GIVEN Claude Code 触发 PostToolUse
WHEN tool_use_id 对应的行存在
THEN UPDATE（tool_output, completed_at, duration_ms, succeeded=1）

GIVEN Claude Code 触发 PostToolUse
WHEN tool_use_id 对应的行不存在（竞态）
THEN INSERT 完整行

## cc-track summary

```
cc-track summary                    # 今天
cc-track summary --week             # 最近 7 天
cc-track summary --month            # 最近 30 天
cc-track summary --since 2026-03-01 # 自定义范围
cc-track summary --json             # JSON 输出
```

输出：sessions 数、prompts 数、tool calls 按类型分布（名称+百分比）、错误率、总时长。

## cc-track session

```
cc-track session list               # 最近 sessions（默认 10 条）
cc-track session list --limit 20
cc-track session show <id>          # 支持前缀匹配
```

`show` 输出：session 元数据 + 事件 timeline。

## cc-track waste

```
cc-track waste                      # 分析最近 sessions
cc-track waste --session <id>       # 指定 session
```

5 种检测模式：重复调用、过度读取、失败重试、Edit 反复、僵尸 session。

## cc-track roi

```
cc-track roi                        # 今天
cc-track roi --since 2026-03-01
cc-track roi --repo /path
```

输出：sessions 数、总时长、tool calls、lines added/removed、commits 数。

## cc-track export

```
cc-track export --format csv|json
cc-track export --since 2026-03-01
```
