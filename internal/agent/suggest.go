package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shenghuikevin/cc-track/internal/analysis"
)

const systemPrompt = `你是一个 Claude Code 使用效率专家。用户会给你一份 Claude Code session 的浪费检测报告，包含各种浪费模式（重复调用、过度读取、失败重试、Edit 反复、僵尸 session）。

请针对每条 finding 给出：
1. 简短的原因分析（为什么会出现这种浪费）
2. 具体可操作的改进建议
3. 严重程度（info / warning / critical）

用中文回答，保持简洁。每条建议不超过 3 句话。

输出格式：
---
### Finding N: [类型]
**原因**: ...
**建议**: ...
**严重程度**: info/warning/critical
---`

// Suggest uses an LLM to generate improvement suggestions for waste findings.
func Suggest(client *Client, report *analysis.WasteReport) (string, error) {
	if len(report.Findings) == 0 {
		return "没有检测到浪费模式，表现不错！", nil
	}

	// Build findings summary for the LLM
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("分析了 %d 个 session，发现 %d 个问题：\n\n", report.SessionsAnalyzed, len(report.Findings)))

	for i, f := range report.Findings {
		sb.WriteString(fmt.Sprintf("Finding %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  类型: %s\n", f.Type))
		sb.WriteString(fmt.Sprintf("  描述: %s\n", f.Summary))
		if f.Details != "" {
			sb.WriteString(fmt.Sprintf("  详情: %s\n", f.Details))
		}
		if f.Count > 0 {
			sb.WriteString(fmt.Sprintf("  次数: %d\n", f.Count))
		}
		sb.WriteString(fmt.Sprintf("  Session: %s\n\n", f.SessionID))
	}

	result, err := client.Chat(systemPrompt, sb.String())
	if err != nil {
		return "", fmt.Errorf("agent: suggest: %w", err)
	}
	return result, nil
}

// SuggestJSON returns suggestions as structured JSON (best-effort parse from LLM output).
func SuggestJSON(client *Client, report *analysis.WasteReport) (string, error) {
	if len(report.Findings) == 0 {
		result := map[string]interface{}{
			"suggestions": []interface{}{},
			"message":     "没有检测到浪费模式",
		}
		b, _ := json.MarshalIndent(result, "", "  ")
		return string(b), nil
	}

	text, err := Suggest(client, report)
	if err != nil {
		return "", err
	}

	// Wrap the LLM text output in a JSON structure
	result := map[string]interface{}{
		"sessions_analyzed": report.SessionsAnalyzed,
		"findings_count":    len(report.Findings),
		"suggestions":       text,
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}
