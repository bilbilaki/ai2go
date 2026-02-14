package tools

import (
	"encoding/json"

	"github.com/bilbilaki/ai2go/internal/api"
)

func GetProcessCPUTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "monitor_process_cpu",
			Description: "Monitors CPU usage percentage for one or more process IDs (PIDs). Returns current CPU usage as percentages.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"pids": {
						"type": "array",
						"items": { "type": "integer" },
						"description": "List of process IDs to monitor (e.g., [1234, 5678])"
					},
					"interval": {
						"type": "number",
						"description": "Monitoring interval in seconds (default: 1.0)"
					},
					"duration": {
						"type": "number",
						"description": "Total monitoring duration in seconds. 0 for single measurement, >0 for continuous monitoring"
					}
				},
				"required": ["pids"]
			}`),
		},
	}
}

func GetProcessStatsTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "get_process_stats",
			Description: "Gets detailed process statistics including CPU times, memory usage, and process state from /proc filesystem.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"pid": {
						"type": "integer",
						"description": "Process ID to get statistics for"
					},
					"include_memory": {
						"type": "boolean",
						"description": "Include memory usage information (default: true)"
					},
					"include_status": {
						"type": "boolean",
						"description": "Include process status information (default: true)"
					}
				},
				"required": ["pid"]
			}`),
		},
	}
}

func GetSystemCPUTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "get_system_cpu",
			Description: "Gets overall system CPU usage statistics including user, system, idle, and other CPU states.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"detailed": {
						"type": "boolean",
						"description": "Return detailed breakdown of CPU states (default: false)"
					},
					"cores": {
						"type": "boolean",
						"description": "Include per-core statistics if available (default: false)"
					}
				}
			}`),
		},
	}
}

func GetTopProcessesTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "get_top_processes",
			Description: "Finds and returns the top N processes by CPU or memory usage. Similar to 'top' or 'ps' command.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"count": {
						"type": "integer",
						"description": "Number of top processes to return (default: 10)"
					},
					"sort_by": {
						"type": "string",
						"enum": ["cpu", "memory", "pid", "name"],
						"description": "Sort processes by this metric (default: 'cpu')"
					},
					"user": {
						"type": "string",
						"description": "Filter processes by username (optional)"
					},
					"include_threads": {
						"type": "boolean",
						"description": "Include thread-level information (default: false)"
					}
				}
			}`),
		},
	}
}

func GetProcessTreeTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "get_process_tree",
			Description: "Gets the process tree hierarchy showing parent-child relationships for a given PID or all processes.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"pid": {
						"type": "integer",
						"description": "Root PID to show tree from. If omitted, shows full system process tree"
					},
					"max_depth": {
						"type": "integer",
						"description": "Maximum depth of tree to display (default: 10)"
					},
					"include_stats": {
						"type": "boolean",
						"description": "Include CPU/memory stats for each process (default: false)"
					}
				}
			}`),
		},
	}
}

func GetProcessFindTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "find_process",
			Description: "Finds processes by name, command line, or other attributes. Returns matching PIDs and details.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "Process name or substring to search for"
					},
					"command": {
						"type": "string",
						"description": "Command line substring to search for"
					},
					"user": {
						"type": "string",
						"description": "Filter by username"
					},
					"exact": {
						"type": "boolean",
						"description": "Match exact name instead of substring (default: false)"
					},
					"case_sensitive": {
						"type": "boolean",
						"description": "Case-sensitive search (default: false)"
					}
				}
			}`),
		},
	}
}

func GetProcessKillTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "kill_process",
			Description: "Sends signals to processes (terminate, kill, etc.). Supports graceful shutdown with timeout.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"pid": {
						"type": "integer",
						"description": "Process ID to send signal to"
					},
					"signal": {
						"type": "string",
						"enum": ["TERM", "KILL", "HUP", "INT", "USR1", "USR2"],
						"description": "Signal to send (default: 'TERM')"
					},
					"graceful_timeout": {
						"type": "integer",
						"description": "Seconds to wait for graceful termination before force kill (default: 5)"
					},
					"force": {
						"type": "boolean",
						"description": "Force kill immediately without graceful shutdown (default: false)"
					}
				},
				"required": ["pid"]
			}`),
		},
	}
}

func GetProcessStartTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "start_process",
			Description: "Starts a new process with given command and arguments. Returns PID and status.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {
						"type": "string",
						"description": "Command to execute"
					},
					"args": {
						"type": "array",
						"items": { "type": "string" },
						"description": "Command arguments"
					},
					"working_dir": {
						"type": "string",
						"description": "Working directory for the process"
					},
					"env": {
						"type": "object",
						"description": "Environment variables to set"
					},
					"detached": {
						"type": "boolean",
						"description": "Run process in background/detached (default: false)"
					},
					"timeout": {
						"type": "integer",
						"description": "Timeout in seconds for command execution (default: 0 = no timeout)"
					}
				},
				"required": ["command"]
			}`),
		},
	}
}
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

func GetApplyUnifiedDiffPatchTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "apply_unified_diff_patch",
			Description: "Applies a standard unified diff patch to a worktree using the editor git engine with checkpoint + rollback safety.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"work_tree": { "type": "string", "description": "Target project directory/worktree." },
					"patch": { "type": "string", "description": "Unified diff content (git diff format)." },
					"verify_mode": {
						"type": "string",
						"enum": ["none", "syntax", "tests"],
						"description": "Post-apply verification mode. Default: none."
					}
				},
				"required": ["work_tree", "patch"]
			}`),
		},
	}
}

func GetCreateCheckpointTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "create_checkpoint",
			Description: "Creates a git-backed editor checkpoint commit for a worktree.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"work_tree": { "type": "string", "description": "Target project directory/worktree." },
					"file_path": { "type": "string", "description": "Optional file path to stage; if omitted stages all changes." },
					"message": { "type": "string", "description": "Checkpoint commit message." }
				},
				"required": ["work_tree"]
			}`),
		},
	}
}

func GetUndoCheckpointsTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "undo_checkpoints",
			Description: "Undo the last N editor checkpoints in a worktree.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"work_tree": { "type": "string", "description": "Target project directory/worktree." },
					"steps": { "type": "integer", "description": "Number of checkpoints to undo. Default: 1." }
				},
				"required": ["work_tree"]
			}`),
		},
	}
}

func GetEditorHistoryTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "editor_history",
			Description: "Shows recent editor checkpoint history for a worktree.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"work_tree": { "type": "string", "description": "Target project directory/worktree." },
					"limit": { "type": "integer", "description": "Number of history entries to return. Default: 10." }
				},
				"required": ["work_tree"]
			}`),
		},
	}
}

func GetPageSizeTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "get_page_size",
			Description: "Returns OS memory page size in bytes.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
	}
}

func GetProcessSignalTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "send_process_signal",
			Description: "Sends a signal to a process tree with optional graceful timeout.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"pid": { "type": "integer", "description": "Root process ID." },
					"signal": { "type": "string", "description": "Signal name or number (default TERM)." },
					"graceful_timeout": { "type": "integer", "description": "Graceful timeout in seconds before force kill. Default 0 (no forced retry)." },
					"force": { "type": "boolean", "description": "Force immediate KILL if true." }
				},
				"required": ["pid"]
			}`),
		},
	}
}

func GetCPUUsageSampleTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "get_process_cpu_usage_sample",
			Description: "Samples and returns CPU usage percent for given PIDs.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"pids": {
						"type": "array",
						"items": { "type": "integer" },
						"description": "Process IDs to sample."
					},
					"as_integer": {
						"type": "boolean",
						"description": "Return rounded integer percentages if true."
					}
				},
				"required": ["pids"]
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
