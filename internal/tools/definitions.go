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
