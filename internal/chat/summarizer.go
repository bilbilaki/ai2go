package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
)

const (
	summaryChunkChars      = 12000
	maxSummaryMessageChars = 4000
	summaryMergeGroupSize  = 6
)

type SummaryResult struct {
	Summary string
	Chunks  int
	Passes  int
}

func SummarizeHistoryMultiPass(ctx context.Context, history *History, client *api.Client, model string) (SummaryResult, error) {
	if history == nil {
		return SummaryResult{}, fmt.Errorf("history is required")
	}
	if client == nil {
		return SummaryResult{}, fmt.Errorf("api client is required")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return SummaryResult{}, fmt.Errorf("model is required")
	}

	transcript := buildSummarizableTranscript(history.GetMessages())
	chunks := chunkByChars(transcript, summaryChunkChars)
	if len(chunks) == 0 {
		return SummaryResult{}, fmt.Errorf("no content available to summarize")
	}

	partialSummaries := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		msgs := []api.Message{
			{Role: "system", Content: "You are a conversation compression assistant. Keep essential context, decisions, constraints, and unresolved tasks."},
			{
				Role: "user",
				Content: fmt.Sprintf(
					"Chunk %d/%d. Summarize this section.\n"+
						"Requirements:\n"+
						"- Keep user goals, decisions, constraints, file paths, errors, and unfinished work.\n"+
						"- Keep important numeric values and limits.\n"+
						"- Remove filler and duplicated content.\n"+
						"- Output compact bullet points.\n\n%s",
					i+1, len(chunks), chunk,
				),
			},
		}

		resp, err := client.RunCompletionOnce(ctx, msgs, nil, model)
		if err != nil {
			return SummaryResult{}, fmt.Errorf("chunk %d summarize failed: %w", i+1, err)
		}
		partial := strings.TrimSpace(resp.Content)
		if partial == "" {
			partial = fmt.Sprintf("- Chunk %d had no usable summary output.", i+1)
		}
		partialSummaries = append(partialSummaries, partial)
	}

	passes := 1
	summaries := partialSummaries
	for len(summaries) > 1 {
		next := make([]string, 0, (len(summaries)+summaryMergeGroupSize-1)/summaryMergeGroupSize)
		for i := 0; i < len(summaries); i += summaryMergeGroupSize {
			end := i + summaryMergeGroupSize
			if end > len(summaries) {
				end = len(summaries)
			}
			merged, err := summarizeGroup(ctx, client, model, summaries[i:end])
			if err != nil {
				return SummaryResult{}, err
			}
			next = append(next, merged)
		}
		summaries = next
		passes++
	}

	// Force a refinement pass so summarization always uses at least two model calls.
	if len(chunks) == 1 {
		refined, err := summarizeGroup(ctx, client, model, summaries)
		if err != nil {
			return SummaryResult{}, err
		}
		summaries[0] = refined
		passes++
	}

	finalSummary := strings.TrimSpace(summaries[0])
	if finalSummary == "" {
		return SummaryResult{}, fmt.Errorf("summary pipeline returned empty content")
	}

	return SummaryResult{
		Summary: finalSummary,
		Chunks:  len(chunks),
		Passes:  passes,
	}, nil
}

func summarizeGroup(ctx context.Context, client *api.Client, model string, summaries []string) (string, error) {
	finalPrompt := "Combine these summaries into one thread memory.\n" +
		"Output format:\n" +
		"1) Goals\n2) Decisions/Config\n3) Code Changes\n4) Open Issues\n5) Next Actions\n\n" +
		strings.Join(summaries, "\n\n---\n\n")
	finalMsgs := []api.Message{
		{Role: "system", Content: "You produce compact, high-recall thread memory for future turns."},
		{Role: "user", Content: finalPrompt},
	}
	finalResp, err := client.RunCompletionOnce(ctx, finalMsgs, nil, model)
	if err != nil {
		return "", fmt.Errorf("summary merge failed: %w", err)
	}
	finalSummary := strings.TrimSpace(finalResp.Content)
	if finalSummary == "" {
		return "", fmt.Errorf("summary merge returned empty content")
	}
	return finalSummary, nil
}

func buildSummarizableTranscript(messages []api.Message) string {
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		role := strings.ToUpper(strings.TrimSpace(msg.Role))
		if role == "" {
			role = "UNKNOWN"
		}

		content := compactWhitespace(msg.Content)
		if content != "" {
			if len([]rune(content)) > maxSummaryMessageChars {
				runes := []rune(content)
				content = string(runes[:maxSummaryMessageChars]) + " ...[truncated]"
			}
			lines = append(lines, fmt.Sprintf("%s: %s", role, content))
		}

		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			tools := make([]string, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				name := strings.TrimSpace(tc.Function.Name)
				if name != "" {
					tools = append(tools, name)
				}
			}
			if len(tools) > 0 {
				lines = append(lines, fmt.Sprintf("ASSISTANT_TOOL_CALLS: %s", strings.Join(tools, ", ")))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func chunkByChars(content string, maxChars int) []string {
	content = strings.TrimSpace(content)
	if content == "" || maxChars <= 0 {
		return nil
	}

	src := strings.Split(content, "\n")
	out := make([]string, 0, len(src)/8+1)
	var b strings.Builder

	flush := func() {
		chunk := strings.TrimSpace(b.String())
		if chunk != "" {
			out = append(out, chunk)
		}
		b.Reset()
	}

	for _, line := range src {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		addition := line
		if b.Len() > 0 {
			addition = "\n" + line
		}

		if b.Len()+len(addition) <= maxChars {
			b.WriteString(addition)
			continue
		}

		if b.Len() > 0 {
			flush()
		}

		for len(line) > maxChars {
			out = append(out, strings.TrimSpace(line[:maxChars]))
			line = line[maxChars:]
		}
		if strings.TrimSpace(line) != "" {
			b.WriteString(strings.TrimSpace(line))
		}
	}
	flush()

	return out
}

func compactWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}
