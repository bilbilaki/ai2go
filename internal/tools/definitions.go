package tools

import (
	"encoding/json"

	"github.com/bilbilaki/ai2go/internal/api"
)

func GetCLITool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "run_command",
			Description: "Executes a command in the Linux shell / terminal and returns the output.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {
						"type": "string",
						"description": "The shell command to run (e.g., 'ls -la', 'cat file.txt')"
					}
				},
				"required": ["command"]
			}`),
		},
	}
}

func GetReadFileTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "read_file",
			Description: "Reads a file and returns content with line numbers (e.g. '1 | package main'). REQUIRED before using patch_file.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": { "type": "string" }
				},
				"required": ["path"]
			}`),
		},
	}
}

func GetPatchFileTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "patch_file",
			Description: "Edits a file using line-based patches. Syntax: 'N--' (delete), 'N++ content' (replace), '0++' (prepend), '00++' (append).",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": { "type": "string" },
					"patch": { "type": "string", "description": "The patch string (e.g., '10++ new_code\\n20--')" }
				},
				"required": ["path", "patch"]
			}`),
		},
	}
}

func GetAskUserTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "ask_user",
			Description: "Ask the user a clarification question. Use when multiple solution paths exist or requirements are ambiguous.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"question": { "type": "string", "description": "Question to ask the user." },
					"options": {
						"type": "array",
						"items": { "type": "string" },
						"description": "Optional selectable choices. If omitted, user can answer freely."
					}
				},
				"required": ["question"]
			}`),
		},
	}
}

func GetOrganizeMediaFilesTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "organize_media_files",
			Description: "Groups loose media files into title folders using filename heuristics. Supports dry-run preview and apply mode.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"directory": { "type": "string", "description": "Directory containing media files." },
					"recursive": { "type": "boolean", "description": "Scan recursively. Default false." },
					"dry_run": { "type": "boolean", "description": "If true, preview moves only. Default true." },
					"min_group_size": { "type": "integer", "description": "Only create folders for groups with at least this many files. Default 2." },
					"max_preview_actions": { "type": "integer", "description": "Maximum move previews in output. Default 200." },
					"extensions": {
						"type": "array",
						"items": { "type": "string" },
						"description": "Optional media extensions override (e.g. [\".mkv\", \".mp4\"])."
					}
				},
				"required": ["directory"]
			}`),
		},
	}
}

func GetSubagentFactoryTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "subagent_factory",
			Description: "Splits a mega prompt into many tasks, runs subagents concurrently, writes per-task outputs + a report, and returns batch metadata.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"task_list_name": { "type": "string", "description": "Name used in generated task IDs (e.g. frontend_refactor)." },
					"mega_prompt": { "type": "string", "description": "A large prompt that contains multiple tasks." },
					"split_symbol": { "type": "string", "description": "Delimiter for splitting tasks (default: ---TASK---)." },
					"split_regex": { "type": "string", "description": "Optional regex splitter; overrides split_symbol if provided." },
					"base_instruction": { "type": "string", "description": "Optional instruction prepended to every task." },
					"max_concurrency": { "type": "integer", "description": "Concurrent subagents (1..200, default 3)." },
					"timeout_sec": { "type": "integer", "description": "Per-task timeout in seconds (default 600)." },
					"ttl_seconds": { "type": "integer", "description": "TTL for volatile task context store (default 600)." },
					"output_dir": { "type": "string", "description": "Optional output directory for results/report." },
					"model": { "type": "string", "description": "Optional model override for subagents." }
				},
				"required": ["mega_prompt"]
			}`),
		},
	}
}

func GetSubagentContextProviderTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "subagent_context_provider",
			Description: "Returns summarized volatile context for a subagent task ID; can consume/delete it after read.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"task_id": { "type": "string", "description": "Task ID generated by subagent_factory, e.g. 003_backend_fixes." },
					"consume": { "type": "boolean", "description": "Delete this task context after retrieval (default: true)." }
				},
				"required": ["task_id"]
			}`),
		},
	}
}

func GetProjectArchitectTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "project_architect",
			Description: "Builds a detailed multi-step execution plan from a rough project request using a fresh model context.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"prompt": {
						"type": "string",
						"description": "Raw project request to expand into implementation-ready tasks."
					}
				},
				"required": ["prompt"]
			}`),
		},
	}
}
