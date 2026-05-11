package agent

import (
	"fmt"
	"strings"
)

// CompressReport holds statistics about a compression operation.
type CompressReport struct {
	DroppedCount int // messages dropped
	TokensSaved  int // estimated tokens freed
	NewLevel     int // summary level generated (0 = pure drop, no summary)
}

// Compressor compresses message history when approaching context limits.
// Returns the compressed message list and a report, or (nil, nil) if no
// compression was needed.
type Compressor interface {
	Compress(messages []Message, maxTokens int) ([]Message, *CompressReport)
}

// DropCompressor drops old user+assistant turn groups without summarization.
// This is the default strategy — identical to the pre-interface behavior.
type DropCompressor struct{}

func (d *DropCompressor) Compress(messages []Message, maxTokens int) ([]Message, *CompressReport) {
	if maxTokens <= 0 {
		return nil, nil
	}

	est := 0
	for _, m := range messages {
		est += estimateTokens(string(m.Content))
		if m.Role == "assistant" {
			est += 20
		}
	}

	threshold := int(float64(maxTokens) * 0.7)
	if est <= threshold {
		return nil, nil
	}

	if len(messages) <= 6 {
		return nil, nil
	}

	// Find the oldest message we must keep: the last user message and everything after it.
	keepStart := len(messages)
	for j := len(messages) - 1; j >= 0; j-- {
		if messages[j].Role == "user" {
			keepStart = j
			break
		}
	}
	if keepStart == len(messages) && len(messages) > 2 {
		keepStart = len(messages) - 2
	}

	// Walk from the front, dropping complete user+response turn groups.
	result := make([]Message, len(messages))
	copy(result, messages)
	dropped := 0
	for i := 0; i < keepStart; {
		if result[i].Role != "user" {
			i++
			continue
		}
		end := i + 1
		for end < keepStart && result[end].Role != "user" {
			end++
		}
		if end > keepStart {
			break
		}
		for j := i; j < end; j++ {
			est -= estimateTokens(string(result[j].Content))
		}
		dropped += end - i
		result = append(result[:i], result[end:]...)
		keepStart -= (end - i)
	}

	if dropped == 0 {
		return nil, nil
	}

	tokensSaved := 0
	for j := 0; j < dropped && j < len(messages); j++ {
		tokensSaved += estimateTokens(string(messages[j].Content))
	}

	return result, &CompressReport{
		DroppedCount: dropped,
		TokensSaved:  tokensSaved,
		NewLevel:     0,
	}
}

// segmentSummary holds a summary segment with its level.
type segmentSummary struct {
	Content      string // the summary text
	Level        int    // 1 = original summary, 2 = merged, etc.
	CoveredCount int    // how many original message groups this covers
}

// extractSummarySegments returns all summary segments found in the message list.
func extractSummarySegments(messages []Message) []segmentSummary {
	var out []segmentSummary
	for i := 0; i < len(messages); {
		if messages[i].Role != "user" {
			i++
			continue
		}
		end := i + 1
		for end < len(messages) && messages[end].Role != "user" {
			end++
		}
		if seg := parseSegment(messages[i:end]); seg != nil {
			out = append(out, *seg)
		}
		i = end
	}
	return out
}

// toMessage returns the summary as a synthetic user message.
func (s *segmentSummary) toMessage() Message {
	var sb strings.Builder
	sb.WriteString("[L")
	sb.WriteString(fmt.Sprint(s.Level))
	sb.WriteString(" 摘要, 覆盖 ")
	sb.WriteString(fmt.Sprint(s.CoveredCount))
	sb.WriteString(" 组] ")
	sb.WriteString(s.Content)
	return Message{Role: "user", Content: TextContent(sb.String())}
}
