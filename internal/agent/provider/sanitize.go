package provider

import (
	"encoding/json"

	"go.uber.org/zap"
)

// sanitizeToolPairing ensures every assistant tool_use has a matching tool_result
// in the following messages. If a session was interrupted mid-tool-execution, the
// assistant message was logged but some tool results were not. Without this fix,
// the Anthropic API rejects the request with: "tool_use ids were found without
// tool_result blocks immediately after".
func SanitizeToolPairing(messages []Message) []Message {
	cleaned := make([]Message, len(messages))
	copy(cleaned, messages)

	for i := 0; i < len(cleaned); i++ {
		if cleaned[i].Role != "assistant" {
			continue
		}
		toolIDs := extractToolUseIDs(cleaned[i].Content)
		if len(toolIDs) == 0 {
			continue
		}

		// Collect all tool_result IDs from consecutive tool messages after this assistant.
		found := make(map[string]bool)
		for j := i + 1; j < len(cleaned) && cleaned[j].Role == "tool"; j++ {
			for _, id := range extractToolResultIDs(cleaned[j].Content) {
				found[id] = true
			}
		}

		// If all matched, skip. Otherwise strip orphaned tool_use blocks.
		allFound := true
		for _, id := range toolIDs {
			if !found[id] {
				allFound = false
				break
			}
		}
		if !allFound {
			zap.S().Warnw("stripping orphaned tool_use blocks",
				"message_index", i,
				"tool_use_ids", toolIDs,
				"found_results", found,
			)
			cleaned[i].Content = stripOrphanedToolUses(cleaned[i].Content, found)
		}
	}
	return cleaned
}

func extractToolUseIDs(content json.RawMessage) []string {
	var blocks []struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal(content, &blocks); err != nil {
		return nil
	}
	var ids []string
	for _, b := range blocks {
		if b.Type == "tool_use" && b.ID != "" {
			ids = append(ids, b.ID)
		}
	}
	return ids
}

func extractToolResultIDs(content json.RawMessage) []string {
	var blocks []struct {
		Type      string `json:"type"`
		ToolUseID string `json:"tool_use_id"`
	}
	if err := json.Unmarshal(content, &blocks); err != nil {
		return nil
	}
	var ids []string
	for _, b := range blocks {
		if b.Type == "tool_result" && b.ToolUseID != "" {
			ids = append(ids, b.ToolUseID)
		}
	}
	return ids
}

func stripOrphanedToolUses(content json.RawMessage, validIDs map[string]bool) json.RawMessage {
	var blocks []map[string]any
	if err := json.Unmarshal(content, &blocks); err != nil {
		return content
	}
	var cleaned []map[string]any
	for _, b := range blocks {
		if b["type"] == "tool_use" {
			id, _ := b["id"].(string)
			if !validIDs[id] {
				continue
			}
		}
		cleaned = append(cleaned, b)
	}
	result, _ := json.Marshal(cleaned)
	return json.RawMessage(result)
}
