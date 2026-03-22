---
name: conventions
description: cc-track 代码规范。Tool Wrapper 模式——只包含项目特有约定，agent 写代码时自动加载。写完代码后按 references/review-checklist.md 自审。
---

# cc-track 代码规范

只列项目特有规则。Go 通用实践不再重复。

## SQLite 用法

```go
import _ "modernc.org/sqlite"

db, err := sql.Open("sqlite", dbPath)  // 驱动名 "sqlite" 不是 "sqlite3"
```

## Error 包装

```go
return fmt.Errorf("store: open db: %w", err)
return fmt.Errorf("hook: parse event: %w", err)
// 前缀 = 包名，后接具体操作
```

## 截断

```go
const maxFieldBytes = 10240

func truncate(s string, max int) string {
    if len(s) <= max { return s }
    return s[:max] + "...[truncated]"
}
// 对 tool_input_json 和 tool_output_json 在写 DB 前调用
```

## 包职责边界

| 包 | 可以 | 不可以 |
|---|------|--------|
| `cmd/` | fmt.Print, os.Exit, log | 直接操作 DB |
| `internal/store/` | sql.DB 操作 | fmt.Print, log, 业务逻辑 |
| `internal/hook/` | JSON 解析, 调用 store | fmt.Print, log |
| `internal/analysis/` | 调用 store 查询, 计算 | fmt.Print, 直接 DB 操作 |

## Validation Loop

写完代码后自审：

1. 运行 `go build ./...` — 编译通过
2. 运行 `go test ./...` — 测试通过
3. 运行 `go vet ./...` — 无警告
4. 对照 [references/review-checklist.md](references/review-checklist.md) 检查

如果任何一步失败，修复后重新运行，直到全部通过。
