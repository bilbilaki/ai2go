package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bilbilaki/ai2go/internal/config"
)

type Client struct {
	httpClient *http.Client
	config     *config.Config
}

func NewClient(cfg *config.Config) *Client {
	client := &Client{
		config: cfg,
	}
	client.initHTTPClient()
	return client
}

func (c *Client) initHTTPClient() {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if c.config.ProxyURL != "" {
		proxyStr := c.config.ProxyURL
		if !strings.Contains(proxyStr, "://") {
			proxyStr = "http://" + proxyStr
		}

		proxyURL, err := url.Parse(proxyStr)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
			fmt.Printf("\033[33m[System] Using Proxy: %s\033[0m\n", proxyStr)
		} else {
			fmt.Printf("\033[31m[Error] Invalid proxy URL: %v. Using system/default proxy.\033[0m\n", err)
		}
	}

	timeout := time.Duration(c.config.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	c.httpClient = &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func (c *Client) Reload() {
	c.initHTTPClient()
}

func (c *Client) GetAvailableModels() ([]Model, error) {
	req, err := http.NewRequest("GET", c.config.BaseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, body)
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	return modelsResp.Data, nil
}

func (c *Client) RunCompletion(history []Message, tools []Tool, model string) (Message, error) {
	reqBody, err := json.Marshal(ChatRequest{
		Model:    model,
		Messages: history,
		Stream:   true,
		Tools:    tools,
	})
	if err != nil {
		return Message{}, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.BaseURL+"/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return Message{}, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("error contacting API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return Message{}, fmt.Errorf("API error %d: %s", resp.StatusCode, body)
	}

	return c.handleStreamingResponse(resp.Body)
}

func (c *Client) handleStreamingResponse(body io.ReadCloser) (Message, error) {
	br := bufio.NewReader(body)

	var fullMessage Message
	fullMessage.Role = "assistant"

	toolCallIndices := make(map[int]*ToolCall)
	currentToolCallIndex := -1
	inThinkingBlock := false

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return Message{}, fmt.Errorf("stream error: %w", err)
			}
			break
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk StreamChunk
			if json.Unmarshal([]byte(data), &chunk) == nil {
				for _, choice := range chunk.Choices {
					// Handle Thinking/Reasoning Content
					if choice.Delta.Thinking != "" || choice.Delta.Reasoning != "" {
						if !inThinkingBlock {
							printThinkingBlockStart()
							inThinkingBlock = true
						}
						printThinkingContent(choice.Delta.Thinking + choice.Delta.Reasoning)
					}

					// Handle Text Content
					if choice.Delta.Content != "" {
						if inThinkingBlock {
							printThinkingBlockEnd()
							inThinkingBlock = false
						}
						fmt.Print(choice.Delta.Content)
						fullMessage.Content += choice.Delta.Content
					}

					// Handle Tool Call chunks
					for i, tcChunk := range choice.Delta.ToolCalls {
						idx := i
						currentToolCallIndex = idx

						if _, exists := toolCallIndices[idx]; !exists {
							toolCallIndices[idx] = &ToolCall{
								ID:       tcChunk.ID,
								Type:     tcChunk.Type,
								Function: FunctionCall{},
							}
							toolCallOrder = append(toolCallOrder, key)
						}

						// Append fragments
						if tcChunk.ID != "" {
							toolCallIndices[idx].ID = tcChunk.ID
						}
						if tcChunk.Type != "" {
							toolCallIndices[idx].Type = tcChunk.Type
						}
						if tcChunk.Function.Name != "" {
							toolCallIndices[idx].Function.Name += tcChunk.Function.Name
						}
						if tcChunk.Function.Arguments != "" {
							toolCallIndices[idx].Function.Arguments += tcChunk.Function.Arguments
						}
					}
				}
			}
		}
	}

	// Reassemble tool calls into the final message
	if inThinkingBlock {
		printThinkingBlockEnd()
	}
	for i := 0; i <= currentToolCallIndex; i++ {
		if tc, exists := toolCallIndices[i]; exists {
			fullMessage.ToolCalls = append(fullMessage.ToolCalls, *tc)
		}
	}

	return fullMessage, nil
}

func printThinkingBlockStart() {
	fmt.Print("\n┌─ Thinking ───────────────────────────────────────────────┐\n")
}

func printThinkingContent(content string) {
	if content == "" {
		return
	}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fmt.Printf("│ %s\n", line)
	}
}

func printThinkingBlockEnd() {
	fmt.Print("└─────────────────────────────────────────────────────────┘\n")
}
