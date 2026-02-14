package chat

import (
	"strings"
	"testing"

	"github.com/bilbilaki/ai2go/internal/api"
)

func TestBuildSummarizableTranscriptIncludesToolCalls(t *testing.T) {
	msgs := []api.Message{
		{Role: "user", Content: "Need retry logic"},
		{
			Role:    "assistant",
			Content: "I will inspect files.",
			ToolCalls: []api.ToolCall{
				{Function: api.FunctionCall{Name: "read_file"}},
				{Function: api.FunctionCall{Name: "patch_file"}},
			},
		},
	}

	got := buildSummarizableTranscript(msgs)
	if !strings.Contains(got, "USER: Need retry logic") {
		t.Fatalf("missing user content in transcript: %s", got)
	}
	if !strings.Contains(got, "ASSISTANT_TOOL_CALLS: read_file, patch_file") {
		t.Fatalf("missing tool calls in transcript: %s", got)
	}
}

func TestChunkByCharsSplitsLargeInput(t *testing.T) {
	content := strings.Repeat("x", summaryChunkChars+200)
	chunks := chunkByChars(content, summaryChunkChars)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for i, c := range chunks {
		if len(c) > summaryChunkChars {
			t.Fatalf("chunk %d exceeds limit: %d", i, len(c))
		}
	}
}
