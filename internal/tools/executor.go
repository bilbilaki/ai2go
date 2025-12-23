package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

func ExecuteShellCommand(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	result := string(output)

	if err != nil {
		result += fmt.Sprintf("\nError: %s", err.Error())
	}

	if strings.TrimSpace(result) == "" {
		result = "(Command executed successfully with no output)"
	}

	// Smart Truncation with hint
	maxLength := 4000
	if len(result) > maxLength {
		truncatedLen := len(result) - maxLength
		result = result[:maxLength] + 
			fmt.Sprintf("\n\n... [OUTPUT TRUNCATED - %d more characters] ...\n"+
			"SYSTEM HINT: The output is too long. DO NOT ask the user to read it.\n"+
			"INSTEAD: Run the command again using 'grep', 'head -n 20', 'tail -n 20', or 'sed' to filter the output.", truncatedLen)
	}

	return result, nil
}
