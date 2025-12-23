package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ResolveFileTokens scans the input for "/file", prompts the user to select files,
// and replaces the token with the file content.
func ResolveFileTokens(input string) string {
	for strings.Contains(input, "/file") {
		fmt.Println("\n\033[33m[Attachment System] Found '/file' marker.\033[0m")
		
		// 1. Ask for search term
		fmt.Print("Enter filename or search term (or 'skip'): ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		searchTerm := strings.TrimSpace(scanner.Text())

		if searchTerm == "" || searchTerm == "skip" {
			// Remove the marker if skipped so we don't loop forever
			input = strings.Replace(input, "/file", "(attachment skipped)", 1)
			continue
		}

		// 2. Find matches
		matches := findFiles(searchTerm)
		if len(matches) == 0 {
			fmt.Println("\033[31mNo files found. Try again.\033[0m")
			continue // Loop keeps the /file token so user can try again
		}

		// 3. Show list
		selectedPath := ""
		if len(matches) == 1 {
			selectedPath = matches[0]
			fmt.Printf("Auto-selected: %s\n", selectedPath)
		} else {
			fmt.Println("Found multiple files:")
			for i, path := range matches {
				fmt.Printf("[%d] %s\n", i+1, path)
			}
			fmt.Print("Select file number (0 to cancel): ")
			scanner.Scan()
			selection, _ := strconv.Atoi(scanner.Text())
			if selection > 0 && selection <= len(matches) {
				selectedPath = matches[selection-1]
			} else {
				fmt.Println("Selection cancelled.")
				input = strings.Replace(input, "/file", "(attachment cancelled)", 1)
				continue
			}
		}

		// 4. Read content and replace
		content, err := os.ReadFile(selectedPath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			continue
		}

		// Format: text before \n user attached this file from this path "$content" \n text after
		// We use a distinct format to ensure the AI sees it clearly.
		attachmentBlock := fmt.Sprintf("\n[User attached this file from path: %s]\n```\n%s\n```\n", selectedPath, string(content))
		
		// Replace only the *first* occurrence of /file
		input = strings.Replace(input, "/file", attachmentBlock, 1)
		fmt.Println("\033[32mFile attached successfully.\033[0m")
	}
	return input
}

func findFiles(term string) []string {
	var matches []string
	// Walk the current directory recursively
	filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.Contains(strings.ToLower(path), strings.ToLower(term)) {
			// Ignore hidden files (like .git)
			if !strings.Contains(path, ".git") {
				matches = append(matches, path)
			}
		}
		return nil
	})
	return matches
}