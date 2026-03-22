package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Simulate a transcript with two unique assistant messages and one duplicate (streaming chunk).
	lines := []string{
		`{"type":"user","message":{"role":"user","content":"hello"}}`,
		`{"type":"assistant","message":{"id":"msg_001","role":"assistant","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":100,"output_tokens":20,"cache_creation_input_tokens":50,"cache_read_input_tokens":200}}}`,
		`{"type":"assistant","message":{"id":"msg_001","role":"assistant","content":[{"type":"text","text":"hi there"}],"usage":{"input_tokens":100,"output_tokens":30,"cache_creation_input_tokens":50,"cache_read_input_tokens":200}}}`,
		`{"type":"assistant","message":{"id":"msg_002","role":"assistant","content":[{"type":"text","text":"done"}],"usage":{"input_tokens":150,"output_tokens":40,"cache_creation_input_tokens":0,"cache_read_input_tokens":300}}}`,
	}

	var content []byte
	for _, l := range lines {
		content = append(content, []byte(l+"\n")...)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	usage, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	// msg_001 should use the LAST occurrence (output_tokens=30), msg_002 has output_tokens=40
	if usage.InputTokens != 250 { // 100 + 150
		t.Errorf("InputTokens = %d, want 250", usage.InputTokens)
	}
	if usage.OutputTokens != 70 { // 30 + 40
		t.Errorf("OutputTokens = %d, want 70", usage.OutputTokens)
	}
	if usage.CacheCreationInputTokens != 50 { // 50 + 0
		t.Errorf("CacheCreationInputTokens = %d, want 50", usage.CacheCreationInputTokens)
	}
	if usage.CacheReadInputTokens != 500 { // 200 + 300
		t.Errorf("CacheReadInputTokens = %d, want 500", usage.CacheReadInputTokens)
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := ParseFile("/nonexistent/path.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	os.WriteFile(path, []byte{}, 0o644)

	usage, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if usage.InputTokens != 0 || usage.OutputTokens != 0 {
		t.Error("expected zero tokens for empty file")
	}
}

func TestParseFileMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")

	content := "{invalid json}\n" +
		`{"type":"assistant","message":{"id":"msg_003","role":"assistant","content":[],"usage":{"input_tokens":50,"output_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	os.WriteFile(path, []byte(content), 0o644)

	usage, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	// Should skip the bad line and parse the good one
	if usage.InputTokens != 50 {
		t.Errorf("InputTokens = %d, want 50", usage.InputTokens)
	}
}
