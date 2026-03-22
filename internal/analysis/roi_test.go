package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestAnalyzeROI_NoSessions(t *testing.T) {
	s := setupTestStore(t)

	now := time.Now()
	sinceMs := now.Add(-24 * time.Hour).UnixMilli()
	untilMs := now.UnixMilli()

	report, err := AnalyzeROI(s, sinceMs, untilMs, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalSessions != 0 {
		t.Fatalf("expected 0 sessions, got %d", report.TotalSessions)
	}
	if report.Commits != 0 {
		t.Fatalf("expected 0 commits, got %d", report.Commits)
	}
}

func TestAnalyzeROI_WithSessions(t *testing.T) {
	s := setupTestStore(t)

	now := time.Now()
	sinceMs := now.Add(-1 * time.Hour).UnixMilli()
	untilMs := now.Add(1 * time.Hour).UnixMilli()

	// Insert a session
	if err := s.UpsertSession("roi-1", "/tmp", "proj", "main", "opus"); err != nil {
		t.Fatal(err)
	}

	report, err := AnalyzeROI(s, sinceMs, untilMs, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalSessions != 1 {
		t.Fatalf("expected 1 session, got %d", report.TotalSessions)
	}
}

func TestAnalyzeROI_WithGitRepo(t *testing.T) {
	s := setupTestStore(t)

	// Create a temp git repo with a commit
	dir := t.TempDir()
	mustRun(t, dir, "git", "init")
	mustRun(t, dir, "git", "config", "user.email", "test@test.com")
	mustRun(t, dir, "git", "config", "user.name", "Test")

	// Create a file and commit
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "init")

	now := time.Now()
	sinceMs := now.Add(-1 * time.Hour).UnixMilli()
	untilMs := now.Add(1 * time.Hour).UnixMilli()

	// Insert session with CWD pointing to temp repo
	if err := s.UpsertSession("roi-git", dir, "test", "main", "opus"); err != nil {
		t.Fatal(err)
	}

	report, err := AnalyzeROI(s, sinceMs, untilMs, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalSessions != 1 {
		t.Fatalf("expected 1 session, got %d", report.TotalSessions)
	}
	if report.ReposAnalyzed != 1 {
		t.Fatalf("expected 1 repo analyzed, got %d", report.ReposAnalyzed)
	}
	if report.Commits != 1 {
		t.Fatalf("expected 1 commit, got %d", report.Commits)
	}
	if report.LinesAdded != 3 {
		t.Fatalf("expected 3 lines added, got %d", report.LinesAdded)
	}
}

func TestAnalyzeROI_RepoOverride(t *testing.T) {
	s := setupTestStore(t)

	dir := t.TempDir()
	mustRun(t, dir, "git", "init")
	mustRun(t, dir, "git", "config", "user.email", "test@test.com")
	mustRun(t, dir, "git", "config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "add file")

	now := time.Now()
	sinceMs := now.Add(-1 * time.Hour).UnixMilli()
	untilMs := now.Add(1 * time.Hour).UnixMilli()

	// Session points somewhere else, but we override repo
	if err := s.UpsertSession("roi-override", "/nonexistent", "proj", "main", "opus"); err != nil {
		t.Fatal(err)
	}

	report, err := AnalyzeROI(s, sinceMs, untilMs, dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Commits != 1 {
		t.Fatalf("expected 1 commit with repo override, got %d", report.Commits)
	}
}

func TestAnalyzeROI_DeduplicatesSameRepo(t *testing.T) {
	s := setupTestStore(t)

	dir := t.TempDir()
	mustRun(t, dir, "git", "init")
	mustRun(t, dir, "git", "config", "user.email", "test@test.com")
	mustRun(t, dir, "git", "config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "one")

	now := time.Now()
	sinceMs := now.Add(-1 * time.Hour).UnixMilli()
	untilMs := now.Add(1 * time.Hour).UnixMilli()

	// Two sessions in the same repo (one in a subdirectory)
	subdir := filepath.Join(dir, "sub")
	os.MkdirAll(subdir, 0755)
	if err := s.UpsertSession("roi-dup1", dir, "proj", "main", "opus"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertSession("roi-dup2", subdir, "proj", "main", "opus"); err != nil {
		t.Fatal(err)
	}

	report, err := AnalyzeROI(s, sinceMs, untilMs, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.ReposAnalyzed != 1 {
		t.Fatalf("expected 1 repo (deduped), got %d", report.ReposAnalyzed)
	}
	// Should count commits only once
	if report.Commits != 1 {
		t.Fatalf("expected 1 commit (not double counted), got %d", report.Commits)
	}
}

func mustRun(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}
