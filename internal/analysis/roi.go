package analysis

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/shenghuikevin/cc-track/internal/store"
)

// GitStats holds lines added/removed and commit count from git.
type GitStats struct {
	Commits      int `json:"commits"`
	LinesAdded   int `json:"lines_added"`
	LinesRemoved int `json:"lines_removed"`
}

// ROIReport holds the full ROI analysis result.
type ROIReport struct {
	SinceMs        int64  `json:"since_ms"`
	UntilMs        int64  `json:"until_ms"`
	TotalSessions  int    `json:"total_sessions"`
	TotalPrompts   int    `json:"total_prompts"`
	TotalToolCalls int    `json:"total_tool_calls"`
	TotalDurationMs int64 `json:"total_duration_ms"`
	Commits        int    `json:"commits"`
	LinesAdded     int    `json:"lines_added"`
	LinesRemoved   int    `json:"lines_removed"`
	ReposAnalyzed  int    `json:"repos_analyzed"`
}

// AnalyzeROI combines session data with git stats for the given time range.
func AnalyzeROI(s *store.Store, sinceMs, untilMs int64, repoOverride string) (*ROIReport, error) {
	data, err := s.QueryROISessions(sinceMs, untilMs)
	if err != nil {
		return nil, fmt.Errorf("analysis: %w", err)
	}

	report := &ROIReport{
		SinceMs:         sinceMs,
		UntilMs:         untilMs,
		TotalSessions:   data.TotalSessions,
		TotalPrompts:    data.TotalPrompts,
		TotalToolCalls:  data.TotalToolCalls,
		TotalDurationMs: data.TotalDurationMs,
	}

	repos := data.Repos
	if repoOverride != "" {
		repos = []string{repoOverride}
	}

	since := time.UnixMilli(sinceMs).UTC().Format(time.RFC3339)
	until := time.UnixMilli(untilMs).UTC().Format(time.RFC3339)

	seen := make(map[string]bool)
	for _, repo := range repos {
		root, err := gitTopLevel(repo)
		if err != nil {
			continue // not a git repo, skip
		}
		if seen[root] {
			continue
		}
		seen[root] = true

		stats, err := gitStats(root, since, until)
		if err != nil {
			continue // best-effort
		}
		report.Commits += stats.Commits
		report.LinesAdded += stats.LinesAdded
		report.LinesRemoved += stats.LinesRemoved
	}
	report.ReposAnalyzed = len(seen)

	return report, nil
}

// gitTopLevel returns the git repo root for the given directory.
func gitTopLevel(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// gitStats returns commit count and lines added/removed using git log --numstat.
func gitStats(repoDir, since, until string) (*GitStats, error) {
	cmd := exec.Command("git", "-C", repoDir, "log",
		"--since="+since, "--until="+until,
		"--pretty=format:COMMIT", "--numstat")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("analysis: git log: %w", err)
	}

	stats := &GitStats{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "COMMIT" {
			stats.Commits++
			continue
		}
		if line == "" {
			continue
		}
		// numstat format: added\tremoved\tfilename
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		// Binary files show "-" for added/removed
		if parts[0] == "-" || parts[1] == "-" {
			continue
		}
		added, err1 := strconv.Atoi(parts[0])
		removed, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil {
			stats.LinesAdded += added
			stats.LinesRemoved += removed
		}
	}

	return stats, nil
}
