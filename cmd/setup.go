package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	setupCheck  bool
	setupRemove bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Register/check/remove Claude Code hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		if setupCheck {
			return runSetupCheck()
		}
		if setupRemove {
			return runSetupRemove()
		}
		return runSetup()
	},
}

func init() {
	setupCmd.Flags().BoolVar(&setupCheck, "check", false, "check if hooks are registered")
	setupCmd.Flags().BoolVar(&setupRemove, "remove", false, "remove hooks")
	rootCmd.AddCommand(setupCmd)
}

var hookEvents = []string{
	"SessionStart",
	"UserPromptSubmit",
	"PreToolUse",
	"PostToolUse",
	"PostToolUseFailure",
	"Stop",
	"StopFailure",
	"SubagentStop",
	"SessionEnd",
}

func settingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func ccTrackBinaryPath() string {
	exe, err := exec.LookPath("cc-track")
	if err != nil {
		exe, _ = os.Executable()
	}
	return exe
}

type settingsFile struct {
	data map[string]any
}

func loadSettings() (*settingsFile, error) {
	path := settingsPath()
	sf := &settingsFile{data: make(map[string]any)}

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return sf, nil
	}
	if err != nil {
		return nil, fmt.Errorf("setup: read settings: %w", err)
	}

	if err := json.Unmarshal(raw, &sf.data); err != nil {
		return nil, fmt.Errorf("setup: parse settings: %w", err)
	}
	return sf, nil
}

func (sf *settingsFile) save() error {
	path := settingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("setup: create settings dir: %w", err)
	}

	raw, err := json.MarshalIndent(sf.data, "", "  ")
	if err != nil {
		return fmt.Errorf("setup: marshal settings: %w", err)
	}
	raw = append(raw, '\n')

	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("setup: write settings: %w", err)
	}
	return nil
}

func (sf *settingsFile) getHooks() map[string]any {
	hooks, ok := sf.data["hooks"]
	if !ok {
		return nil
	}
	m, ok := hooks.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func (sf *settingsFile) ensureHooks() map[string]any {
	hooks := sf.getHooks()
	if hooks == nil {
		hooks = make(map[string]any)
		sf.data["hooks"] = hooks
	}
	return hooks
}

func ccTrackCommand() string {
	return ccTrackBinaryPath() + " hook " + hookMarker
}

const hookMarker = "# cc-track-hook"

// isCCTrackHook checks if an event-level entry contains a cc-track hook.
// The format is: { "hooks": [{ "type": "command", "command": "... # cc-track-hook", ... }] }
func isCCTrackHook(entry map[string]any) bool {
	// Check nested hooks array (correct format)
	if hooksArr, ok := entry["hooks"].([]any); ok {
		for _, h := range hooksArr {
			if m, ok := h.(map[string]any); ok {
				cmd, _ := m["command"].(string)
				if strings.Contains(cmd, hookMarker) {
					return true
				}
			}
		}
	}
	// Also check flat format for backwards compat detection
	cmd, _ := entry["command"].(string)
	return strings.Contains(cmd, hookMarker)
}

func runSetup() error {
	sf, err := loadSettings()
	if err != nil {
		return err
	}

	hooks := sf.ensureHooks()
	command := ccTrackCommand()
	added := 0

	for _, event := range hookEvents {
		hookType := hookTypeForEvent(event)
		entries := getHookEntries(hooks, hookType)

		found := false
		for _, entry := range entries {
			if m, ok := entry.(map[string]any); ok && isCCTrackHook(m) {
				found = true
				break
			}
		}

		if !found {
			newEntry := map[string]any{
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": command,
						"async":   true,
					},
				},
			}
			entries = append(entries, newEntry)
			hooks[hookType] = entries
			added++
		}
	}

	envChanged := ensureEnvVar(sf, "CLAUDE_CODE_SESSIONEND_HOOKS_TIMEOUT_MS", "5000")

	if added == 0 && !envChanged {
		fmt.Println("hooks already registered")
		return nil
	}

	if err := sf.save(); err != nil {
		return err
	}
	if added > 0 {
		fmt.Printf("registered %d hooks\n", added)
	}
	if envChanged {
		fmt.Println("set CLAUDE_CODE_SESSIONEND_HOOKS_TIMEOUT_MS=5000")
	}
	return nil
}

func runSetupCheck() error {
	sf, err := loadSettings()
	if err != nil {
		return err
	}

	hooks := sf.getHooks()
	if hooks == nil {
		fmt.Println("no hooks registered")
		return nil
	}

	registered := 0
	for _, event := range hookEvents {
		hookType := hookTypeForEvent(event)
		entries := getHookEntries(hooks, hookType)
		for _, entry := range entries {
			if m, ok := entry.(map[string]any); ok && isCCTrackHook(m) {
				registered++
				break
			}
		}
	}

	fmt.Printf("%d/%d hooks registered\n", registered, len(hookEvents))
	if registered < len(hookEvents) {
		fmt.Println("run 'cc-track setup' to register missing hooks")
	}
	return nil
}

func runSetupRemove() error {
	sf, err := loadSettings()
	if err != nil {
		return err
	}

	hooks := sf.getHooks()
	if hooks == nil {
		fmt.Println("no hooks to remove")
		return nil
	}

	removed := 0
	for _, event := range hookEvents {
		hookType := hookTypeForEvent(event)
		entries := getHookEntries(hooks, hookType)
		var filtered []any
		for _, entry := range entries {
			if m, ok := entry.(map[string]any); ok && isCCTrackHook(m) {
				removed++
				continue
			}
			filtered = append(filtered, entry)
		}
		if len(filtered) == 0 {
			delete(hooks, hookType)
		} else {
			hooks[hookType] = filtered
		}
	}

	if removed == 0 {
		fmt.Println("no cc-track hooks found")
		return nil
	}

	if err := sf.save(); err != nil {
		return err
	}
	fmt.Printf("removed %d hooks\n", removed)
	return nil
}

func hookTypeForEvent(event string) string {
	switch event {
	case "SessionStart", "UserPromptSubmit", "SessionEnd":
		return event
	case "PreToolUse":
		return "PreToolUse"
	case "PostToolUse":
		return "PostToolUse"
	case "PostToolUseFailure":
		return "PostToolUseFailure"
	case "Stop":
		return "Stop"
	case "StopFailure":
		return "StopFailure"
	case "SubagentStop":
		return "SubagentStop"
	default:
		return event
	}
}

// ensureEnvVar ensures a key=value exists in settings.env. Returns true if changed.
func ensureEnvVar(sf *settingsFile, key, value string) bool {
	env, ok := sf.data["env"].(map[string]any)
	if !ok {
		env = make(map[string]any)
		sf.data["env"] = env
	}
	if existing, ok := env[key].(string); ok && existing == value {
		return false
	}
	env[key] = value
	return true
}

func getHookEntries(hooks map[string]any, hookType string) []any {
	v, ok := hooks[hookType]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	return arr
}
