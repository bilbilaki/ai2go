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
			Description: "Reads a file and returns content with line numbers (e.g. '1 | package main'). Supports reading all lines or a specific range. REQUIRED before using patch_file.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {
						"type": "string",
						"description": "Path to the file to read"
					},
					"line_range": {
						"type": "string",
						"description": "Optional line range in format 'start-end' (e.g., '400-600'). If omitted, reads entire file."
					}
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
func GetSearchFilesTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "search_files",
			Description: "Advanced concurrent file search. Filter by extension, path/name regex, and content regex.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"dir": { "type": "string", "description": "Root directory to start search" },
					"ext": { "type": "string", "description": "Comma-separated extensions (e.g. '.go,.js')" },
					"inc_path": { "type": "string", "description": "Include path patterns (e.g. 'src/*')" },
					"exc_path": { "type": "string", "description": "Exclude path patterns (e.g. '.git/*,node_modules/*')" },
					"content": { "type": "string", "description": "Regex to search inside file content" },
					"content_exclude": { "type": "boolean", "description": "If true, exclude files matching content regex" }
				},
				"required": ["dir"]
			}`),
		},
	}
}

// GetListTreeTool defines the schema for generating a visual directory structure.
func GetListTreeTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "list_tree",
			Description: "Generates a visual ASCII tree of the directory structure.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"dir": { "type": "string", "description": "Directory to visualize" }
				},
				"required": ["dir"]
			}`),
		},
	}
}
