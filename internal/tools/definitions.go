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