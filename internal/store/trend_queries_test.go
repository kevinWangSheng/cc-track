package store

import (
	"testing"
	"time"
)

func TestQueryDailyStats_Empty(t *testing.T) {
	s := openTestDB(t)

	now := time.Now()
	stats, err := s.QueryDailyStats(now.Add(-24*time.Hour).UnixMilli(), now.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 0 {
		t.Fatalf("expected 0 days, got %d", len(stats))
	}
}

func TestQueryDailyStats_GroupsByDay(t *testing.T) {
	s := openTestDB(t)

	// Insert 2 sessions on "today"
	if err := s.UpsertSession("s1", "/tmp", "proj", "main", "opus"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertSession("s2", "/tmp", "proj", "main", "opus"); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	stats, err := s.QueryDailyStats(now.Add(-1*time.Hour).UnixMilli(), now.Add(1*time.Hour).UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 day, got %d", len(stats))
	}
	if stats[0].Sessions != 2 {
		t.Fatalf("expected 2 sessions, got %d", stats[0].Sessions)
	}
}

func TestQueryDailySessionIDs(t *testing.T) {
	s := openTestDB(t)

	if err := s.UpsertSession("s1", "/tmp", "proj", "main", "opus"); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	byDay, err := s.QueryDailySessionIDs(now.Add(-1*time.Hour).UnixMilli(), now.Add(1*time.Hour).UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	if len(byDay) != 1 {
		t.Fatalf("expected 1 day, got %d", len(byDay))
	}
	for _, ids := range byDay {
		if len(ids) != 1 || ids[0] != "s1" {
			t.Fatalf("expected [s1], got %v", ids)
		}
	}
}

