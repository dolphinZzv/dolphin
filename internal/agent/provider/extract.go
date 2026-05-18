package provider

import (
	"encoding/json"
	"strings"
)

func ExtractText(content json.RawMessage) string {
	var blocks []map[string]any
	if err := json.Unmarshal(content, &blocks); err != nil {
		return ""
	}
	var buf strings.Builder
	for _, b := range blocks {
		if t, ok := b["text"].(string); ok {
			buf.WriteString(t)
		}
	}
	return buf.String()
}


func EstimateTokens(content string) int {
	cjk := 0
	for _, r := range content {
		if r >= 0x2E80 && r <= 0x9FFF || r >= 0xF900 && r <= 0xFAFF || r >= 0xFE30 && r <= 0xFE4F {
			cjk++
		}
	}
	// CJK: ~1 token each (conservative). Non-CJK: bytes / 3.5.
	nonCJKTokens := 0
	if nonCJKBytes := len(content) - cjk*3; nonCJKBytes > 0 {
		nonCJKTokens = nonCJKBytes * 10 / 35
	}
	return cjk + nonCJKTokens
}


func ExtractToolCallID(content json.RawMessage) string {
	var blocks []map[string]any
	if err := json.Unmarshal(content, &blocks); err != nil {
		return ""
	}
	for _, b := range blocks {
		if id, ok := b["tool_use_id"].(string); ok {
			return id
		}
	}
	return ""
}

func ExtractToolResult(content json.RawMessage) string {
	var blocks []map[string]any
	if err := json.Unmarshal(content, &blocks); err != nil {
		return string(content)
	}
	for _, b := range blocks {
		if b["type"] == "tool_result" {
			switch v := b["content"].(type) {
			case string:
				return v
			case []any:
				// Anthropic format: [{type: "text", text: "..."}]
				for _, item := range v {
					if m, ok := item.(map[string]any); ok {
						if t, ok := m["text"].(string); ok {
							return t
						}
					}
				}
			}
		}
	}
	return string(content)
}
