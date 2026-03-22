package hook

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/shenghuikevin/cc-track/internal/transcript"
)

// HandleEvent parses a hook event JSON and dispatches to the store.
func HandleEvent(data []byte, s *store.Store) error {
	var base BaseEvent
	if err := json.Unmarshal(data, &base); err != nil {
		return fmt.Errorf("hook: parse base: %w", err)
	}

	switch base.HookEventName {
	case "SessionStart":
		return handleSessionStart(data, base, s)
	case "UserPromptSubmit":
		return handleUserPromptSubmit(data, base, s)
	case "PreToolUse":
		return handlePreToolUse(data, s)
	case "PostToolUse":
		return handlePostToolUse(data, s)
	case "PostToolUseFailure":
		return handlePostToolUseFailure(data, s)
	case "Stop":
		return handleStop(base, s)
	case "StopFailure":
		return handleStopFailure(data, base, s)
	case "SubagentStop":
		return handleSubagentStop(base, s)
	case "SessionEnd":
		return handleSessionEnd(data, base, s)
	default:
		return fmt.Errorf("hook: unknown event: %s", base.HookEventName)
	}
}

func handleSessionStart(data []byte, base BaseEvent, s *store.Store) error {
	var e SessionStartEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("hook: parse SessionStart: %w", err)
	}

	project := deriveProject(e.CWD)
	branch := deriveBranch(e.CWD)
	return s.UpsertSession(e.SessionID, e.CWD, project, branch, e.Model)
}

func handleUserPromptSubmit(data []byte, base BaseEvent, s *store.Store) error {
	var e UserPromptSubmitEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("hook: parse UserPromptSubmit: %w", err)
	}

	if err := s.InsertPrompt(e.SessionID, e.Prompt); err != nil {
		return err
	}
	return s.IncrementPrompts(e.SessionID)
}

func handlePreToolUse(data []byte, s *store.Store) error {
	var e PreToolUseEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("hook: parse PreToolUse: %w", err)
	}
	return s.InsertToolCall(e.SessionID, e.ToolUseID, e.ToolName, string(e.ToolInput))
}

func handlePostToolUse(data []byte, s *store.Store) error {
	var e PostToolUseEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("hook: parse PostToolUse: %w", err)
	}

	if err := s.CompleteToolCall(e.SessionID, e.ToolUseID, e.ToolName, string(e.ToolInput), string(e.ToolResponse)); err != nil {
		return err
	}
	return s.IncrementToolCalls(e.SessionID)
}

func handlePostToolUseFailure(data []byte, s *store.Store) error {
	var e PostToolUseFailureEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("hook: parse PostToolUseFailure: %w", err)
	}
	return s.FailToolCall(e.ToolUseID, e.Error)
}

func handleStop(base BaseEvent, s *store.Store) error {
	if err := s.InsertStopEvent(base.SessionID, "Stop", "", ""); err != nil {
		return err
	}
	collectTokenUsage(base, s)
	return s.EndSession(base.SessionID, "stop")
}

func handleStopFailure(data []byte, base BaseEvent, s *store.Store) error {
	var e StopFailureEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("hook: parse StopFailure: %w", err)
	}
	return s.InsertStopEvent(base.SessionID, "StopFailure", e.Error, e.ErrorDetails)
}

func handleSubagentStop(base BaseEvent, s *store.Store) error {
	return s.InsertStopEvent(base.SessionID, "SubagentStop", "", "")
}

func handleSessionEnd(data []byte, base BaseEvent, s *store.Store) error {
	var e SessionEndEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("hook: parse SessionEnd: %w", err)
	}
	collectTokenUsage(base, s)
	return s.EndSession(e.SessionID, e.Reason)
}

// collectTokenUsage parses the transcript file and stores token totals.
// Errors are silently ignored — token tracking is best-effort.
func collectTokenUsage(base BaseEvent, s *store.Store) {
	if base.TranscriptPath == "" {
		return
	}
	usage, err := transcript.ParseFile(base.TranscriptPath)
	if err != nil {
		return
	}
	_ = s.UpdateTokenUsage(
		base.SessionID,
		usage.InputTokens,
		usage.OutputTokens,
		usage.CacheReadInputTokens,
		usage.CacheCreationInputTokens,
	)
}

func deriveProject(cwd string) string {
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return filepath.Base(cwd)
	}
	return filepath.Base(strings.TrimSpace(string(out)))
}

func deriveBranch(cwd string) string {
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
