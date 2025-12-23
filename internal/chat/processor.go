package chat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/config"
	"github.com/bilbilaki/ai2go/internal/tools"
)

func ProcessConversation(history *History, toolsList []api.Tool, cfg *config.Config, apiClient *api.Client) {
	for {
		assistantMsg, err := apiClient.RunCompletion(history.GetMessages(), toolsList, cfg.CurrentModel)
		if err != nil {
			fmt.Printf("\nError during completion: %v\n", err)
			return
		}

		history.AddAssistantMessage(assistantMsg)

		// If the AI didn't call any tools, we are done with this turn
		if len(assistantMsg.ToolCalls) == 0 {
			break
		}

		// Process tool calls
		for _, tCall := range assistantMsg.ToolCalls {
			if tCall.Function.Name == "run_command" {
				// Parse the JSON arguments string to get the command
				var args map[string]string
				json.Unmarshal([]byte(tCall.Function.Arguments), &args)
				cmdToRun := args["command"]

				// Check auto-accept
				if !cfg.AutoAccept {
					fmt.Printf("\n\033[33m[Tool Request] Command: %s\033[0m\n", cmdToRun)
					fmt.Print("Allow execution? (y/n): ")
					confirmScanner := bufio.NewScanner(os.Stdin)
					confirmScanner.Scan()
					if strings.ToLower(strings.TrimSpace(confirmScanner.Text())) != "y" {
						fmt.Println("Execution denied.")
						history.AddToolResponse(tCall.ID, "User denied permission to execute this command.")
						continue
					}
				} else {
					fmt.Printf("\n\033[33m[Auto-Running] Command: %s\033[0m\n", cmdToRun)
				}

				// Execute the command
				output, err := tools.ExecuteShellCommand(cmdToRun)

				// Show the result to the user
				if err != nil {
					fmt.Printf("\033[31m[Error]\033[0m %v\n", err)
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: %v", err))
				}
				fmt.Printf("\033[32m[Output]\033[0m\n%s\n----------------\n", output)
				history.AddToolResponse(tCall.ID, output)
			}
		}
	}
}