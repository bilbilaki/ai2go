package chat

import (
	"fmt"
	"runtime"
	"unicode/utf8"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/pandodao/tokenizer-go"
)

const maxToolResponseChars = 6000

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
		Content: fmt.Sprintf(`You are an advanced terminal assistant. 
Current OS: %s
 
RULES:
1. You can use 'run_command' to execute shell commands.
1. You can use 'read_file' first to see line numbers.
2. You can use 'patch_file' with this custom syntax to edit:
   - "26--"         -> Remove line 26.
   - "26++ code"    -> Replace line 26 with "code".
   - "26++"         -> Clear line 26 (make it empty).
   - "0++ code"     -> Insert "code" at the VERY START of file.
   - "00++ code"    -> Append "code" to the VERY END of file.
3. IMPORTANT: If You want Using 'patch_file' for Editing files Use the ORIGINAL line numbers from 'read_file'. The tool handles the offsets automatically. Do not manually calculate shifted line numbers.
4. HANDLING LONG OUTPUT:
   - If a command returns "[OUTPUT TRUNCATED]", DO NOT apologize. 
   - IMMEDIATELY run a new command to filter the data (e.g., 'grep "error" file.log', 'tail -n 10 file.log').
   - Never output huge chunks of text yourself.
5. Always explain your plan briefly before executing commands.`, osName),
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
	content = truncateForHistory(content, maxToolResponseChars)

	// FIX: The API requires a non-empty 'content' field for tool messages.
	// If the tool produced no output, we must provide a placeholder.
	if content == "" {
		content = "Tool executed successfully (no output)."
	}

	h.messages = append(h.messages, api.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
	})

	// Only count tokens if we have a counter
	if h.counter != nil {
		h.counter.Add(ApproximateTokens(content))
	}
}

func truncateForHistory(s string, maxChars int) string {
	if maxChars <= 0 || s == "" {
		return s
	}

	if !utf8.ValidString(s) {
		s = string([]rune(s))
	}

	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}

	trimmed := string(runes[:maxChars])
	return fmt.Sprintf("%s\n\n... [TOOL OUTPUT TRUNCATED FOR HISTORY: %d chars removed] ...", trimmed, len(runes)-maxChars)
}

func (h *History) Clear(currentModel string) {
	h.SetSystemMessage(currentModel)
	h.counter.Reset()
}

func (h *History) LoadMessages(messages []api.Message, currentModel string) {
	if len(messages) == 0 {
		h.Clear(currentModel)
		return
	}

	h.messages = make([]api.Message, len(messages))
	copy(h.messages, messages)

	h.counter.Reset()
	for _, msg := range h.messages {
		if msg.Role == "system" {
			continue
		}
		h.counter.Add(ApproximateTokens(msg.Content))
	}
}

func (h *History) GetMessages() []api.Message {
	return h.messages
}

func (h *History) GetMessagesForAPI() ([]api.Message, bool) {
	if len(h.messages) == 0 {
		return h.messages, false
	}

	clean := make([]api.Message, 0, len(h.messages))
	pendingToolCalls := map[string]struct{}{}
	changed := false

	for _, msg := range h.messages {
		switch msg.Role {
		case "assistant":
			for _, tc := range msg.ToolCalls {
				if tc.ID != "" {
					pendingToolCalls[tc.ID] = struct{}{}
				}
			}
			clean = append(clean, msg)
		case "tool":
			if msg.ToolCallID == "" {
				changed = true
				continue
			}
			if _, ok := pendingToolCalls[msg.ToolCallID]; !ok {
				changed = true
				continue
			}
			delete(pendingToolCalls, msg.ToolCallID)
			clean = append(clean, msg)
		default:
			clean = append(clean, msg)
		}
	}

	return clean, changed
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
