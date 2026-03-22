# 浪费检测规则 Checklist

供 `internal/analysis/waste.go` 实现和测试时参照。

## 5 种检测模式

### 1. 重复工具调用
- **规则**：同一 session 内，相同 tool_name + 相似 tool_input，60 秒内出现 3+ 次
- **相似判定**：
  - Read：相同 file_path
  - Bash：相同 command
  - Grep：相同 pattern + path
  - Edit：相同 file_path + old_string
- **测试要点**：正好 2 次不触发，3 次触发；不同 tool_name 不触发；超过 60s 窗口不触发

### 2. 过度文件读取
- **规则**：同一 session 内，相同文件被 Read 5+ 次
- **测试要点**：4 次不触发，5 次触发；不同文件分别计数

### 3. 失败重试
- **规则**：同一工具连续失败 3+ 次，tool_input 相似
- **测试要点**：2 次连续失败不触发；中间有成功则重置计数

### 4. Edit 反复
- **规则**：同一文件的 Edit 出现 A→B→A 模式（revert）
- **检测**：对比 Edit 的 tool_input 中 old_string/new_string，如果 N+2 的 old_string 等于 N 的 new_string
- **测试要点**：A→B→C 不触发；A→B→A 触发

### 5. 僵尸 Session
- **规则**：session 时长 >30min 但 tool_calls <5 且 prompts <3
- **测试要点**：31min + 4 calls + 2 prompts 触发；29min 不触发；6 calls 不触发
