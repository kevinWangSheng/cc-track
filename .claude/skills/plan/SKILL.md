---
name: plan
description: cc-track 技术架构和设计决策。做架构相关改动、理解系统设计、或需要了解技术决策原因时使用。Tool Wrapper 模式——按需加载特定技术的详细参考。
---

# cc-track 架构设计

## 数据流

```
Claude Code → hook 事件 → spawn cc-track hook → stdin JSON → SQLite
                                                      │
                               ┌──────────────────────┼─────────────────┐
                               ▼                      ▼                 ▼
                        cc-track summary      cc-track waste    cc-track roi
```

## 技术决策

| 决策 | 选择 | 原因 |
|------|------|------|
| 运行方式 | 无 daemon，短生命周期进程 | 个人工具不应管理后台进程。Go 启动 <10ms |
| SQLite 驱动 | modernc.org/sqlite | 纯 Go，无 CGO，`go install` 跨平台即用 |
| 数据采集 | command hook（stdin） | 无需 HTTP server，无端口冲突，崩溃不影响下次 |
| Token/Cost | 分阶段获取 | hooks 不含 token 数据，先读 stats-cache.json |

当需要了解具体技术细节时，加载对应 reference：
- Hook 事件结构体：[references/hook-events.md](references/hook-events.md)
- 并发安全设计：[references/concurrency.md](references/concurrency.md)

## 项目结构

```
cmd/         → cobra 命令，处理 flags 和用户输出
internal/
  hook/      → JSON 解析和事件分发
  store/     → DB 操作（纯数据层，无业务逻辑）
  analysis/  → 查询 + 计算，返回结构体
  config/    → 路径常量和配置
  output/    → 终端格式化（table/json）
```

## Gotchas

- store 层的方法签名接收具体 event struct，不接收 raw JSON
- handler.go 做两遍解析：先 BaseEvent 读 hook_event_name，再按类型解析具体 struct
- CompleteToolCall 必须先 UPDATE 再 fallback INSERT（处理竞态），不能只 INSERT
- setup 合并 settings.json 时必须用 JSON 解析，不能字符串替换
