package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yourusername/terminal-assistant/internal/api"
	"github.com/yourusername/terminal-assistant/internal/config"
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
