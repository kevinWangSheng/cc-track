package store

import (
	"testing"
)

func openTestDB(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenAndMigrate(t *testing.T) {
	s := openTestDB(t)

	var version int
	err := s.db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Errorf("schema version = %d, want %d", version, currentSchemaVersion)
	}
}

func TestUpsertSession(t *testing.T) {
	s := openTestDB(t)

	err := s.UpsertSession("sess-1", "/home/user/proj", "myproject", "main", "opus")
	if err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}

	var cwd, model string
	err = s.db.QueryRow("SELECT cwd, model FROM sessions WHERE id = ?", "sess-1").Scan(&cwd, &model)
	if err != nil {
		t.Fatalf("query session: %v", err)
	}
	if cwd != "/home/user/proj" || model != "opus" {
		t.Errorf("session = (%q, %q), want (/home/user/proj, opus)", cwd, model)
	}

	// upsert should update model
	err = s.UpsertSession("sess-1", "/home/user/proj", "myproject", "main", "sonnet")
	if err != nil {
		t.Fatalf("UpsertSession (update): %v", err)
	}
	err = s.db.QueryRow("SELECT model FROM sessions WHERE id = ?", "sess-1").Scan(&model)
	if err != nil {
		t.Fatalf("query session after upsert: %v", err)
	}
	if model != "sonnet" {
		t.Errorf("model after upsert = %q, want sonnet", model)
	}
}

func TestEndSession(t *testing.T) {
	s := openTestDB(t)
	_ = s.UpsertSession("sess-1", "/tmp", "", "", "")

	err := s.EndSession("sess-1", "logout")
	if err != nil {
		t.Fatalf("EndSession: %v", err)
	}

	var endReason string
	var endedAt int64
	err = s.db.QueryRow("SELECT ended_at, end_reason FROM sessions WHERE id = ?", "sess-1").Scan(&endedAt, &endReason)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if endedAt == 0 {
		t.Error("ended_at should be set")
	}
	if endReason != "logout" {
		t.Errorf("end_reason = %q, want logout", endReason)
	}
}

func TestInsertPromptAndIncrement(t *testing.T) {
	s := openTestDB(t)
	_ = s.UpsertSession("sess-1", "/tmp", "", "", "")

	err := s.InsertPrompt("sess-1", "hello world")
	if err != nil {
		t.Fatalf("InsertPrompt: %v", err)
	}
	err = s.IncrementPrompts("sess-1")
	if err != nil {
		t.Fatalf("IncrementPrompts: %v", err)
	}

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM prompts WHERE session_id = ?", "sess-1").Scan(&count)
	if count != 1 {
		t.Errorf("prompts count = %d, want 1", count)
	}

	var total int
	s.db.QueryRow("SELECT total_prompts FROM sessions WHERE id = ?", "sess-1").Scan(&total)
	if total != 1 {
		t.Errorf("total_prompts = %d, want 1", total)
	}
}

func TestToolCallLifecycle(t *testing.T) {
	s := openTestDB(t)
	_ = s.UpsertSession("sess-1", "/tmp", "", "", "")

	// PreToolUse
	err := s.InsertToolCall("sess-1", "tu-1", "Read", `{"file":"foo.go"}`)
	if err != nil {
		t.Fatalf("InsertToolCall: %v", err)
	}

	// PostToolUse
	err = s.CompleteToolCall("sess-1", "tu-1", "Read", `{"file":"foo.go"}`, `{"content":"..."}`)
	if err != nil {
		t.Fatalf("CompleteToolCall: %v", err)
	}

	var succeeded int
	s.db.QueryRow("SELECT succeeded FROM tool_calls WHERE tool_use_id = ?", "tu-1").Scan(&succeeded)
	if succeeded != 1 {
		t.Errorf("succeeded = %d, want 1", succeeded)
	}
}

func TestCompleteToolCallWithoutPre(t *testing.T) {
	s := openTestDB(t)
	_ = s.UpsertSession("sess-1", "/tmp", "", "", "")

	// PostToolUse arrives before PreToolUse
	err := s.CompleteToolCall("sess-1", "tu-2", "Write", `{}`, `{"ok":true}`)
	if err != nil {
		t.Fatalf("CompleteToolCall (no pre): %v", err)
	}

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM tool_calls WHERE tool_use_id = ?", "tu-2").Scan(&count)
	if count != 1 {
		t.Errorf("tool_calls count = %d, want 1", count)
	}
}

func TestFailToolCall(t *testing.T) {
	s := openTestDB(t)
	_ = s.UpsertSession("sess-1", "/tmp", "", "", "")
	_ = s.InsertToolCall("sess-1", "tu-3", "Bash", `{"cmd":"rm -rf /"}`)

	err := s.FailToolCall("tu-3", "permission denied")
	if err != nil {
		t.Fatalf("FailToolCall: %v", err)
	}

	var succeeded int
	var errMsg string
	s.db.QueryRow("SELECT succeeded, error_message FROM tool_calls WHERE tool_use_id = ?", "tu-3").Scan(&succeeded, &errMsg)
	if succeeded != 0 {
		t.Errorf("succeeded = %d, want 0", succeeded)
	}
	if errMsg != "permission denied" {
		t.Errorf("error_message = %q, want 'permission denied'", errMsg)
	}
}

func TestInsertStopEvent(t *testing.T) {
	s := openTestDB(t)
	_ = s.UpsertSession("sess-1", "/tmp", "", "", "")

	err := s.InsertStopEvent("sess-1", "Stop", "", "")
	if err != nil {
		t.Fatalf("InsertStopEvent: %v", err)
	}

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM stop_events WHERE session_id = ?", "sess-1").Scan(&count)
	if count != 1 {
		t.Errorf("stop_events count = %d, want 1", count)
	}
}

func TestTruncate(t *testing.T) {
	short := "hello"
	if got := truncate(short, 10); got != short {
		t.Errorf("truncate(%q, 10) = %q", short, got)
	}

	long := "abcdefghijk"
	got := truncate(long, 5)
	want := "abcde...[truncated]"
	if got != want {
		t.Errorf("truncate(%q, 5) = %q, want %q", long, got, want)
	}
}
