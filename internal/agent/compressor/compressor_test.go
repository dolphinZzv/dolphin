package compressor

import (
	"encoding/json"
	"strings"
	"testing"

	"dolphin/internal/agent/provider"
)

func TestCompressPreamble(t *testing.T) {
	msgs := []provider.Message{
		{Role: "user", Content: provider.TextContent("hi")},
		{Role: "assistant", Content: provider.TextContent("hello")},
	}

	result := compressPreamble(msgs, 100000)
	if result.CanDrop {
		t.Error("expected no compression for small context")
	}
}

func TestDropCompressorBelowThreshold(t *testing.T) {
	msgs := []provider.Message{
		{Role: "user", Content: provider.TextContent("hi")},
		{Role: "assistant", Content: provider.TextContent("hello")},
	}

	comp := &DropCompressor{}
	compressed, report := comp.Compress(msgs, 100000)
	if report != nil {
		t.Errorf("expected no compression below threshold, got report: %+v", report)
	}
	if compressed != nil {
		t.Error("expected nil compressed result")
	}
}

func TestDropCompressorDropsOldMessages(t *testing.T) {
	// Need 7+ messages to pass the len <= 6 guard in compressPreamble
	msgs := make([]provider.Message, 8)
	for i := 0; i < 8; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		msgs[i] = provider.Message{Role: role, Content: provider.TextContent("msg")}
	}

	comp := &DropCompressor{}
	compressed, report := comp.Compress(msgs, 100)
	if report == nil {
		t.Fatal("expected compression")
	}
	if report.DroppedCount == 0 {
		t.Error("expected at least one dropped message")
	}
	if compressed == nil {
		t.Fatal("expected compressed result")
	}
}

func TestDropCompressorInsufficientMessages(t *testing.T) {
	msgs := []provider.Message{
		{Role: "user", Content: json.RawMessage(`[{"type":"text","text":"hello"}]`)},
	}

	comp := &DropCompressor{}
	compressed, report := comp.Compress(msgs, 10)
	if report != nil {
		t.Error("expected no compression with only 1 message")
	}
	if compressed != nil {
		t.Error("expected nil result")
	}
}

func TestDropCompressorWithCJKContent(t *testing.T) {
	text := strings.Repeat("你好世界", 1000)
	msgs := make([]provider.Message, 8)
	for i := 0; i < 8; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		if i%2 == 0 {
			msgs[i] = provider.Message{Role: role, Content: provider.TextContent(text)}
		} else {
			msgs[i] = provider.Message{Role: role, Content: provider.TextContent("ok")}
		}
	}

	comp := &DropCompressor{}
	compressed, report := comp.Compress(msgs, 100)
	if report == nil {
		t.Fatal("expected compression for large CJK content")
	}
	if report.DroppedCount == 0 {
		t.Error("expected messages to be dropped")
	}
	if compressed == nil {
		t.Fatal("expected compressed result")
	}
}

func TestDropCompressorOnlyDropsCompleteTurns(t *testing.T) {
	msgs := make([]provider.Message, 9)
	for i := 0; i < 9; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		msgs[i] = provider.Message{Role: role, Content: provider.TextContent("msg")}
	}

	comp := &DropCompressor{}
	compressed, report := comp.Compress(msgs, 100)
	if report == nil {
		t.Fatal("expected compression")
	}
	if compressed == nil {
		t.Fatal("expected compressed result")
	}
	// Last user message "msg" should remain since it has no matching assistant message
	lastContent := string(compressed[len(compressed)-1].Content)
	if !strings.Contains(lastContent, "msg") {
		t.Error("expected last user message to be preserved")
	}
}
