package chat

import (
	"fmt"
"github.com/pandodao/tokenizer-go"
	"github.com/bilbilaki/ai2go/internal/api"
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
	sysMsg := api.Message{
		Role: "system",
		Content: fmt.Sprintf(`You are an advanced Linux terminal assistant. 
Current model: %s

RULES:
1. You can use 'run_command' to execute shell commands.
2. HANDLING LONG OUTPUT:
   - If a command returns "[OUTPUT TRUNCATED]", DO NOT apologize. 
   - IMMEDIATELY run a new command to filter the data (e.g., 'grep "error" file.log', 'tail -n 10 file.log').
   - Never output huge chunks of text yourself.
3. Always explain your plan briefly before executing commands.`, currentModel),
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