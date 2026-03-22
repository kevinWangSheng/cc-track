---
name: testing
description: cc-track 测试规范。Reviewer 模式——按 checklist 验证测试质量。写测试或审查测试时使用。
---

# cc-track 测试规范

## 核心策略

- SQLite 用 `:memory:` 真实 DB，不 mock
- 每个测试函数独立 DB 实例
- Hook 解析用 testdata/ JSON fixtures
- 集成测试用 `//go:build integration` 隔离

## 写测试流程

1. 确定要测试的函数和场景
2. 从 [assets/test-template.go](assets/test-template.go) 复制模板
3. 填入具体 test cases
4. 运行 `go test ./path/to/package/...`
5. 对照 [references/test-review-checklist.md](references/test-review-checklist.md) 自审

## Test Helper

每个需要 DB 的测试包中提供：

```go
func newTestDB(t *testing.T) *Store {
    t.Helper()
    s, err := Open(":memory:")
    if err != nil { t.Fatal(err) }
    t.Cleanup(func() { s.Close() })
    return s
}
```

## Fixtures

```
internal/hook/testdata/
├── session_start.json
├── user_prompt_submit.json
├── pre_tool_use.json
├── post_tool_use.json
├── post_tool_use_failure.json
├── stop.json
├── stop_failure.json
├── subagent_stop.json
├── session_end.json
└── invalid.json
```

每个 fixture 必须是 Claude Code hook 实际发送的完整 JSON。

## Validation Loop

测试写完后：

1. `go test ./...` — 全部通过
2. `go test -race ./...` — 无 race condition
3. 对照 [references/test-review-checklist.md](references/test-review-checklist.md) 检查
4. 不通过则修复后重新运行
