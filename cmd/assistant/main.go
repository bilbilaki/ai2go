package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/chat"
	"github.com/bilbilaki/ai2go/internal/commands"
	"github.com/bilbilaki/ai2go/internal/config"
	"github.com/bilbilaki/ai2go/internal/session"
	"github.com/bilbilaki/ai2go/internal/storage"
	"github.com/bilbilaki/ai2go/internal/tools"
	"github.com/bilbilaki/ai2go/internal/utils" // Import the new utils package
	"os"
	"strings"
)

func main() {
	// Load configuration
	cfg := config.Load()

	fmt.Println("\n=== Terminal Assistant with Model Switching ===")
	commands.ShowHelp()
	if cfg.FirstSetup {
		fmt.Println("\n\033[1;33mWelcome! For first setup, run /setup.\033[0m")
	}
	if !cfg.FirstSetup {
		fmt.Println("\nCurrent model:", cfg.CurrentModel)
	}
	fmt.Println("\n" + strings.Repeat("=", 50))

	history := chat.NewHistory(cfg.CurrentModel)
	cliTool := tools.GetCLITool()
	readTool := tools.GetReadFileTool()   // <--- New
	patchTool := tools.GetPatchFileTool() // <--- New
	searchTool := tools.GetSearchFilesTool()
	fileTreeTool := tools.GetListTreeTool()
	toolsList := []api.Tool{cliTool, readTool, patchTool, searchTool, fileTreeTool}
	apiClient := api.NewClient(cfg)
	scanner := bufio.NewScanner(os.Stdin)
	var store *storage.Store
	var state session.State

	configDir, err := config.ConfigDir()
	if err != nil {
		fmt.Printf("\n\033[31m[Warning]\033[0m Failed to resolve config dir for history: %v\n", err)
	} else {
		dbPath := filepath.Join(configDir, "history.db")
		store, err = storage.Open(dbPath)
		if err != nil {
			fmt.Printf("\n\033[31m[Warning]\033[0m Failed to open history database: %v\n", err)
		} else {
			defer store.Close()
		}
	}

	if store != nil {
		chatID, err := store.CreateChat("Untitled")
		if err != nil {
			fmt.Printf("\n\033[31m[Warning]\033[0m Failed to create initial chat: %v\n", err)
		} else {
			state.ChatID = chatID
			state.Title = "Untitled"
		}
	} else {
		state.Title = "Untitled"
	}

	for {
		tokens := history.GetTotalTokens()
		fmt.Printf("Tokens: %d (±50) > ", tokens)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {

			continue
		}

		// Handle special commands
		if strings.HasPrefix(input, "/") && !strings.Contains(input, "/file") {
			commands.HandleCommand(input, history, cfg, apiClient, store, &state)
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}
		if cfg.CurrentModel == "" {
			fmt.Println("\n\033[1;33mNo model configured. Run /setup to select a model.\033[0m")
			continue
		}
		finalMessage := input

		// Check if we need to resolve files OR if the user just wants to type more
		for {
			// If the current chunk has /file, resolve it immediately
			if strings.Contains(finalMessage, "/file") {
				finalMessage = utils.ResolveFileTokens(finalMessage)

				// Show status
				fmt.Println("\n\033[36m[Draft Mode] File attached.\033[0m")
				fmt.Println("\033[90mCurrent message length:", len(finalMessage), "characters.\033[0m")
				fmt.Println("Type more to append to this message, or press [ENTER] to send to AI.")

				// Wait for more input
				fmt.Print(">> ")
				if !scanner.Scan() {
					break
				}
				appendInput := scanner.Text() // allow leading spaces

				if strings.TrimSpace(appendInput) == "" {
					// User hit Enter on empty line -> Send it!
					break
				} else {
					// User typed more text -> Append it and loop again
					// Add a space for natural flow if needed
					finalMessage += " " + appendInput
					continue
				}
			} else {
				// No /file token, just send normally
				break
			}
		}

		// 4. Send to AI
		history.AddUserMessage(finalMessage)
		if store != nil && state.ChatID != 0 {
			if err := store.SaveMessage(state.ChatID, "user", finalMessage); err != nil {
				fmt.Printf("\n\033[31m[Warning]\033[0m Failed to save user message: %v\n", err)
			}
		}
		if !state.HasMessages {
			state.HasMessages = true
			if store != nil && state.ChatID != 0 {
				title, err := commands.GenerateChatTitle(apiClient, cfg.CurrentModel, finalMessage)
				if err != nil || strings.TrimSpace(title) == "" {
					title = "Untitled"
				}
				if err := store.UpdateChatTitle(state.ChatID, title); err != nil {
					fmt.Printf("\n\033[31m[Warning]\033[0m Failed to update chat title: %v\n", err)
				} else {
					state.Title = title
				}
			}
		}
		chat.ProcessConversation(history, toolsList, cfg, apiClient, store, state.ChatID)
	}
}
