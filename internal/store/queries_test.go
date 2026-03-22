package store

import (
	"testing"
	"time"
)

func seedTestData(t *testing.T, s *Store) {
	t.Helper()
	now := time.Now().UnixMilli()

	_ = s.UpsertSession("sess-100", "/tmp/proj", "proj", "main", "opus")
	_ = s.InsertPrompt("sess-100", "hello")
	_ = s.IncrementPrompts("sess-100")
	_ = s.InsertToolCall("sess-100", "tu-100", "Read", `{"file":"a.go"}`)
	_ = s.CompleteToolCall("sess-100", "tu-100", "Read", `{"file":"a.go"}`, `{"ok":true}`)
	_ = s.IncrementToolCalls("sess-100")
	_ = s.InsertToolCall("sess-100", "tu-101", "Bash", `{"cmd":"ls"}`)
	_ = s.FailToolCall("tu-101", "denied")
	_ = s.IncrementToolCalls("sess-100")
	_ = s.EndSession("sess-100", "stop")

	// Verify session has a valid started_at
	var startedAt int64
	s.db.QueryRow("SELECT started_at FROM sessions WHERE id = ?", "sess-100").Scan(&startedAt)
	if startedAt == 0 {
		t.Fatalf("session started_at is 0")
	}
	_ = now
}

func TestQuerySummary(t *testing.T) {
	s := openTestDB(t)
	seedTestData(t, s)

	sinceMs := time.Now().Add(-1 * time.Hour).UnixMilli()
	untilMs := time.Now().Add(1 * time.Hour).UnixMilli()

	sum, err := s.QuerySummary(sinceMs, untilMs)
	if err != nil {
		t.Fatalf("QuerySummary: %v", err)
	}

	if sum.TotalSessions != 1 {
		t.Errorf("TotalSessions = %d, want 1", sum.TotalSessions)
	}
	if sum.TotalPrompts != 1 {
		t.Errorf("TotalPrompts = %d, want 1", sum.TotalPrompts)
	}
	if sum.TotalToolCalls != 2 {
		t.Errorf("TotalToolCalls = %d, want 2", sum.TotalToolCalls)
	}
	if len(sum.ToolBreakdown) == 0 {
		t.Error("ToolBreakdown is empty")
	}
	if sum.ErrorRate == 0 {
		t.Error("ErrorRate should be > 0")
	}
}

func TestListSessions(t *testing.T) {
	s := openTestDB(t)
	seedTestData(t, s)

	rows, err := s.ListSessions(10)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("ListSessions returned %d rows, want 1", len(rows))
	}
}

func TestFindSessionByPrefix(t *testing.T) {
	s := openTestDB(t)
	seedTestData(t, s)

	sess, err := s.FindSessionByPrefix("sess-1")
	if err != nil {
		t.Fatalf("FindSessionByPrefix: %v", err)
	}
	if sess.ID != "sess-100" {
		t.Errorf("found session %q, want sess-100", sess.ID)
	}

	_, err = s.FindSessionByPrefix("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent prefix")
	}
}

func TestUpdateTokenUsage(t *testing.T) {
	s := openTestDB(t)
	_ = s.UpsertSession("sess-tok", "/tmp", "", "", "opus")

	err := s.UpdateTokenUsage("sess-tok", 1000, 500, 2000, 300)
	if err != nil {
		t.Fatalf("UpdateTokenUsage: %v", err)
	}

	var in, out, cacheRead, cacheCreate int64
	s.db.QueryRow(`SELECT total_input_tokens, total_output_tokens, total_cache_read_tokens, total_cache_creation_tokens
		FROM sessions WHERE id = ?`, "sess-tok").Scan(&in, &out, &cacheRead, &cacheCreate)

	if in != 1000 || out != 500 || cacheRead != 2000 || cacheCreate != 300 {
		t.Errorf("tokens = (%d,%d,%d,%d), want (1000,500,2000,300)", in, out, cacheRead, cacheCreate)
	}
}

func TestQuerySummaryWithTokens(t *testing.T) {
	s := openTestDB(t)
	_ = s.UpsertSession("sess-t1", "/tmp", "", "", "opus")
	_ = s.UpdateTokenUsage("sess-t1", 100, 50, 200, 30)
	_ = s.UpsertSession("sess-t2", "/tmp", "", "", "opus")
	_ = s.UpdateTokenUsage("sess-t2", 400, 150, 800, 70)

	sinceMs := time.Now().Add(-1 * time.Hour).UnixMilli()
	untilMs := time.Now().Add(1 * time.Hour).UnixMilli()
	sum, err := s.QuerySummary(sinceMs, untilMs)
	if err != nil {
		t.Fatalf("QuerySummary: %v", err)
	}
	if sum.TotalInputTokens != 500 {
		t.Errorf("TotalInputTokens = %d, want 500", sum.TotalInputTokens)
	}
	if sum.TotalOutputTokens != 200 {
		t.Errorf("TotalOutputTokens = %d, want 200", sum.TotalOutputTokens)
	}
	if sum.TotalCacheReadTokens != 1000 {
		t.Errorf("TotalCacheReadTokens = %d, want 1000", sum.TotalCacheReadTokens)
	}
	if sum.TotalCacheCreationTokens != 100 {
		t.Errorf("TotalCacheCreationTokens = %d, want 100", sum.TotalCacheCreationTokens)
	}
}

func TestGetSessionTimeline(t *testing.T) {
	s := openTestDB(t)
	seedTestData(t, s)

	tl, err := s.GetSessionTimeline("sess-100")
	if err != nil {
		t.Fatalf("GetSessionTimeline: %v", err)
	}

	if len(tl.Prompts) != 1 {
		t.Errorf("Prompts = %d, want 1", len(tl.Prompts))
	}
	if len(tl.ToolCalls) != 2 {
		t.Errorf("ToolCalls = %d, want 2", len(tl.ToolCalls))
	}
}
