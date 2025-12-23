package tools

import (
	"context"
	"fmt"
	"strings"
)

// ExecuteShellCommand runs a command cross-platform.
// It uses 'prepareCommand' (defined in os_*.go files) to handle OS differences.
func ExecuteShellCommand(ctx context.Context, command string) (string, error) {
	
	// 1. Get the OS-specific command struct
	cmd := prepareCommand(ctx, command)

	// 2. Run it
	output, err := cmd.CombinedOutput()
	result := string(output)

	// 3. Handle Interrupts
	if ctx.Err() == context.Canceled {
		result += "\n\n[SYSTEM: Command execution was interrupted by the user via Ctrl+C.]"
		return result, nil
	}

	if err != nil {
		result += fmt.Sprintf("\nError: %s", err.Error())
	}

	if strings.TrimSpace(result) == "" {
		result = "(Command executed successfully with no output)"
	}

	// 4. Smart Truncation
	maxLength := 4000
	if len(result) > maxLength {
		truncatedLen := len(result) - maxLength
		result = result[:maxLength] + 
			fmt.Sprintf("\n\n... [OUTPUT TRUNCATED - %d more characters] ...\n"+
			"SYSTEM HINT: The output is too long. DO NOT ask the user to read it.\n"+
			"INSTEAD: Run the command again using filters (like 'grep' on Linux or 'findstr' on Windows).", truncatedLen)
	}

	return result, nil
}