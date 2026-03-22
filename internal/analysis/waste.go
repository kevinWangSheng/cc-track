package analysis

import (
	"encoding/json"
	"fmt"

	"github.com/shenghuikevin/cc-track/internal/store"
)

// WasteType identifies the kind of waste detected.
type WasteType string

const (
	WasteDuplicateCalls   WasteType = "duplicate_calls"
	WasteExcessiveReads   WasteType = "excessive_reads"
	WasteFailedRetries    WasteType = "failed_retries"
	WasteEditRevert       WasteType = "edit_revert"
	WasteZombieSession    WasteType = "zombie_session"
)

// Finding represents a single waste detection result.
type Finding struct {
	Type      WasteType `json:"type"`
	SessionID string    `json:"session_id"`
	Summary   string    `json:"summary"`
	Details   string    `json:"details,omitempty"`
	Count     int       `json:"count,omitempty"`
}

// WasteReport holds all findings from waste analysis.
type WasteReport struct {
	SessionsAnalyzed int       `json:"sessions_analyzed"`
	Findings         []Finding `json:"findings"`
}

// AnalyzeWaste runs all 5 waste detectors on the given sessions.
func AnalyzeWaste(s *store.Store, sessionIDs []string) (*WasteReport, error) {
	report := &WasteReport{SessionsAnalyzed: len(sessionIDs)}

	for _, sid := range sessionIDs {
		calls, err := s.GetToolCallsForSession(sid)
		if err != nil {
			return nil, fmt.Errorf("analysis: %w", err)
		}

		report.Findings = append(report.Findings, detectDuplicateCalls(sid, calls)...)
		report.Findings = append(report.Findings, detectExcessiveReads(sid, calls)...)
		report.Findings = append(report.Findings, detectFailedRetries(sid, calls)...)
		report.Findings = append(report.Findings, detectEditReverts(sid, calls)...)
	}

	// Zombie detection
	zombies, err := s.GetZombieCandidates(sessionIDs)
	if err != nil {
		return nil, fmt.Errorf("analysis: %w", err)
	}
	report.Findings = append(report.Findings, detectZombieSessions(zombies)...)

	return report, nil
}

// toolInputKey extracts a comparable key from tool_input_json based on tool type.
func toolInputKey(toolName, inputJSON string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &m); err != nil {
		return inputJSON
	}

	switch toolName {
	case "Read":
		if fp, ok := m["file_path"]; ok {
			return fmt.Sprintf("%v", fp)
		}
	case "Bash":
		if cmd, ok := m["command"]; ok {
			return fmt.Sprintf("%v", cmd)
		}
	case "Grep":
		p, _ := m["pattern"]
		path, _ := m["path"]
		return fmt.Sprintf("%v:%v", p, path)
	case "Edit":
		fp, _ := m["file_path"]
		old, _ := m["old_string"]
		return fmt.Sprintf("%v:%v", fp, old)
	}

	// Fallback: use full JSON
	return inputJSON
}

// 1. Duplicate calls: same tool_name + similar input, 3+ times within 60s window.
func detectDuplicateCalls(sessionID string, calls []store.WasteToolCall) []Finding {
	type entry struct {
		timestamps []int64
	}

	groups := make(map[string]*entry)
	var findings []Finding

	for _, c := range calls {
		key := c.ToolName + "|" + toolInputKey(c.ToolName, c.ToolInputJSON)
		e, ok := groups[key]
		if !ok {
			e = &entry{}
			groups[key] = e
		}
		e.timestamps = append(e.timestamps, c.StartedAt)

		// Check sliding window: count timestamps within 60s of the latest
		windowStart := c.StartedAt - 60_000
		count := 0
		for _, ts := range e.timestamps {
			if ts >= windowStart {
				count++
			}
		}
		if count == 3 {
			findings = append(findings, Finding{
				Type:      WasteDuplicateCalls,
				SessionID: sessionID,
				Summary:   fmt.Sprintf("Tool %q called 3+ times with similar input within 60s", c.ToolName),
				Details:   fmt.Sprintf("key: %s", toolInputKey(c.ToolName, c.ToolInputJSON)),
				Count:     count,
			})
		}
	}

	return findings
}

// 2. Excessive reads: same file Read 5+ times in one session.
func detectExcessiveReads(sessionID string, calls []store.WasteToolCall) []Finding {
	fileCounts := make(map[string]int)

	for _, c := range calls {
		if c.ToolName != "Read" {
			continue
		}
		fp := toolInputKey("Read", c.ToolInputJSON)
		fileCounts[fp]++
	}

	var findings []Finding
	for fp, count := range fileCounts {
		if count >= 5 {
			findings = append(findings, Finding{
				Type:      WasteExcessiveReads,
				SessionID: sessionID,
				Summary:   fmt.Sprintf("File %q read %d times", fp, count),
				Count:     count,
			})
		}
	}
	return findings
}

// 3. Failed retries: same tool fails consecutively 3+ times with similar input.
func detectFailedRetries(sessionID string, calls []store.WasteToolCall) []Finding {
	var findings []Finding
	var streak int
	var lastKey string

	for _, c := range calls {
		key := c.ToolName + "|" + toolInputKey(c.ToolName, c.ToolInputJSON)
		if c.Succeeded == 0 && key == lastKey {
			streak++
		} else if c.Succeeded == 0 {
			lastKey = key
			streak = 1
		} else {
			// Success resets
			lastKey = ""
			streak = 0
		}

		if streak == 3 {
			findings = append(findings, Finding{
				Type:      WasteFailedRetries,
				SessionID: sessionID,
				Summary:   fmt.Sprintf("Tool %q failed 3+ consecutive times with similar input", c.ToolName),
				Details:   fmt.Sprintf("key: %s", toolInputKey(c.ToolName, c.ToolInputJSON)),
				Count:     streak,
			})
		}
	}

	return findings
}

// editInput extracts file_path, old_string, new_string from an Edit tool_input_json.
type editInput struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

func parseEditInput(inputJSON string) *editInput {
	var e editInput
	if err := json.Unmarshal([]byte(inputJSON), &e); err != nil {
		return nil
	}
	if e.FilePath == "" {
		return nil
	}
	return &e
}

// 4. Edit revert: A→B→A pattern on the same file.
func detectEditReverts(sessionID string, calls []store.WasteToolCall) []Finding {
	// Collect Edit calls per file in order
	type fileEdits struct {
		edits []editInput
	}
	byFile := make(map[string]*fileEdits)

	for _, c := range calls {
		if c.ToolName != "Edit" {
			continue
		}
		e := parseEditInput(c.ToolInputJSON)
		if e == nil {
			continue
		}
		fe, ok := byFile[e.FilePath]
		if !ok {
			fe = &fileEdits{}
			byFile[e.FilePath] = fe
		}
		fe.edits = append(fe.edits, *e)
	}

	var findings []Finding
	for fp, fe := range byFile {
		for i := 2; i < len(fe.edits); i++ {
			// Check if edit[i].old_string == edit[i-2].new_string (A→B→A revert)
			if fe.edits[i].OldString == fe.edits[i-2].NewString && fe.edits[i].OldString != "" {
				findings = append(findings, Finding{
					Type:      WasteEditRevert,
					SessionID: sessionID,
					Summary:   fmt.Sprintf("Edit reverted on %q (A→B→A pattern)", fp),
				})
				break // One finding per file is enough
			}
		}
	}
	return findings
}

// 5. Zombie sessions: >30min but tool_calls<5 and prompts<3.
func detectZombieSessions(sessions []store.WasteSession) []Finding {
	var findings []Finding
	for _, s := range sessions {
		if s.TotalToolCalls < 5 && s.TotalPrompts < 3 {
			findings = append(findings, Finding{
				Type:      WasteZombieSession,
				SessionID: s.ID,
				Summary: fmt.Sprintf(
					"Session lasted %dm but only had %d prompts and %d tool calls",
					s.DurationMs/60_000, s.TotalPrompts, s.TotalToolCalls,
				),
			})
		}
	}
	return findings
}
