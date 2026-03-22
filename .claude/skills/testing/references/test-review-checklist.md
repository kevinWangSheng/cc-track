# Test Review Checklist

## Critical

- [ ] 测试使用 `:memory:` DB，不写文件系统
- [ ] 每个测试函数独立 DB 实例（不共享状态）
- [ ] 没有 mock SQLite（用真实 DB 操作）
- [ ] 测试覆盖了正常路径和错误路径
- [ ] 断言检查了具体值，不只是 `err == nil`

## Warning

- [ ] 多场景测试使用 table-driven 格式
- [ ] 测试用了 `t.Helper()` 标记 helper 函数
- [ ] 测试用了 `t.Cleanup()` 清理资源
- [ ] fixture JSON 文件在 testdata/ 目录下
- [ ] 测试名称描述了场景而非实现：`TestCompleteToolCall_MissingPreToolUse` 而非 `TestFunc3`

## Info

- [ ] 边界条件有覆盖（空输入、超大输入、并发写入）
- [ ] 集成测试用 `//go:build integration` 隔离
- [ ] 浪费检测算法测试了阈值边界（刚好达到/未达到）
