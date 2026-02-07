package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/chat"
	"github.com/bilbilaki/ai2go/internal/config"
	"github.com/bilbilaki/ai2go/internal/ui"
)

func HandleCommand(cmd string, history *chat.History, store *chat.ThreadStore, cfg *config.Config, apiClient *api.Client) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	command := parts[0]

	switch command {
	case "/models", "/model":
		HandleModelSelection(cfg, apiClient)

	case "/current":
		fmt.Printf("Current model: %s\n", ui.Name(cfg.CurrentModel))

	case "/clear":
		history.Clear(cfg.CurrentModel)
		if err := store.SyncActiveHistory(history); err != nil {
			fmt.Printf("\033[31mError saving thread: %v\033[0m\n", err)
		}
		fmt.Println("\033[32mConversation history cleared.\033[0m")
	case "/threads":
		handleThreadsList(parts, store)
	case "/thread":
		handleThreadCommand(parts, history, store, cfg)
	case "/search":
		handleSearch(parts, store)

	case "/change_url":
		fmt.Print("Enter new Base URL: ")
		reader := bufio.NewReader(os.Stdin)
		newUrl, _ := reader.ReadString('\n')
		cfg.SetBaseURL(strings.TrimSpace(newUrl))
		fmt.Println("Base URL updated!")

	case "/change_apikey":
		fmt.Print("Enter new API Key: ")
		reader := bufio.NewReader(os.Stdin)
		newKey, _ := reader.ReadString('\n')
		cfg.SetAPIKey(strings.TrimSpace(newKey))
		fmt.Println("API Key updated!")

	case "/proxy":
		fmt.Print("Enter proxy URL (leave blank to disable): ")
		reader := bufio.NewReader(os.Stdin)
		newProxy, _ := reader.ReadString('\n')
		cfg.SetProxyURL(strings.TrimSpace(newProxy))
		if cfg.ProxyURL == "" {
			fmt.Println("Proxy disabled!")
		} else {
			fmt.Println("Proxy updated!")
		}

	case "/autoaccept":
		cfg.ToggleAutoAccept()
		status := "OFF"
		if cfg.AutoAccept {
			status = "ON"
		}
		fmt.Printf("Auto-accept commands is now: %s\n", status)
	case "/summarize":
		fmt.Println("\n\033[33mGenerating session summary...\033[0m")

		// 1. Prepare the prompt and sanitized history
		msgs := history.GetSanitizedMessages()
		prompt := "Summarize the current conversation history. " +
			"Focus on the user's goals, key commands executed, and important context. " +
			"Ignore specific details of long tool outputs (represented as 'toolcall successfully done'). " +
			"Be concise but comprehensive."

		msgs = append(msgs, api.Message{
			Role:    "user",
			Content: prompt,
		})

		// 2. Run completion (This will print the summary to the screen as it generates, which is good feedback)
		summaryMsg, err := apiClient.RunCompletion(context.Background(), msgs, nil, cfg.CurrentModel)
		if err != nil {
			fmt.Printf("\033[31mError generating summary: %v\033[0m\n", err)
			return
		}

		// 3. Replace the actual history with the new summary
		history.ReplaceWithSummary(cfg.CurrentModel, summaryMsg.Content)
		if err := store.SyncActiveHistory(history); err != nil {
			fmt.Printf("\033[31mError saving thread: %v\033[0m\n", err)
		}
		fmt.Println("\n\033[32mHistory summarized and context refreshed.\033[0m")
	case "/help":
		ShowHelp()
	case "/setup":
		HandleSetup(cfg, apiClient)
	default:
		fmt.Println(ui.Warn(fmt.Sprintf("Unknown command: %s. Type /help for available commands.", command)))
	}
}

func HandleSetup(cfg *config.Config, apiClient *api.Client) {
	reader := bufio.NewReader(os.Stdin)

	// Prompt for Base URL
	fmt.Print("Enter Base URL (e.g., https://api.openai.com): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		fmt.Println("Base URL is required.")
		return
	}
	cfg.SetBaseURL(baseURL)

	// Prompt for API Key
	fmt.Print("Enter API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		fmt.Println("API Key is required.")
		return
	}
	cfg.SetAPIKey(apiKey)

	// Prompt for Proxy
	fmt.Print("Enter Proxy URL (leave blank to disable): ")
	proxy, _ := reader.ReadString('\n')
	proxy = strings.TrimSpace(proxy)
	cfg.SetProxyURL(proxy)

	// Re-init client with new config
	apiClient = api.NewClient(cfg)

	// Fetch models
	fmt.Println("Fetching available models...")
	models, err := apiClient.GetAvailableModels()
	if err != nil {
		fmt.Printf("Error fetching models: %v. Please check your Base URL and API Key, then re-run /setup.\n", err)
		return
	}
	if len(models) == 0 {
		fmt.Println("No models available. Please check your setup and re-run /setup.")
		return
	}

	// Display and select model
	fmt.Printf("\nAvailable Models (%d):\n", len(models))
	for i, model := range models {
		fmt.Printf("%d. %s\n", i+1, model.ID)
	}
	fmt.Print("Select default model number: ")
	var input string
	fmt.Scanln(&input)
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(models) {
		fmt.Println("Invalid selection.")
		return
	}
	selectedModel := models[selection-1].ID
	cfg.SetCurrentModel(selectedModel)

	// Complete setup
	cfg.FirstSetup = false
	if err := cfg.Save(); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		return
	}
	fmt.Printf("Setup complete! Default model: %s. You can now chat or run /help.\n", selectedModel)
}
