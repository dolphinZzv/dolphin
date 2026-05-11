package agent

import (
	"fmt"
	"strconv"
	"strings"
)

// SegmentCompressor implements strategy A: each compression round generates an L1
// segment from the newest batch of raw messages. When any level accumulates more
// than SegmentMergeLimit segments, they are recursively merged into the next level.
type SegmentCompressor struct {
	MergeLimit int
}

// NewSegmentCompressor creates a SegmentCompressor with the given merge limit.
func NewSegmentCompressor(mergeLimit int) *SegmentCompressor {
	if mergeLimit <= 0 {
		mergeLimit = 100
	}
	return &SegmentCompressor{MergeLimit: mergeLimit}
}

func (s *SegmentCompressor) Compress(messages []Message, maxTokens int) ([]Message, *CompressReport) {
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

	// Collect the raw messages being dropped (non-summary, non-synthetic).
	var droppedRaw []Message
	i := 0
	for i < keepStart {
		if messages[i].Role != "user" {
			i++
			continue
		}
		end := i + 1
		for end < keepStart && messages[end].Role != "user" {
			end++
		}
		if end > keepStart {
			break
		}
		droppedRaw = append(droppedRaw, messages[i:end]...)
		i = end // i stays; messages shifted below
	}

	if len(droppedRaw) == 0 {
		return nil, nil
	}

	// Build result: existing summary segments (before keepStart) + new L1 from dropped raw + remaining
	var result []Message

	// Collect existing summary segments (messages with "[LX 摘要" prefix before keepStart)
	var segments []*segmentSummary
	for j := 0; j < keepStart; {
		if messages[j].Role != "user" {
			j++
			continue
		}
		end := j + 1
		for end < keepStart && messages[end].Role != "user" {
			end++
		}
		if end > keepStart {
			break
		}
		if seg := parseSegment(messages[j:end]); seg != nil {
			segments = append(segments, seg)
		}
		j = end
	}

	// Create new L1 segment from dropped raw messages
	newSeg := &segmentSummary{
		Content:      summarizeRawMessages(droppedRaw),
		Level:        1,
		CoveredCount: len(droppedRaw),
	}
	segments = append(segments, newSeg)

	// Recursive merge: while any level has > MergeLimit segments, merge them up
	segments = s.mergeLevels(segments)

	// Build output: summary segments + remaining messages
	for _, seg := range segments {
		result = append(result, seg.toMessage())
	}
	droppedCount := keepStart
	result = append(result, messages[keepStart:]...)

	// Estimate tokens saved
	tokensSaved := 0
	for j := 0; j < droppedCount && j < len(messages); j++ {
		tokensSaved += estimateTokens(string(messages[j].Content))
	}
	for _, seg := range segments {
		tokensSaved -= estimateTokens(string(seg.toMessage().Content))
	}

	return result, &CompressReport{
		DroppedCount: droppedCount,
		TokensSaved:  max(tokensSaved, 0),
		NewLevel:     1,
	}
}

// mergeLevels recursively merges segments at any level that exceed the limit.
func (s *SegmentCompressor) mergeLevels(segments []*segmentSummary) []*segmentSummary {
	for {
		// Count segments per level
		levelCounts := make(map[int][]int) // level → indices
		for idx, seg := range segments {
			levelCounts[seg.Level] = append(levelCounts[seg.Level], idx)
		}

		// Find the lowest level that exceeds the limit
		merged := false
		for level := 1; ; level++ {
			indices, ok := levelCounts[level]
			if !ok {
				break // no more levels
			}
			if len(indices) > s.MergeLimit {
				// Merge all segments at this level into one at level+1
				mergedContent := ""
				totalCovered := 0
				for _, idx := range indices {
					mergedContent += segments[idx].Content + "\n"
					totalCovered += segments[idx].CoveredCount
				}

				mergedSeg := &segmentSummary{
					Content:      "[合并 " + strconv.Itoa(len(indices)) + " 段 L" + strconv.Itoa(level) + "] " + mergedContent,
					Level:        level + 1,
					CoveredCount: totalCovered,
				}

				// Remove old segments, insert merged segment
				// Indices are in ascending order; replace first, remove rest
				firstIdx := indices[0]
				segments[firstIdx] = mergedSeg
				// Remove remaining indices (in reverse to preserve positions)
				for k := len(indices) - 1; k >= 1; k-- {
					segments = append(segments[:indices[k]], segments[indices[k]+1:]...)
				}
				merged = true
				break
			}
		}
		if !merged {
			break
		}
	}
	return segments
}

// parseSegment checks if messages form a summary segment and returns it.
func parseSegment(messages []Message) *segmentSummary {
	if len(messages) == 0 || messages[0].Role != "user" {
		return nil
	}
	text := extractText(messages[0].Content)
	// Match "[LX 摘要, 覆盖 N 组] content"
	if !strings.HasPrefix(text, "[L") {
		return nil
	}
	// Simple heuristic: check for the summary marker
	idx := strings.Index(text, " 摘要")
	if idx < 0 {
		return nil
	}
	levelStr := text[2:idx]
	level := 0
	if _, err := fmt.Sscanf(levelStr, "%d", &level); err != nil || level < 1 {
		return nil
	}

	// Extract covered count
	covered := 0
	prefixEnd := strings.Index(text, "] ")
	if prefixEnd < 0 {
		return nil
	}
	prefix := text[1:prefixEnd]
	if covIdx := strings.Index(prefix, "覆盖 "); covIdx >= 0 {
		covPart := prefix[covIdx+len("覆盖 "):]
		if spaceIdx := strings.Index(covPart, " 组"); spaceIdx >= 0 {
			fmt.Sscanf(covPart[:spaceIdx], "%d", &covered)
		}
	}

	content := text[prefixEnd+2:]
	return &segmentSummary{
		Content:      content,
		Level:        level,
		CoveredCount: max(covered, 1),
	}
}

// summarizeRawMessages returns a placeholder summary of raw messages.
// When LLM integration is added, this calls the provider.
func summarizeRawMessages(messages []Message) string {
	if len(messages) == 0 {
		return ""
	}
	texts := make([]string, 0, len(messages))
	for _, m := range messages {
		t := extractText(m.Content)
		if t != "" {
			texts = append(texts, t)
		}
	}
	return strings.Join(texts, " | ")
}
