# Hook 事件结构体

所有事件共享 BaseEvent，按 hook_event_name 分发到具体类型。

```go
type BaseEvent struct {
    SessionID     string `json:"session_id"`
    CWD           string `json:"cwd"`
    HookEventName string `json:"hook_event_name"`
    AgentID       string `json:"agent_id,omitempty"`
    AgentType     string `json:"agent_type,omitempty"`
}

type SessionStartEvent struct {
    BaseEvent
    Model  string `json:"model"`
    Source string `json:"source"` // startup, resume, clear, compact
}

type UserPromptSubmitEvent struct {
    BaseEvent
    Prompt string `json:"prompt"`
}

type PreToolUseEvent struct {
    BaseEvent
    ToolName  string          `json:"tool_name"`
    ToolInput json.RawMessage `json:"tool_input"`
    ToolUseID string          `json:"tool_use_id"`
}

type PostToolUseEvent struct {
    BaseEvent
    ToolName     string          `json:"tool_name"`
    ToolInput    json.RawMessage `json:"tool_input"`
    ToolResponse json.RawMessage `json:"tool_response"`
    ToolUseID    string          `json:"tool_use_id"`
}

type PostToolUseFailureEvent struct {
    BaseEvent
    ToolName     string          `json:"tool_name"`
    ToolInput    json.RawMessage `json:"tool_input"`
    ToolUseID    string          `json:"tool_use_id"`
    Error        string          `json:"error"`
}

type StopEvent struct {
    BaseEvent
    StopHookActive       bool   `json:"stop_hook_active"`
    LastAssistantMessage string `json:"last_assistant_message"`
}

type StopFailureEvent struct {
    BaseEvent
    Error                string `json:"error"`
    ErrorDetails         string `json:"error_details"`
    LastAssistantMessage string `json:"last_assistant_message"`
}

type SubagentStopEvent struct {
    BaseEvent
    LastAssistantMessage string `json:"last_assistant_message"`
}

type SessionEndEvent struct {
    BaseEvent
    Reason string `json:"reason"` // clear, resume, logout, prompt_input_exit
}
```

## Handler 分发逻辑

```go
func HandleEvent(data []byte, store *store.Store) error {
    var base BaseEvent
    if err := json.Unmarshal(data, &base); err != nil {
        return fmt.Errorf("hook: parse base: %w", err)
    }
    switch base.HookEventName {
    case "SessionStart":
        var e SessionStartEvent
        // unmarshal + store.UpsertSession(e)
    case "PreToolUse":
        var e PreToolUseEvent
        // unmarshal + store.InsertToolCall(e)
    case "PostToolUse":
        var e PostToolUseEvent
        // unmarshal + store.CompleteToolCall(e)
    // ... 其他事件
    }
}
```
