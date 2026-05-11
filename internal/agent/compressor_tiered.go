package agent

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
)

// TieredCompressor implements strategy B: a three-tier cache.
// [L2 远历史] [L1 中期] [原文 最近N轮]
type TieredCompressor struct {
	provider Provider
	rawKeep  int // number of recent user+assistant pairs to keep raw (default 3)
}

// NewTieredCompressor creates a TieredCompressor with an LLM provider.
func NewTieredCompressor(provider Provider) *TieredCompressor {
	return &TieredCompressor{provider: provider, rawKeep: 3}
}

func (t *TieredCompressor) Compress(messages []Message, maxTokens int) ([]Message, *CompressReport) {
	if maxTokens <= 0 || len(messages) <= 6 {
		return nil, nil
	}

	est := 0
	for _, m := range messages {
		est += estimateTokens(string(m.Content))
	}
	threshold := int(float64(maxTokens) * 0.7)
	if est <= threshold {
		return nil, nil
	}

	// Find the last user message position
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

	// Count raw pairs to keep from the end
	rawPairs := 0
	rawStart := keepStart
	for j := keepStart; j >= 0 && rawPairs < t.rawKeep; {
		if messages[j].Role == "user" {
			rawPairs++
		}
		if j > 0 {
			j--
		} else {
			break
		}
		// walk back to find the user message that starts this pair
		for j >= 0 && messages[j].Role != "user" {
			j--
		}
		if rawPairs < t.rawKeep && j >= 0 {
			rawStart = j
		}
	}

	// Split: [old...rawStart-1] [rawStart...keepStart-1] [keepStart...]
	// If rawStart==keepStart, all messages before keepStart are candidates for compression
	var oldMessages []Message
	var l1Messages []Message
	if rawStart < keepStart {
		// Split the middle section: oldest part → L2, middle → L1
		midStart := rawStart
		if keepStart-rawStart > 6 {
			midStart = rawStart + (keepStart-rawStart)/2
		}
		oldMessages = messages[rawStart:midStart]
		l1Messages = messages[midStart:keepStart]
	} else {
		oldMessages = messages[:keepStart]
	}

	var result []Message
	tokensSaved := 0
	droppedCount := 0

	// Generate L2 summary from oldest messages
	if len(oldMessages) > 0 {
		l2Summary := t.summarize(oldMessages, 2)
		result = append(result, Message{Role: "user", Content: TextContent(l2Summary)})
		droppedCount += len(oldMessages)
		for _, m := range oldMessages {
			tokensSaved += estimateTokens(string(m.Content))
		}
		tokensSaved -= estimateTokens(l2Summary)
	}

	// Generate L1 summaries from middle messages
	if len(l1Messages) > 0 {
		l1Summary := t.summarize(l1Messages, 1)
		result = append(result, Message{Role: "user", Content: TextContent(l1Summary)})
		droppedCount += len(l1Messages)
		for _, m := range l1Messages {
			tokensSaved += estimateTokens(string(m.Content))
		}
		tokensSaved -= estimateTokens(l1Summary)
	}

	result = append(result, messages[keepStart:]...)

	if droppedCount == 0 {
		return nil, nil
	}

	return result, &CompressReport{
		DroppedCount: droppedCount,
		TokensSaved:  max(tokensSaved, 0),
		NewLevel:     2,
	}
}

// summarize calls the LLM to generate a summary of the given messages.
// level is the summary tier (1 or 2).
func (t *TieredCompressor) summarize(messages []Message, level int) string {
	texts := make([]string, 0, len(messages))
	for _, m := range messages {
		txt := extractText(m.Content)
		if txt != "" {
			texts = append(texts, m.Role+": "+txt)
		}
	}

	if t.provider == nil {
		return "[L" + itoa(level) + " 摘要] " + strings.Join(texts, " | ")
	}

	systemPrompt := "你是一个对话摘要助手。请用1-2句话简要摘要以下对话内容，保留关键信息。只输出摘要文本，不要加前缀或标记。"
	userContent := "请摘要以下对话：\n" + strings.Join(texts, "\n")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := t.provider.Complete(ctx, ProviderRequest{
		Messages:  []Message{{Role: "user", Content: TextContent(userContent)}},
		System:    systemPrompt,
		MaxTokens: 300,
	})
	if err != nil {
		zap.S().Debugw("tiered compressor: LLM summary failed, using concatenation", "error", err)
		return "[L" + itoa(level) + " 摘要] " + strings.Join(texts, " | ")
	}

	summary := extractText(resp.Content)
	if summary == "" {
		return "[L" + itoa(level) + " 摘要] " + strings.Join(texts, " | ")
	}
	return "[L" + itoa(level) + " 摘要] " + summary
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}
