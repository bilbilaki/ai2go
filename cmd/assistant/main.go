package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/yourusername/terminal-assistant/internal/api"
	"github.com/yourusername/terminal-assistant/internal/chat"
	"github.com/yourusername/terminal-assistant/internal/commands"
	"github.com/yourusername/terminal-assistant/internal/config"
	"github.com/yourusername/terminal-assistant/internal/tools"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize HTTP client
	apiClient := api.NewClient(cfg)

	// Test API connection
	models, err := apiClient.GetAvailableModels()
	if err != nil {
		fmt.Printf("Warning: Could not fetch models: %v\n", err)
		fmt.Println("Using default model:", cfg.CurrentModel)
	} else {
		fmt.Printf("Connected successfully. Found %d models.\n", len(models))
	}

	fmt.Println("\n=== Terminal Assistant with Model Switching ===")
	commands.ShowHelp()
	fmt.Println("\nCurrent model:", cfg.CurrentModel)
	fmt.Println("\n" + strings.Repeat("=", 50))

	// Initialize chat history
	history := chat.NewHistory(cfg.CurrentModel)
	
	// Get available tools
	cliTool := tools.GetCLITool()
	toolsList := []api.Tool{cliTool}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle special commands
		if strings.HasPrefix(input, "/") {
			commands.HandleCommand(input, &history, cfg, apiClient)
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		// Process user input
		history.AddUserMessage(input)
		chat.ProcessConversation(history, toolsList, cfg, apiClient)
	}
}
