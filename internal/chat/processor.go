package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"

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
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: invalid arguments for run_command: %v", err))
					continue
				}
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
				ctx, cancel := context.WithCancel(context.Background())

				// 2. Setup signal channel to listen for Ctrl+C
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

				// 3. Start a goroutine to watch for the signal
				go func() {
					select {
					case <-sigChan:
						fmt.Println("\n\033[31m!!! Stopping command... !!!\033[0m")
						cancel() // Cancels the context passed to ExecuteShellCommand
					case <-ctx.Done():
						// Command finished normally
					}
				}()

				// 4. Execute the command with the context
				output, err := tools.ExecuteShellCommand(ctx, cmdToRun)

				// 5. Cleanup: Stop listening for signals and ensure context is cancelled
				signal.Stop(sigChan)
				cancel()

				// Show the result to the user
				if err != nil {
					fmt.Printf("\033[31m[Error]\033[0m %v\n", err)
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: %v", err))
				}
				fmt.Printf("\033[32m[Output]\033[0m\n%s\n----------------\n", output)
				history.AddToolResponse(tCall.ID, output)
			}
			if tCall.Function.Name == "read_file" {
				// Check auto-accept
				var args map[string]string
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: invalid arguments for read_file: %v", err))
					continue
				}
				cmdToRun := args["path"]
				lineRange := args["line_range"] // Get optional line_range parameter

				// Build display string
				displayStr := cmdToRun
				if lineRange != "" {
					displayStr = fmt.Sprintf("%s (lines %s)", cmdToRun, lineRange)
				}

				if !cfg.AutoAccept {
					fmt.Printf("\n\033[33m[Tool Request] reading file: %s\033[0m\n", displayStr)
					fmt.Print("Allow reading file? (y/n): ")
					confirmScanner := bufio.NewScanner(os.Stdin)
					confirmScanner.Scan()
					if strings.ToLower(strings.TrimSpace(confirmScanner.Text())) != "y" {
						fmt.Println("reading file denied.")
						history.AddToolResponse(tCall.ID, "User denied permission to reading.")
						continue
					}
				} else {
					fmt.Printf("\n\033[33m[Auto-Running] reading file: %s\033[0m\n", displayStr)
				}

				ctx, cancel := context.WithCancel(context.Background())

				// 2. Setup signal channel to listen for Ctrl+C
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

				// 3. Start a goroutine to watch for the signal
				go func() {
					select {
					case <-sigChan:
						fmt.Println("\n\033[31m!!! Stopping AI tool... !!!\033[0m")
						cancel() // Cancels the context passed to ReadFileWithLines
					case <-ctx.Done():
						// Command finished normally
					}
				}()

				// Pass lineRange parameter (empty string if not provided)
				output, err := tools.ReadFileWithLines(cmdToRun, lineRange)

				// 5. Cleanup: Stop listening for signals and ensure context is cancelled
				signal.Stop(sigChan)
				cancel()

				// Show the result to the user
				if err != nil {
					fmt.Printf("\033[31m[Error]\033[0m %v\n", err)
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: %v", err))
				} else {
					fmt.Printf("\033[32m[Output]\033[0m\n%s\n----------------\n", output)
					history.AddToolResponse(tCall.ID, output)
				}
			}
			if tCall.Function.Name == "patch_file" {
				var args map[string]string
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: invalid arguments for patch_file: %v", err))
					continue
				}
				cmdToRun := args["path"]
				cmdToPatch := args["patch"]
				if !cfg.AutoAccept {
					fmt.Printf("\n\033[33m[Tool Request] Edit File: %s\033[0m\n", cmdToRun)
					fmt.Print("Allow Edit File? (y/n): ")
					confirmScanner := bufio.NewScanner(os.Stdin)
					confirmScanner.Scan()
					if strings.ToLower(strings.TrimSpace(confirmScanner.Text())) != "y" {
						fmt.Println("Edit File denied.")
						history.AddToolResponse(tCall.ID, "User denied permission to Edit this File.")
						continue
					}
				} else {
					fmt.Printf("\n\033[33m[Auto-Running] Edit File: %s\033[0m\n", cmdToRun)
				}
				ctx, cancel := context.WithCancel(context.Background())

				// 2. Setup signal channel to listen for Ctrl+C
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

				// 3. Start a goroutine to watch for the signal
				go func() {
					select {
					case <-sigChan:
						fmt.Println("\n\033[31m!!! Stopping AI Tool... !!!\033[0m")
						cancel() // Cancels the context passed to ExecuteShellCommand
					case <-ctx.Done():
						// Command finished normally
					}
				}()

				fmt.Printf("\n\033[33m[Tool] Patching file: %s\033[0m\n", args["path"])
				output, err := tools.ApplyFilePatch(cmdToRun, cmdToPatch)

				// Common output handling
				if err != nil {
					fmt.Printf("\033[31m[Error]\033[0m %v\n", err)
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: %v", err))
				}
				fmt.Printf("\033[32m[Output]\033[0m\n%s\n----------------\n", output)
				history.AddToolResponse(tCall.ID, output)
			}
			if tCall.Function.Name == "search_files" {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: invalid arguments for search_files: %v", err))
					continue
				}

				// Build Config from AI arguments
				dir, _ := args["dir"].(string)
				absPath, _ := filepath.Abs(dir)

				cfgSearch := &tools.Config{
					RootPath:       absPath,
					ContentInclude: true,
					WorkerCount:    runtime.NumCPU(),
				}

				if v, ok := args["ext"].(string); ok {
					for _, e := range strings.Split(v, ",") {
						ext := strings.TrimSpace(e)
						if !strings.HasPrefix(ext, ".") {
							ext = "." + ext
						}
						cfgSearch.Extensions = append(cfgSearch.Extensions, strings.ToLower(ext))
					}
				}
				if v, ok := args["inc_path"].(string); ok {
					cfgSearch.IncludePathRegex = tools.CompilePatterns(v)
				}
				if v, ok := args["exc_path"].(string); ok {
					cfgSearch.ExcludePathRegex = tools.CompilePatterns(v)
				}
				if v, ok := args["content"].(string); ok {
					re, err := regexp.Compile(v)
					if err != nil {
						history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: invalid content regex: %v", err))
						continue
					}
					cfgSearch.ContentRegex = re
				}
				if v, ok := args["content_exclude"].(bool); ok {
					cfgSearch.ContentInclude = !v
				}

				fmt.Printf("\n\033[33m[Tool Request] Searching in: %s\033[0m\n", dir)

				if !cfg.AutoAccept {
					fmt.Printf("\n\033[33m[Tool Request] Search File: %s\033[0m\n", args)
					fmt.Print("Allow Search File? (y/n): ")
					confirmScanner := bufio.NewScanner(os.Stdin)
					confirmScanner.Scan()
					if strings.ToLower(strings.TrimSpace(confirmScanner.Text())) != "y" {
						fmt.Println("Search File denied.")
						history.AddToolResponse(tCall.ID, "User denied permission to Search for this File.")
						continue
					}
				} else {
					fmt.Printf("\n\033[33m[Auto-Running] Search File: %s\033[0m\n", args)
				}
				ctx, cancel := context.WithCancel(context.Background())

				// 2. Setup signal channel to listen for Ctrl+C
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

				// 3. Start a goroutine to watch for the signal
				go func() {
					select {
					case <-sigChan:
						fmt.Println("\n\033[31m!!! Stopping AI Tool... !!!\033[0m")
						cancel() // Cancels the context passed to ExecuteShellCommand
					case <-ctx.Done():
						// Command finished normally
					}
				}()

				output, err := tools.SearchFilesWrapper(ctx, cfgSearch)
				cancel()

				if err != nil {
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: %v", err))
				} else {
					fmt.Printf("\033[32m[Output]\033[0m Found results.\n")
					history.AddToolResponse(tCall.ID, output)
				}
			}

			if tCall.Function.Name == "list_tree" {
				var args map[string]string
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: invalid arguments for list_tree: %v", err))
					continue
				}
				dir := args["dir"]

				absPath, _ := filepath.Abs(dir)
				cfgTree := &tools.Config{RootPath: absPath, ShowTree: true}

				fmt.Printf("\n\033[33m[Tool Request] Listing Tree: %s\033[0m\n", dir)

				if !cfg.AutoAccept {
					fmt.Print("Allow listing tree? (y/n): ")
					confirmScanner := bufio.NewScanner(os.Stdin)
					confirmScanner.Scan()
					if strings.ToLower(strings.TrimSpace(confirmScanner.Text())) != "y" {
						fmt.Println("Listing tree denied.")
						history.AddToolResponse(tCall.ID, "User denied permission to list tree.")
						continue
					}
				} else {
					fmt.Printf("\n\033[33m[Auto-Running] Listing Tree: %s\033[0m\n", dir)
				}

				output, err := tools.ListTreeWrapper(cfgTree)

				if err != nil {
					history.AddToolResponse(tCall.ID, fmt.Sprintf("Error: %v", err))
				} else {
					history.AddToolResponse(tCall.ID, output)
				}
			}

		}
	}
}
