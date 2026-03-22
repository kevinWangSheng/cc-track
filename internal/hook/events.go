package hook

import "encoding/json"

type BaseEvent struct {
	SessionID      string `json:"session_id"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	AgentID        string `json:"agent_id,omitempty"`
	AgentType      string `json:"agent_type,omitempty"`
}

type SessionStartEvent struct {
	BaseEvent
	Model  string `json:"model"`
	Source string `json:"source"`
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
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
	ToolUseID string          `json:"tool_use_id"`
	Error     string          `json:"error"`
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
	Reason string `json:"reason"`
}
