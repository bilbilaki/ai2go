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
2. You can use 'read_file' to inspect files with line numbers.
3. Prefer 'apply_unified_diff_patch' for edits using standard unified diffs (git diff format).
   - Required args: 'work_tree', 'patch'
   - Optional: 'verify_mode' in ['none', 'syntax', 'tests']
   - It auto-checkpoints and auto-rolls back on apply/verify failures.
4. You can use 'create_checkpoint', 'editor_history', and 'undo_checkpoints' for manual checkpoint workflow.
5. Legacy 'patch_file' is still available for old line-based patches, but use unified diff tools by default.
6. You can use process/system helpers when needed:
   - 'get_process_cpu_usage_sample' for PID CPU sampling
   - 'send_process_signal' for process tree signals
   - 'get_page_size' for OS page size
7. You can use 'subagent_factory' to split a mega task into concurrent subagent tasks and generate a report (requires experimental mode ON).
8. You can use 'subagent_context_provider' with task_id to fetch summarized volatile context from a subagent run.
9. You can use 'project_architect' to transform a rough project request into a detailed, implementation-ready step/task plan.
10. If user asks to create a big project, or asks for long multi-step work with subagents, FIRST call 'project_architect' using the user request as prompt, then split/delegate tasks to subagents.
11. For delegated execution, decide required subagent count from the generated plan and assign one concrete task per subagent.
12. Subagents do not need 'project_architect'; planner is for main agent orchestration.
13. When calling 'subagent_factory' for coding tasks, pass explicit 'timeout_sec' and 'max_concurrency'. Prefer lower concurrency for tasks that touch shared files.
14. Do not run dependent file-overlapping tasks in parallel. Run them step-by-step if they modify the same modules.
15. HANDLING LONG OUTPUT:
   - If a command returns "[OUTPUT TRUNCATED]", DO NOT apologize. 
   - IMMEDIATELY run a new command to filter the data (e.g., 'grep "error" file.log', 'tail -n 10 file.log').
   - Never output huge chunks of text yourself.
16. Use 'ask_user' when requirements are ambiguous or there are multiple valid solution paths.
    - Pass a clear 'question'.
    - Add 'options' only if useful; otherwise ask free text.
    - You may ask follow-up questions via repeated 'ask_user' calls until requirements are clear.
17. For large messy media folders, prefer 'organize_media_files' instead of long shell loops:
    - First run with dry_run=true and show preview summary.
    - Then ask for confirmation and run with dry_run=false.
18. Always explain your plan briefly before executing commands.`, osName),
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

	h.messages = append(h.messages, api.Message{
		Role: "system",
		Content: fmt.Sprintf(
			"Conversation memory (compressed summary):\n%s\n\nUse this as authoritative prior context for follow-up turns.",
			summary,
		),
	})

	fmt.Printf("\n\033[90m[Debug] Token count reset to: %d\033[0m\n", h.GetTotalTokens())
}
