---
name: spec
description: cc-track 产品需求规格。开发新功能、理解需求、或需要确认验收标准时使用。采用 Inversion 模式——新增需求时先采访用户再行动。
---

# cc-track 需求规格

## Inversion Gate

当用户要求新增功能或修改行为时，在写代码之前必须完成以下采访：

1. **目标确认**：这个功能解决什么问题？
2. **接口确认**：CLI 命令长什么样？有哪些 flags？
3. **数据确认**：需要新增/修改 DB 表吗？数据从哪来？
4. **边界确认**：有什么不应该做的？
5. **验收确认**：怎么验证这个功能是正确的？

收集完答案后，先更新本 spec，再开始实现。

## 产品定义

面向个人开发者的本地 CLI 工具，通过 Claude Code hooks 采集 session 数据，存入本地 SQLite。核心问题：**钱花在哪了？**

## CLI 接口定义

详细的命令接口、参数、输出格式见 [references/cli-interface.md](references/cli-interface.md)。

## 数据模型

SQLite schema 定义见 [references/data-model.md](references/data-model.md)。

## Hook 事件映射

| Hook Event | DB 操作 |
|------------|---------|
| SessionStart | INSERT sessions |
| UserPromptSubmit | INSERT prompts, sessions.total_prompts++ |
| PreToolUse | INSERT tool_calls（部分） |
| PostToolUse | UPDATE tool_calls（补全），sessions.total_tool_calls++ |
| PostToolUseFailure | UPDATE tool_calls（succeeded=0） |
| Stop | INSERT stop_events, UPDATE sessions.ended_at |
| StopFailure | INSERT stop_events |
| SubagentStop | INSERT stop_events |
| SessionEnd | UPDATE sessions.ended_at + end_reason |

## Validation

功能完成后验证清单：

- [ ] 对应的 `go test` 全部通过
- [ ] `cc-track setup --check` 正常
- [ ] 手动在 Claude Code 中触发对应事件，DB 中有正确数据
- [ ] `--json` flag 输出合法 JSON
