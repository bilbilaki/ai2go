package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/config"
	"github.com/bilbilaki/ai2go/internal/subagent"
	"github.com/bilbilaki/ai2go/internal/tools"
	"github.com/bilbilaki/ai2go/internal/ui"
)

func ProcessConversation(ctx context.Context, history *History, toolsList []api.Tool, cfg *config.Config, apiClient *api.Client, pauseCtrl *PauseController) {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		if err := pauseCtrl.WaitIfPaused(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println(ui.Warn("[System] Request canceled by user."))
				return
			}
			fmt.Printf("\nError while paused: %v\n", err)
			return
		}

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
		if store != nil && chatID != 0 {
			if err := store.SaveMessage(chatID, assistantMsg.Role, assistantMsg.Content); err != nil {
				fmt.Printf("\n[Warning] Failed to save assistant message: %v\n", err)
			}
		}

		// If the AI didn't call any tools, we are done with this turn
		if len(assistantMsg.ToolCalls) == 0 {
			break
		}

		// Process tool calls
		for _, tCall := range assistantMsg.ToolCalls {
			if err := pauseCtrl.WaitIfPaused(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					fmt.Println(ui.Warn("[System] Request canceled by user."))
					return
				}
				fmt.Printf("\nError while paused: %v\n", err)
				return
			}

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
					toolContent = fmt.Sprintf("Error: %v", err)
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

			case "apply_unified_diff_patch":
				var args map[string]string
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for apply_unified_diff_patch: %v", err)
					break
				}
				workTree := strings.TrimSpace(args["work_tree"])
				patch := args["patch"]
				verifyMode := tools.VerifyMode(strings.TrimSpace(args["verify_mode"]))
				if verifyMode == "" {
					verifyMode = tools.VerifyModeNone
				}
				if workTree == "" {
					toolResponse = "Error: apply_unified_diff_patch requires a non-empty 'work_tree' argument."
					break
				}
				if strings.TrimSpace(patch) == "" {
					toolResponse = "Error: apply_unified_diff_patch requires a non-empty 'patch' argument."
					break
				}
				output, err := tools.ApplyUnifiedDiffPatch(workTree, patch, verifyMode)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: %v", err)
				} else {
					toolResponse = output
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "create_checkpoint":
				var args map[string]string
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for create_checkpoint: %v", err)
					break
				}
				workTree := strings.TrimSpace(args["work_tree"])
				if workTree == "" {
					toolResponse = "Error: create_checkpoint requires a non-empty 'work_tree' argument."
					break
				}
				head, err := tools.CreateCheckpoint(workTree, strings.TrimSpace(args["file_path"]), strings.TrimSpace(args["message"]))
				if err != nil {
					toolResponse = fmt.Sprintf("Error: %v", err)
				} else {
					toolResponse = fmt.Sprintf("Checkpoint created: %s", head)
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "undo_checkpoints":
				var args map[string]any
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for undo_checkpoints: %v", err)
					break
				}
				workTree, _ := args["work_tree"].(string)
				workTree = strings.TrimSpace(workTree)
				if workTree == "" {
					toolResponse = "Error: undo_checkpoints requires a non-empty 'work_tree' argument."
					break
				}
				steps := 1
				if raw, ok := args["steps"]; ok {
					switch v := raw.(type) {
					case float64:
						steps = int(v)
					case string:
						if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
							steps = n
						}
					}
				}
				head, err := tools.UndoLastCheckpoints(workTree, steps)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: %v", err)
				} else {
					toolResponse = fmt.Sprintf("Undo complete. HEAD=%s", head)
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "editor_history":
				var args map[string]any
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for editor_history: %v", err)
					break
				}
				workTree, _ := args["work_tree"].(string)
				workTree = strings.TrimSpace(workTree)
				if workTree == "" {
					toolResponse = "Error: editor_history requires a non-empty 'work_tree' argument."
					break
				}
				limit := 10
				if raw, ok := args["limit"]; ok {
					switch v := raw.(type) {
					case float64:
						limit = int(v)
					case string:
						if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
							limit = n
						}
					}
				}
				output, err := tools.EditorHistory(workTree, limit)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: %v", err)
				} else {
					toolResponse = output
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "get_process_cpu_usage_sample":
				var args map[string]any
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for get_process_cpu_usage_sample: %v", err)
					break
				}
				rawPids, _ := args["pids"].([]any)
				if len(rawPids) == 0 {
					toolResponse = "Error: get_process_cpu_usage_sample requires non-empty 'pids'."
					break
				}
				pids := make([]int, 0, len(rawPids))
				for _, p := range rawPids {
					if f, ok := p.(float64); ok {
						pids = append(pids, int(f))
					}
				}
				asInteger, _ := args["as_integer"].(bool)
				if asInteger {
					vals, err := tools.GetProcessCPUUsageSimple(pids)
					if err != nil {
						toolResponse = fmt.Sprintf("Error: %v", err)
					} else if blob, mErr := json.Marshal(vals); mErr == nil {
						toolResponse = string(blob)
					}
				} else {
					vals, err := tools.GetProcessCPUUsage(pids)
					if err != nil {
						toolResponse = fmt.Sprintf("Error: %v", err)
					} else if blob, mErr := json.Marshal(vals); mErr == nil {
						toolResponse = string(blob)
					}
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "send_process_signal":
				var args map[string]any
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for send_process_signal: %v", err)
					break
				}
				pidF, ok := args["pid"].(float64)
				if !ok {
					toolResponse = "Error: send_process_signal requires integer 'pid'."
					break
				}
				signalName, _ := args["signal"].(string)
				signalName = strings.TrimSpace(signalName)
				if signalName == "" {
					signalName = "TERM"
				}
				grace := 0
				if g, ok := args["graceful_timeout"].(float64); ok {
					grace = int(g)
				}
				force, _ := args["force"].(bool)
				err := tools.KillProcessTreeWithTimeout(int(pidF), signalName, grace, force)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: %v", err)
				} else {
					toolResponse = fmt.Sprintf("Signal handling completed for pid=%d", int(pidF))
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "get_page_size":
				toolResponse = fmt.Sprintf("%d", os.Getpagesize())
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "ask_user":
				question, options, err := parseAskUserArgs(tCall.Function.Arguments)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for ask_user: %v", err)
					break
				}

				answer, selectedIdx := askUserForClarification(question, options)
				payload := map[string]any{
					"question": question,
					"answer":   answer,
				}
				if selectedIdx >= 0 && selectedIdx < len(options) {
					payload["selected_option_index"] = selectedIdx
					payload["selected_option"] = options[selectedIdx]
				}
				blob, _ := json.Marshal(payload)
				toolResponse = string(blob)
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "organize_media_files":
				output, err := tools.OrganizeMediaFiles(tCall.Function.Arguments)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: organize_media_files failed: %v", err)
				} else {
					toolResponse = output
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "subagent_factory":
				if !cfg.SubagentExperimental {
					toolResponse = "Error: subagent experimental mode is OFF. Run /subagent_experimental to enable it."
					break
				}

				input, err := subagent.ParseFactoryInput(tCall.Function.Arguments)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for subagent_factory: %v", err)
					break
				}

				systemPrompt := extractSystemPrompt(history.GetMessages())
				fmt.Printf("\n%s\n", ui.Tool("[Subagent] Starting subagent batch..."))
				report, err := subagent.DefaultManager().RunFactory(ctx, apiClient, cfg.CurrentModel, systemPrompt, input, cfg.SubagentExperimental)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: subagent_factory failed: %v", err)
				} else {
					toolResponse = subagent.FormatBatchReport(report)
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "subagent_context_provider":
				taskID, consume, err := subagent.ParseContextProviderInput(tCall.Function.Arguments)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for subagent_context_provider: %v", err)
					break
				}

				output, err := subagent.DefaultManager().GetTaskContextSummary(taskID, consume)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: %v", err)
				} else {
					toolResponse = output
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			case "project_architect":
				var args map[string]string
				if err := json.Unmarshal([]byte(tCall.Function.Arguments), &args); err != nil {
					toolResponse = fmt.Sprintf("Error: invalid arguments for project_architect: %v", err)
					break
				}
				rawPrompt := strings.TrimSpace(args["prompt"])
				if rawPrompt == "" {
					toolResponse = "Error: project_architect requires a non-empty 'prompt' argument."
					break
				}

				fmt.Printf("\n%s\n", ui.Tool("[Project Architect] Building detailed execution plan..."))
				output, err := tools.BuildProjectArchitecturePlan(ctx, apiClient, cfg.CurrentModel, rawPrompt)
				if err != nil {
					toolResponse = fmt.Sprintf("Error: project_architect failed: %v", err)
				} else {
					toolResponse = output
				}
				fmt.Printf("%s\n%s\n----------------\n", ui.Tool("[Output]"), toolResponse)

			default:
				toolResponse = fmt.Sprintf("Error: unsupported tool '%s'", tCall.Function.Name)
			}

			history.AddToolResponse(tCall.ID, toolResponse)
		}
	}
}

func extractSystemPrompt(messages []api.Message) string {
	for _, msg := range messages {
		if msg.Role == "system" && strings.TrimSpace(msg.Content) != "" {
			return msg.Content
		}
	}
	return ""
}

func parseAskUserArgs(raw string) (string, []string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return "", nil, err
	}

	question, _ := args["question"].(string)
	question = strings.TrimSpace(question)
	if question == "" {
		return "", nil, fmt.Errorf("ask_user requires a non-empty 'question'")
	}

	options := make([]string, 0)
	if arr, ok := args["options"].([]any); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					options = append(options, s)
				}
			}
		}
	}

	return question, options, nil
}

func askUserForClarification(question string, options []string) (string, int) {
	fmt.Printf("\n%s\n", ui.Tool("[Clarification Required]"))
	fmt.Printf("%s\n", question)
	if len(options) > 0 {
		fmt.Println("Options:")
		for i, opt := range options {
			fmt.Printf("  %d. %s\n", i+1, opt)
		}
		fmt.Print("Your answer (number or custom text): ")
	} else {
		fmt.Print("Your answer: ")
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		raw, _ := reader.ReadString('\n')
		text := strings.TrimSpace(raw)
		if text == "" {
			fmt.Print("Please enter a response: ")
			continue
		}

		if len(options) > 0 {
			if n, err := strconv.Atoi(text); err == nil {
				if n >= 1 && n <= len(options) {
					return options[n-1], n - 1
				}
				fmt.Print("Invalid option number. Enter a valid number or custom text: ")
				continue
			}
		}

		return text, -1
	}
}
