package commands

import (
"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/config"
)

func HandleModelSelection(cfg *config.Config, apiClient *api.Client) {
	fmt.Println("\n\033[1mFetching available models...\033[0m")

	models, err := apiClient.GetAvailableModels()
	if err != nil {
		fmt.Printf("\033[31mError fetching models: %v\033[0m\n", err)
		return
	}

	if len(models) == 0 {
		fmt.Println("\033[33mNo models found.\033[0m")
		return
	}

	// Display models
	fmt.Printf("\n\033[1mAvailable Models (%d):\033[0m\n", len(models))
	for i, model := range models {
		marker := " "
		if model.ID == cfg.CurrentModel {
			marker = "*"
		}
		fmt.Printf("%s %2d. %-35s (created: %s)\n", marker, i+1, model.ID,
			time.Unix(model.Created, 0).Format("2006-01-02"))
	}

	// Simple, blocking input read
	fmt.Print("\nSelect model number (or 'c' to cancel): ")
	var input string
	fmt.Scanln(&input)

	if input == "" || strings.ToLower(input) == "c" {
		fmt.Println("\nModel selection cancelled.")
		return
	}

	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(models) {
		fmt.Printf("\033[31mInvalid selection.\033[0m\n")
		return
	}

	selectedModel := models[selection-1].ID
	if selectedModel == cfg.CurrentModel {
		fmt.Println("\033[33mAlready using this model.\033[0m")
		return
	}

	cfg.SetCurrentModel(selectedModel)
	fmt.Printf("\033[32m✓ Model switched to: %s\033[0m\n", selectedModel)
}
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type Delta struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type Choice struct {
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type StreamChunk struct {
	Choices []Choice `json:"choices"`
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}
