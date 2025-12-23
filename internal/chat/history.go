package chat

import (
	"fmt"
	"runtime"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/pandodao/tokenizer-go"
)

// TokenCounter approximates session token usage (1 token ~4 chars).
type TokenCounter struct {
	total int64
}

func (tc *TokenCounter) Add(tokens int64) {
	tc.total += tokens
}

func (tc *TokenCounter) Reset() {
	tc.total = 0
}

func (tc *TokenCounter) GetTotal() int64 {
	return tc.total
}

// ApproximateTokens returns ~token count for text (rough: 4 chars/token).
func ApproximateTokens(text string) int64 {
	t := tokenizer.MustCalToken(text)
	return int64(t)
}

type History struct {
	messages []api.Message
	counter  *TokenCounter
}

func NewHistory(currentModel string) *History {
	h := &History{
		messages: []api.Message{},
		counter:  &TokenCounter{},
	}
	h.SetSystemMessage(currentModel)
	return h
}

func (h *History) SetSystemMessage(currentModel string) {
	osName := "Linux/Mac"
	if runtime.GOOS == "windows" {
		osName = "Windows"
	}
	sysMsg := api.Message{
		Role: "system",
		Content: 
		fmt.Sprintf(`You are an advanced terminal assistant. 
Current OS: %s

RULES:
1. You can use 'run_command' to execute shell commands.
2. HANDLING LONG OUTPUT:
   - If a command returns "[OUTPUT TRUNCATED]", DO NOT apologize. 
   - IMMEDIATELY run a new command to filter the data (e.g., 'grep "error" file.log', 'tail -n 10 file.log').
   - Never output huge chunks of text yourself.
3. Always explain your plan briefly before executing commands.`, osName),
	}
	h.messages = []api.Message{sysMsg}
}

func (h *History) AddUserMessage(content string) {
	h.messages = append(h.messages, api.Message{
		Role:    "user",
		Content: content,
	})
	h.counter.Add(ApproximateTokens(content))
}

func (h *History) AddAssistantMessage(msg api.Message) {
	h.messages = append(h.messages, msg)
	h.counter.Add(ApproximateTokens(msg.Content))
}

func (h *History) AddToolResponse(toolCallID, content string) {
	h.messages = append(h.messages, api.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
	})
	h.counter.Add(ApproximateTokens(content))
}

func (h *History) Clear(currentModel string) {
	h.SetSystemMessage(currentModel)
	h.counter.Reset()
}

func (h *History) GetMessages() []api.Message {
	return h.messages
}

func (h *History) GetTotalTokens() int64 {
	return h.counter.GetTotal()
}

func (h *History) GetSanitizedMessages() []api.Message {
	sanitized := make([]api.Message, len(h.messages))
	for i, msg := range h.messages {
		tempMsg := msg
		if tempMsg.Role == "tool" {
			tempMsg.Content = "toolcall successfully done"
		}
		sanitized[i] = tempMsg
	}
	return sanitized
}

func (h *History) ReplaceWithSummary(currentModel, summary string) {
	h.Clear(currentModel)

	summaryContext := fmt.Sprintf("Here is a summary of our conversation so far. Use this context to continue assisting me:\n\n%s", summary)
	
	h.AddUserMessage(summaryContext)

	h.AddAssistantMessage(api.Message{
		Role:    "assistant",
		Content: "Understood. I have updated my context with the summary and am ready to continue.",
	})
	
	fmt.Printf("\n\033[90m[Debug] Token count reset to: %d\033[0m\n", h.GetTotalTokens())
}