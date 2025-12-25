package tools

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ReadFileWithLines returns content with line numbers (e.g., "1 | package main").
func ReadFileWithLines(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var result strings.Builder
	scanner := bufio.NewScanner(file)
	lineNum := 1
	for scanner.Scan() {
		// Format: 1 | <content>
		result.WriteString(fmt.Sprintf("%d | %s\n", lineNum, scanner.Text()))
		lineNum++
	}
	return result.String(), nil
}

// ApplyFilePatch applies the custom "26++" / "26--" syntax.
// It uses original line numbers to ensure stability.
func ApplyFilePatch(path, patchContent string) (string, error) {
	// 1. Read the original file
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	originalLines := strings.Split(string(content), "\n")
	
	// Fix split edge case: if file ends with newline, Split gives an empty string at the end.
	if len(originalLines) > 0 && originalLines[len(originalLines)-1] == "" {
		originalLines = originalLines[:len(originalLines)-1]
	}

	// 2. Parse Patch
	// Regex matches: "26" or "0" or "00", then "++" or "--", then optional content
	re := regexp.MustCompile(`^(\d+|00)(\+\+|--)\s?(.*)$`)

	type Operation struct {
		Type    string // "delete", "replace", "prepend", "append"
		Content string
	}
	ops := make(map[string]Operation)

	scanner := bufio.NewScanner(strings.NewReader(patchContent))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" { continue }
		
		matches := re.FindStringSubmatch(line)
		if len(matches) < 3 { continue }

		target := matches[1] // Line number, "0", or "00"
		operator := matches[2]
		text := matches[3]

		if operator == "--" {
			ops[target] = Operation{Type: "delete"}
		} else {
			ops[target] = Operation{Type: "replace", Content: text}
		}
	}

	// 3. Reconstruct Content
	var newLines []string

	// Handle Prepend (0++)
	if op, ok := ops["0"]; ok && op.Type != "delete" {
		newLines = append(newLines, op.Content)
	}

	// Process Original Lines
	for i, line := range originalLines {
		lineNumStr := strconv.Itoa(i + 1)
		
		if op, ok := ops[lineNumStr]; ok {
			if op.Type == "delete" {
				continue // Skip this line
			} else if op.Type == "replace" {
				newLines = append(newLines, op.Content)
			}
		} else {
			newLines = append(newLines, line)
		}
	}

	// Handle Append (00++)
	if op, ok := ops["00"]; ok && op.Type != "delete" {
		newLines = append(newLines, op.Content)
	}

	// 4. Write to disk
	finalContent := strings.Join(newLines, "\n")
	// Ensure single trailing newline
	if !strings.HasSuffix(finalContent, "\n") {
		finalContent += "\n"
	}

	err = os.WriteFile(path, []byte(finalContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return fmt.Sprintf("Successfully patched %s. Check content to verify.", path), nil
}