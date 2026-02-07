package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/config"
	"github.com/bilbilaki/ai2go/internal/tools"
	"github.com/bilbilaki/ai2go/internal/ui"
)

func ProcessConversation(ctx context.Context, history *History, toolsList []api.Tool, cfg *config.Config, apiClient *api.Client) {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		if ctx.Err() != nil {
			fmt.Println(ui.Warn("[System] Current run stopped."))
			return
		}

		msgs, changed := history.GetMessagesForAPI()
		if changed {
			fmt.Println(ui.Warn("[History Repair] Removed invalid tool messages from current thread."))
			history.LoadMessages(msgs, cfg.CurrentModel)
		}

		assistantMsg, err := apiClient.RunCompletion(ctx, msgs, toolsList, cfg.CurrentModel)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println(ui.Warn("[System] Request canceled by user."))
				return
			}
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
			toolResponse := ""
			switch tCall.Function.Name {
			case "run_command":
				var args map[string]string
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for run_command: %v", err)
					break
				}
				cmdToRun := strings.TrimSpace(args["command"])
				if cmdToRun == "" {
					toolResponse = "Error: run_command requires a non-empty 'command' argument."
					break
				}

				if !cfg.AutoAccept {
					fmt.Printf("\n%s\n", ui.Tool(fmt.Sprintf("[Tool Request] Command: %s", cmdToRun)))
					fmt.Print("Allow execution? (y/n): ")
					confirmScanner := bufio.NewScanner(os.Stdin)
					confirmScanner.Scan()
					if strings.ToLower(strings.TrimSpace(confirmScanner.Text())) != "y" {
						fmt.Println("Execution denied.")
						toolResponse = "User denied permission to execute this command."
						break
					}
				} else {
					fmt.Printf("\n%s\n", ui.Tool(fmt.Sprintf("[Auto-Running] Command: %s", cmdToRun)))
				}

				output, err := tools.ExecuteShellCommand(ctx, cmdToRun)

				if err != nil {
					fmt.Printf("\033[31m[Error]\033[0m %v\n", err)
					if strings.TrimSpace(output) == "" {
						toolResponse = fmt.Sprintf("Error: %v", err)
					} else {
						toolResponse = fmt.Sprintf("%s\n\nError: %v", output, err)
					}
				} else {
					toolResponse = output
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), output)

			case "read_file":
				var args map[string]string
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for read_file: %v", err)
					break
				}
				pathToRead := strings.TrimSpace(args["path"])
				if pathToRead == "" {
					toolResponse = "Error: read_file requires a non-empty 'path' argument."
					break
				}

				if !cfg.AutoAccept {
					fmt.Printf("\n%s\n", ui.Tool(fmt.Sprintf("[Tool Request] read file: %s", pathToRead)))
					fmt.Print("Allow reading file? (y/n): ")
					confirmScanner := bufio.NewScanner(os.Stdin)
					confirmScanner.Scan()
					if strings.ToLower(strings.TrimSpace(confirmScanner.Text())) != "y" {
						fmt.Println("reading file denied.")
						toolResponse = "User denied permission to read this file."
						break
					}
				} else {
					fmt.Printf("\n%s\n", ui.Tool(fmt.Sprintf("[Auto-Running] read file: %s", pathToRead)))
				}

				output, err := tools.ReadFileWithLines(pathToRead)
				if err != nil {
					fmt.Printf("\033[31m[Error]\033[0m %v\n", err)
					toolResponse = fmt.Sprintf("Error: %v", err)
				} else {
					toolResponse = output
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), output)

			case "patch_file":
				var args map[string]string
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for patch_file: %v", err)
					break
				}
				pathToPatch := strings.TrimSpace(args["path"])
				patch := args["patch"]
				if pathToPatch == "" {
					toolResponse = "Error: patch_file requires a non-empty 'path' argument."
					break
				}
				if strings.TrimSpace(patch) == "" {
					toolResponse = "Error: patch_file requires a non-empty 'patch' argument."
					break
				}

				if !cfg.AutoAccept {
					fmt.Printf("\n%s\n", ui.Tool(fmt.Sprintf("[Tool Request] Edit File: %s", pathToPatch)))
					fmt.Print("Allow Edit File? (y/n): ")
					confirmScanner := bufio.NewScanner(os.Stdin)
					confirmScanner.Scan()
					if strings.ToLower(strings.TrimSpace(confirmScanner.Text())) != "y" {
						fmt.Println("Edit File denied.")
						toolResponse = "User denied permission to edit this file."
						break
					}
				} else {
					fmt.Printf("\n%s\n", ui.Tool(fmt.Sprintf("[Auto-Running] Edit File: %s", pathToPatch)))
				}

				fmt.Printf("\n%s\n", ui.Tool(fmt.Sprintf("[Tool] Patching file: %s", pathToPatch)))
				output, err := tools.ApplyFilePatch(pathToPatch, patch)
				if err != nil {
					fmt.Printf("\033[31m[Error]\033[0m %v\n", err)
					toolResponse = fmt.Sprintf("Error: %v", err)
				} else {
					toolResponse = output
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), output)

			default:
				toolResponse = fmt.Sprintf("Error: unsupported tool '%s'", tCall.Function.Name)
			}

			history.AddToolResponse(tCall.ID, toolResponse)
		}
	}
}
