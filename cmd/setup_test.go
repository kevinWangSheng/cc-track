package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupTestHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	return tmp
}

func TestRunSetup(t *testing.T) {
	setupTestHome(t)

	err := runSetup()
	if err != nil {
		t.Fatalf("runSetup: %v", err)
	}

	raw, err := os.ReadFile(settingsPath())
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parse settings: %v", err)
	}

	hooks, ok := data["hooks"].(map[string]any)
	if !ok {
		t.Fatal("hooks not found in settings")
	}

	// Check all hook types are present with correct nested structure
	for _, event := range hookEvents {
		hookType := hookTypeForEvent(event)
		entries, ok := hooks[hookType].([]any)
		if !ok {
			t.Errorf("hook type %q not found", hookType)
			continue
		}
		found := false
		for _, e := range entries {
			m, ok := e.(map[string]any)
			if !ok || !isCCTrackHook(m) {
				continue
			}
			found = true
			// Verify nested hooks structure
			hooksArr, ok := m["hooks"].([]any)
			if !ok || len(hooksArr) == 0 {
				t.Errorf("hook %q: missing nested hooks array", hookType)
				continue
			}
			inner, ok := hooksArr[0].(map[string]any)
			if !ok {
				t.Errorf("hook %q: inner hook is not a map", hookType)
				continue
			}
			if inner["type"] != "command" {
				t.Errorf("hook %q: type = %v, want command", hookType, inner["type"])
			}
			if async, ok := inner["async"].(bool); !ok || !async {
				t.Errorf("hook %q: async not set to true", hookType)
			}
		}
		if !found {
			t.Errorf("cc-track hook not found in %q", hookType)
		}
	}
}

func TestRunSetupIdempotent(t *testing.T) {
	setupTestHome(t)

	_ = runSetup()
	_ = runSetup() // second call should be no-op

	raw, _ := os.ReadFile(settingsPath())
	var data map[string]any
	json.Unmarshal(raw, &data)

	hooks := data["hooks"].(map[string]any)
	// Each hook type should have exactly 1 entry
	for _, event := range hookEvents {
		hookType := hookTypeForEvent(event)
		entries := hooks[hookType].([]any)
		count := 0
		for _, e := range entries {
			if m, ok := e.(map[string]any); ok && isCCTrackHook(m) {
				count++
			}
		}
		if count != 1 {
			t.Errorf("hook %q: expected 1 cc-track entry, got %d", hookType, count)
		}
	}
}

func TestRunSetupPreservesExistingHooks(t *testing.T) {
	home := setupTestHome(t)

	// Write existing settings with a different hook
	existing := map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{
				map[string]any{
					"command": "notifier --start",
					"async":   true,
				},
			},
		},
	}
	raw, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), raw, 0o644)

	err := runSetup()
	if err != nil {
		t.Fatalf("runSetup: %v", err)
	}

	raw, _ = os.ReadFile(settingsPath())
	var data map[string]any
	json.Unmarshal(raw, &data)

	hooks := data["hooks"].(map[string]any)
	entries := hooks["SessionStart"].([]any)

	if len(entries) != 2 {
		t.Fatalf("SessionStart entries = %d, want 2", len(entries))
	}

	// First should be the existing notifier
	first := entries[0].(map[string]any)
	if first["command"] != "notifier --start" {
		t.Errorf("first hook command = %q, want 'notifier --start'", first["command"])
	}
}

func TestRunSetupRemove(t *testing.T) {
	setupTestHome(t)

	_ = runSetup()
	err := runSetupRemove()
	if err != nil {
		t.Fatalf("runSetupRemove: %v", err)
	}

	raw, _ := os.ReadFile(settingsPath())
	var data map[string]any
	json.Unmarshal(raw, &data)

	hooks, ok := data["hooks"].(map[string]any)
	if ok {
		for _, event := range hookEvents {
			hookType := hookTypeForEvent(event)
			if entries, ok := hooks[hookType]; ok {
				arr := entries.([]any)
				for _, e := range arr {
					if m, ok := e.(map[string]any); ok && isCCTrackHook(m) {
						t.Errorf("cc-track hook still present in %q after remove", hookType)
					}
				}
			}
		}
	}
}
