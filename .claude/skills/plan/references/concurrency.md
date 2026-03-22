# 并发安全设计

## 问题

Claude Code hooks 是 async 的，多个 hook 进程可能同时写 SQLite。

## 方案

### SQLite 层
- `PRAGMA journal_mode=WAL`：允许并发读写
- `PRAGMA busy_timeout=5000`：写冲突时等待 5 秒重试
- 每个 hook 进程打开独立连接，写完即关

### PreToolUse/PostToolUse 竞态

PostToolUse 可能在 PreToolUse 之前到达（极端情况）。处理方式：

```go
func (s *Store) CompleteToolCall(e PostToolUseEvent) error {
    result, err := s.db.Exec(
        "UPDATE tool_calls SET tool_output_json=?, completed_at=?, duration_ms=?, succeeded=1 WHERE tool_use_id=?",
        truncate(string(e.ToolResponse), 10240),
        time.Now().UnixMilli(),
        /* duration */,
        e.ToolUseID,
    )
    if err != nil {
        return fmt.Errorf("store: complete tool call: %w", err)
    }
    affected, _ := result.RowsAffected()
    if affected == 0 {
        // PreToolUse 还没到，INSERT 完整行
        return s.insertFullToolCall(e)
    }
    return nil
}
```

### 无需应用层锁

SQLite WAL + busy_timeout 已经足够。hook 进程生命周期极短（<100ms），冲突概率极低。
