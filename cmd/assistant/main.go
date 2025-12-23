package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/chat"
	"github.com/bilbilaki/ai2go/internal/commands"
	"github.com/bilbilaki/ai2go/internal/config"
	"github.com/bilbilaki/ai2go/internal/tools"
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
	toolsList := []api.Tool{cliTool}
apiClient := api.NewClient(cfg)
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
			commands.HandleCommand(input, history, cfg, apiClient)
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
