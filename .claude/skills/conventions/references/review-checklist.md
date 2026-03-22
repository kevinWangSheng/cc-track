# Code Review Checklist

按严重程度分组。每次提交前逐项检查。

## Critical（必须修复）

- [ ] 没有使用 CGO 或 `github.com/mattn/go-sqlite3`
- [ ] SQL 语句全部用 `?` 占位符，无字符串拼接
- [ ] `internal/` 层没有 `fmt.Print`、`log.*`、`os.Exit`
- [ ] error 全部被处理或显式忽略（`_ = `）
- [ ] 新增/修改的函数有对应测试

## Warning（应该修复）

- [ ] error 消息有包名前缀：`fmt.Errorf("pkg: action: %w", err)`
- [ ] JSON tag 使用 snake_case
- [ ] 时间戳使用 `time.Now().UnixMilli()`（不是 Unix()）
- [ ] tool_input/tool_output 写 DB 前经过 truncate
- [ ] SQLite 连接设置了 WAL 和 busy_timeout

## Info（建议改进）

- [ ] 函数签名接收具体类型而非 interface{}（除非必要）
- [ ] 测试使用 table-driven 格式（多 case 场景）
- [ ] 测试用 `:memory:` DB，不写临时文件
