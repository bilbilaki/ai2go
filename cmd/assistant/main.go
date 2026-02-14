package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/chat"
	"github.com/bilbilaki/ai2go/internal/commands"
	"github.com/bilbilaki/ai2go/internal/config"
	"github.com/bilbilaki/ai2go/internal/tools"
	"github.com/bilbilaki/ai2go/internal/ui"
	"github.com/bilbilaki/ai2go/internal/utils" // Import the new utils package
	"github.com/chzyer/readline"
	"os"
	"os/signal"
)

func main() {
	// Load configuration
	cfg := config.Load()

	fmt.Println(ui.System("\n=== Terminal Assistant with Model Switching ==="))
	commands.ShowHelp()
	if cfg.FirstSetup {
		fmt.Println(ui.Warn("\nWelcome! For first setup, run /setup."))
	}
	if !cfg.FirstSetup {
		fmt.Println("\nCurrent model:", ui.Name(cfg.CurrentModel))
	}
	fmt.Println("\n" + ui.System(strings.Repeat("=", 50)))

	store, history, err := chat.NewThreadStore(cfg.CurrentModel)
	if err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to load thread store: %v", err)))
		return
	}
	fmt.Printf("Active thread: %s (%s)\n", ui.Thread(store.ActiveThreadTitle()), store.ActiveThreadID())

	cliTool := tools.GetCLITool()
	readTool := tools.GetReadFileTool()   // <--- New
	patchTool := tools.GetPatchFileTool() // <--- New
	applyUnifiedPatchTool := tools.GetApplyUnifiedDiffPatchTool()
	createCheckpointTool := tools.GetCreateCheckpointTool()
	undoCheckpointsTool := tools.GetUndoCheckpointsTool()
	editorHistoryTool := tools.GetEditorHistoryTool()
	cpuUsageSampleTool := tools.GetCPUUsageSampleTool()
	processSignalTool := tools.GetProcessSignalTool()
	pageSizeTool := tools.GetPageSizeTool()
	askUserTool := tools.GetAskUserTool()
	organizeMediaTool := tools.GetOrganizeMediaFilesTool()
	subagentFactoryTool := tools.GetSubagentFactoryTool()
	subagentContextTool := tools.GetSubagentContextProviderTool()
	projectArchitectTool := tools.GetProjectArchitectTool()
	toolsList := []api.Tool{cliTool, readTool, patchTool, applyUnifiedPatchTool, createCheckpointTool, undoCheckpointsTool, editorHistoryTool, cpuUsageSampleTool, processSignalTool, pageSizeTool, askUserTool, organizeMediaTool, subagentFactoryTool, subagentContextTool, projectArchitectTool}
	apiClient := api.NewClient(cfg)

	homeDir, _ := os.UserHomeDir()
	historyPath := filepath.Join(homeDir, ".config", "ai2go", "input_history.txt")
	_ = os.MkdirAll(filepath.Dir(historyPath), 0755)
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          ui.Prompt(history.GetTotalTokens(), cfg.CurrentModel, store.ActiveThreadTitle()),
		AutoComplete:    commands.NewAutoCompleter(),
		HistoryFile:     historyPath,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to initialize terminal UI: %v", err)))
		return
	}
	defer rl.Close()

	for {
		rl.SetPrompt(ui.Prompt(history.GetTotalTokens(), cfg.CurrentModel, store.ActiveThreadTitle()))
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			fmt.Println(ui.Warn("Input canceled."))
			continue
		}
		if err == io.EOF {
			fmt.Println(ui.System("Goodbye!"))
			break
		}
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Input error: %v", err)))
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Handle special commands
		if strings.HasPrefix(input, "/") && !strings.Contains(input, "/file") {
			commands.HandleCommand(input, history, store, cfg, apiClient)
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println(ui.System("Goodbye!"))
			break
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
				rl.SetPrompt(ui.System("draft> "))
				appendInput, readErr := rl.Readline()
				if readErr == readline.ErrInterrupt {
					fmt.Println(ui.Warn("Draft input canceled."))
					break
				}
				if readErr == io.EOF {
					fmt.Println(ui.System("Goodbye!"))
					return
				}
				if readErr != nil {
					fmt.Println(ui.Error(fmt.Sprintf("Input error: %v", readErr)))
					break
				}

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
		runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		fmt.Println(ui.System("Press Ctrl+C to stop current response/tools."))
		pauseCtrl := chat.NewPauseController()
		pauseSig := make(chan os.Signal, 1)
		chat.RegisterPauseSignal(pauseSig)
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				select {
				case <-runCtx.Done():
					return
				case <-pauseSig:
					if pauseCtrl.Toggle() {
						fmt.Println(ui.Warn("[System] Loop paused (Ctrl+Z). Press Ctrl+Z again to resume."))
					} else {
						fmt.Println(ui.System("[System] Loop resumed."))
					}
				}
			}
		}()

		chat.ProcessConversation(runCtx, history, toolsList, cfg, apiClient, pauseCtrl)
		chat.StopPauseSignal(pauseSig)
		stop()
		<-done
		if err := store.SyncActiveHistory(history); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Warning: failed to persist thread history: %v", err)))
		}
		commands.TryAutoSummarize(history, store, cfg, apiClient)
	}
}
