package tools

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// VerifyMode controls post-patch verification behavior.
type VerifyMode string

const (
	VerifyModeNone   VerifyMode = "none"
	VerifyModeSyntax VerifyMode = "syntax"
	VerifyModeTests  VerifyMode = "tests"
)

// ReadFileWithLines returns content with line numbers (e.g., "1 | package main").
// lineRange format: "start-end" (e.g., "400-600"), empty string reads full file.
func ReadFileWithLines(path, lineRange string) (string, error) {
	sampleFile, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	sample := make([]byte, 8192)
	n, readErr := sampleFile.Read(sample)
	_ = sampleFile.Close()
	if readErr != nil && readErr != io.EOF {
		return "", fmt.Errorf("failed to inspect file: %w", readErr)
	}
	sample = sample[:n]

	if looksBinary(sample) {
		return fmt.Sprintf("Refused to read %s: detected binary/non-text content. Use tools like 'file', 'strings', or targeted commands.", path), nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var startLine, endLine int

	// Parse line range if provided
	if lineRange != "" {
		parts := strings.Split(lineRange, "-")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid line range format, use 'start-end' (e.g., '400-600')")
		}

		var errStart, errEnd error
		startLine, errStart = strconv.Atoi(strings.TrimSpace(parts[0]))
		endLine, errEnd = strconv.Atoi(strings.TrimSpace(parts[1]))

		if errStart != nil || errEnd != nil || startLine < 1 || endLine < startLine {
			return "", fmt.Errorf("invalid line range: start and end must be positive integers with start <= end")
		}
	} else {
		// Read all lines: set a very large end value
		startLine = 1
		endLine = int(^uint(0) >> 1) // Max int value
	}

	var result strings.Builder
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 1
	totalChars := 0
	lineLimited := false
	charLimited := false
	for scanner.Scan() {
		if lineNum > maxReadFileLines {
			lineLimited = true
			break
		}

		line := sanitizeText(scanner.Text())
		lineChars := len([]rune(line))
		if totalChars+lineChars > maxReadFileChars {
			remaining := maxReadFileChars - totalChars
			if remaining > 0 {
				runes := []rune(line)
				if remaining < len(runes) {
					line = string(runes[:remaining])
				}
				result.WriteString(fmt.Sprintf("%d | %s\n", lineNum, line))
				totalChars += len([]rune(line))
				lineNum++
			}
			charLimited = true
			break
		}

		// Format: 1 | <content>
		result.WriteString(fmt.Sprintf("%d | %s\n", lineNum, line))
		totalChars += lineChars
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		if err == bufio.ErrTooLong {
			result.WriteString(fmt.Sprintf("\n... [READ TRUNCATED: line too long while reading %s] ...\n", path))
			return result.String(), nil
		}
		if err != io.EOF {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
	}

	if lineLimited || charLimited {
		result.WriteString(truncationNotice(path, lineNum-1, totalChars, lineLimited, charLimited))
	}

	if strings.TrimSpace(result.String()) == "" {
		return "(Empty text file)", nil
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
	// Regex matches: "26" or "0" or "00", then "++", "--", "<<", ">>", then optional content
	re := regexp.MustCompile(`^(\d+|00)(\+\+|--|<<|>>)\s?(.*)$`)

	type Operation struct {
		Type  string // "delete", "replace", "insert_before", "insert_after"
		Lines []string
	}
	ops := make(map[string]Operation)

	scanner := bufio.NewScanner(strings.NewReader(patchContent))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		target := matches[1] // Line number, "0", or "00"
		operator := matches[2]
		text := strings.ReplaceAll(matches[3], `\n`, "\n")
		lines := []string{}
		if text != "" {
			lines = strings.Split(text, "\n")
		}

		switch operator {
		case "--":
			ops[target] = Operation{Type: "delete"}
		case "++":
			if target == "0" {
				ops[target] = Operation{Type: "insert_before", Lines: lines}
			} else if target == "00" {
				ops[target] = Operation{Type: "insert_after", Lines: lines}
			} else {
				ops[target] = Operation{Type: "replace", Lines: lines}
			}
		case "<<":
			ops[target] = Operation{Type: "insert_before", Lines: lines}
		case ">>":
			ops[target] = Operation{Type: "insert_after", Lines: lines}
		}
	}

	// 3. Reconstruct Content
	var newLines []string

	// Handle Prepend (0<<)
	if op, ok := ops["0"]; ok && op.Type == "insert_before" {
		newLines = append(newLines, op.Lines...)
	}

	// Process Original Lines
	for i, line := range originalLines {
		lineNumStr := strconv.Itoa(i + 1)

		if op, ok := ops[lineNumStr]; ok {
			if op.Type == "delete" {
				continue // Skip this line
			}
			if op.Type == "replace" {
				newLines = append(newLines, op.Lines...)
			} else {
				newLines = append(newLines, line)
			}
		} else {
			newLines = append(newLines, line)
		}

		if op, ok := ops[lineNumStr]; ok && op.Type == "insert_after" {
			newLines = append(newLines, op.Lines...)
		}
	}

	// Handle Append (00>>)
	if op, ok := ops["00"]; ok && op.Type == "insert_after" {
		newLines = append(newLines, op.Lines...)
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

// EnsureEditorGitRepo creates (if needed) and configures a per-worktree git repo under user cache.
func EnsureEditorGitRepo(workTree string) error {
	if workTree == "" {
		return fmt.Errorf("workTree is required")
	}
	absWorkTree, err := filepath.Abs(workTree)
	if err != nil {
		return fmt.Errorf("failed to resolve worktree path: %w", err)
	}
	if err := os.MkdirAll(absWorkTree, 0755); err != nil {
		return fmt.Errorf("failed to ensure worktree exists: %w", err)
	}

	gitDir, err := editorGitDir(absWorkTree)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(gitDir), 0755); err != nil {
		return fmt.Errorf("failed to create editor cache: %w", err)
	}
	if _, statErr := os.Stat(gitDir); os.IsNotExist(statErr) {
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			return fmt.Errorf("failed to create git dir: %w", err)
		}
		if _, err := runGit(absWorkTree, "init", "--quiet"); err != nil {
			return err
		}
	}

	_, err = runGit(absWorkTree, "config", "user.name", "ai2go-editor")
	if err != nil {
		return err
	}
	_, err = runGit(absWorkTree, "config", "user.email", "editor@ai2go.local")
	if err != nil {
		return err
	}

	return nil
}

// CreateCheckpoint creates a commit checkpoint for a file or the whole worktree when filePath is empty.
func CreateCheckpoint(workTree, filePath, message string) (string, error) {
	if err := EnsureEditorGitRepo(workTree); err != nil {
		return "", err
	}

	if strings.TrimSpace(message) == "" {
		message = "editor checkpoint"
	}

	if strings.TrimSpace(filePath) == "" {
		if _, err := runGit(workTree, "add", "-A"); err != nil {
			return "", err
		}
	} else {
		if _, err := runGit(workTree, "add", "--", filePath); err != nil {
			return "", err
		}
	}

	if _, err := runGit(workTree, "commit", "--allow-empty", "-m", message); err != nil {
		return "", err
	}

	head, err := runGit(workTree, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(head), nil
}

// ApplyUnifiedDiffPatch applies a unified diff patch with checkpointing and optional verification.
func ApplyUnifiedDiffPatch(workTree, patchContent string, verifyMode VerifyMode) (string, error) {
	pre, err := CreateCheckpoint(workTree, "", "editor checkpoint: pre-apply")
	if err != nil {
		return "", fmt.Errorf("failed to create pre-apply checkpoint: %w", err)
	}

	if _, err := runGitWithInput(workTree, patchContent, "apply", "--whitespace=nowarn", "-"); err != nil {
		_ = rollbackTo(workTree, pre)
		return "", fmt.Errorf("failed to apply unified diff (rolled back): %w", err)
	}

	if err := runVerification(workTree, verifyMode); err != nil {
		_ = rollbackTo(workTree, pre)
		return "", fmt.Errorf("verification failed and changes were rolled back: %w", err)
	}

	post, err := CreateCheckpoint(workTree, "", "editor checkpoint: post-apply")
	if err != nil {
		_ = rollbackTo(workTree, pre)
		return "", fmt.Errorf("failed to create post-apply checkpoint; rolled back: %w", err)
	}

	return fmt.Sprintf("Patch applied successfully. Checkpoints: pre=%s post=%s", pre, post), nil
}

// UndoLastCheckpoints undoes the last N checkpoints.
func UndoLastCheckpoints(workTree string, steps int) (string, error) {
	if steps < 1 {
		return "", fmt.Errorf("steps must be >= 1")
	}
	if err := EnsureEditorGitRepo(workTree); err != nil {
		return "", err
	}

	target := fmt.Sprintf("HEAD~%d", steps)
	if _, err := runGit(workTree, "reset", "--hard", target); err != nil {
		return "", err
	}
	if _, err := runGit(workTree, "clean", "-fd"); err != nil {
		return "", err
	}
	head, err := runGit(workTree, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(head), nil
}

// EditorHistory returns latest N checkpoints as `sha message` lines.
func EditorHistory(workTree string, n int) (string, error) {
	if n < 1 {
		return "", fmt.Errorf("n must be >= 1")
	}
	if err := EnsureEditorGitRepo(workTree); err != nil {
		return "", err
	}

	out, err := runGit(workTree, "log", fmt.Sprintf("-%d", n), "--pretty=format:%h %s")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == "" {
		return "(No editor checkpoints yet)", nil
	}
	return out, nil
}

func editorGitDir(workTree string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate user cache dir: %w", err)
	}
	hash := sha256.Sum256([]byte(workTree))
	return filepath.Join(cacheDir, "ai2go", "editor", hex.EncodeToString(hash[:]), "git"), nil
}

func runGit(workTree string, args ...string) (string, error) {
	return runGitWithInput(workTree, "", args...)
}

func runGitWithInput(workTree, input string, args ...string) (string, error) {
	absWorkTree, err := filepath.Abs(workTree)
	if err != nil {
		return "", fmt.Errorf("failed to resolve worktree path: %w", err)
	}
	gitDir, err := editorGitDir(absWorkTree)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = absWorkTree
	cmd.Env = append(os.Environ(),
		"GIT_DIR="+gitDir,
		"GIT_WORK_TREE="+absWorkTree,
	)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func runVerification(workTree string, mode VerifyMode) error {
	switch mode {
	case VerifyModeNone, "":
		return nil
	case VerifyModeSyntax:
		_, err := runCommandInDir(workTree, "go", "test", "./...", "-run=^$")
		return err
	case VerifyModeTests:
		_, err := runCommandInDir(workTree, "go", "test", "./...")
		return err
	default:
		return fmt.Errorf("unsupported verify mode: %s", mode)
	}
}

func rollbackTo(workTree, commit string) error {
	if _, err := runGit(workTree, "reset", "--hard", commit); err != nil {
		return err
	}
	if _, err := runGit(workTree, "clean", "-fd"); err != nil {
		return err
	}
	return nil
}

func runCommandInDir(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s failed: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}
