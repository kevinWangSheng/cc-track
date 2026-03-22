package hook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shenghuikevin/cc-track/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestHandleAllEvents(t *testing.T) {
	tests := []struct {
		fixture string
		name    string
	}{
		{"session_start.json", "SessionStart"},
		{"user_prompt_submit.json", "UserPromptSubmit"},
		{"pre_tool_use.json", "PreToolUse"},
		{"post_tool_use.json", "PostToolUse"},
		{"post_tool_use_failure.json", "PostToolUseFailure"},
		{"stop.json", "Stop"},
		{"stop_failure.json", "StopFailure"},
		{"subagent_stop.json", "SubagentStop"},
		{"session_end.json", "SessionEnd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := openTestStore(t)

			// Most events need a session to exist first
			if tt.name != "SessionStart" {
				data := loadFixture(t, "session_start.json")
				if err := HandleEvent(data, s); err != nil {
					t.Fatalf("setup SessionStart: %v", err)
				}
			}

			// PreToolUse needed before PostToolUseFailure
			if tt.name == "PostToolUseFailure" {
				pre := []byte(`{"session_id":"sess-001","cwd":"/tmp/testproject","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"rm -rf /"},"tool_use_id":"tu-002"}`)
				if err := HandleEvent(pre, s); err != nil {
					t.Fatalf("setup PreToolUse: %v", err)
				}
			}

			data := loadFixture(t, tt.fixture)
			if err := HandleEvent(data, s); err != nil {
				t.Errorf("HandleEvent(%s) error: %v", tt.name, err)
			}
		})
	}
}

func TestHandleUnknownEvent(t *testing.T) {
	s := openTestStore(t)
	data := []byte(`{"session_id":"s1","cwd":"/tmp","hook_event_name":"Unknown"}`)
	err := HandleEvent(data, s)
	if err == nil {
		t.Error("expected error for unknown event")
	}
}

func TestHandleInvalidJSON(t *testing.T) {
	s := openTestStore(t)
	err := HandleEvent([]byte(`{invalid`), s)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
