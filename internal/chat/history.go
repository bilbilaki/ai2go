package chat

import (
	"fmt"

	"github.com/bilbilaki/ai2go/internal/api"
)

type History struct {
	messages []api.Message
}

func NewHistory(currentModel string) *History {
	h := &History{
		messages: []api.Message{},
	}
	h.SetSystemMessage(currentModel)
	return h
}

func (h *History) SetSystemMessage(currentModel string) {
	systemMessage := api.Message{
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
	
	// Clear existing messages and set new system message
	h.messages = []api.Message{systemMessage}
}

func (h *History) AddUserMessage(content string) {
	h.messages = append(h.messages, api.Message{
		Role:    "user",
		Content: content,
	})
}

func (h *History) AddAssistantMessage(msg api.Message) {
	h.messages = append(h.messages, msg)
}

func (h *History) AddToolResponse(toolCallID, content string) {
	h.messages = append(h.messages, api.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
	})
}

func (h *History) Clear(currentModel string) {
	h.SetSystemMessage(currentModel)
}

func (h *History) GetMessages() []api.Message {
	return h.messages
}
