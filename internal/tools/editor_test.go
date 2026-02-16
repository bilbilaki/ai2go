package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileWithLinesUsesRaisedLineLimit(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "many-lines.txt")

	var b strings.Builder
	for i := 1; i <= maxReadFileLines+5; i++ {
		b.WriteString(fmt.Sprintf("line-%d\n", i))
	}
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	out, err := ReadFileWithLines(path, "")
	if err != nil {
		t.Fatalf("ReadFileWithLines returned error: %v", err)
	}

	if !strings.Contains(out, fmt.Sprintf("%d | line-%d", maxReadFileLines, maxReadFileLines)) {
		t.Fatalf("expected output to include line %d", maxReadFileLines)
	}
	if strings.Contains(out, fmt.Sprintf("%d | line-%d", maxReadFileLines+1, maxReadFileLines+1)) {
		t.Fatalf("expected output to be truncated before line %d", maxReadFileLines+1)
	}
	if !strings.Contains(out, fmt.Sprintf("line limit (%d)", maxReadFileLines)) {
		t.Fatalf("expected truncation notice to include line limit %d", maxReadFileLines)
	}
}

func TestEditorGitCheckpointHistoryAndUndo(t *testing.T) {
	workTree := t.TempDir()
	file := filepath.Join(workTree, "note.txt")
	if err := os.WriteFile(file, []byte("one\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	first, err := CreateCheckpoint(workTree, "", "first")
	if err != nil {
		t.Fatalf("CreateCheckpoint(first): %v", err)
	}
	if first == "" {
		t.Fatal("expected first commit hash")
	}

	if err := os.WriteFile(file, []byte("two\n"), 0644); err != nil {
		t.Fatalf("write updated fixture: %v", err)
	}
	second, err := CreateCheckpoint(workTree, "", "second")
	if err != nil {
		t.Fatalf("CreateCheckpoint(second): %v", err)
	}
	if second == "" || second == first {
		t.Fatal("expected distinct second commit hash")
	}

	history, err := EditorHistory(workTree, 5)
	if err != nil {
		t.Fatalf("EditorHistory: %v", err)
	}
	if !strings.Contains(history, "second") || !strings.Contains(history, "first") {
		t.Fatalf("unexpected history output: %q", history)
	}

	head, err := UndoLastCheckpoints(workTree, 1)
	if err != nil {
		t.Fatalf("UndoLastCheckpoints: %v", err)
	}
	if head != first {
		t.Fatalf("expected head %s after undo, got %s", first, head)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read note after undo: %v", err)
	}
	if string(content) != "one\n" {
		t.Fatalf("expected file rollback to original content, got %q", string(content))
	}
}

func TestApplyUnifiedDiffPatchRollsBackOnVerifyFailure(t *testing.T) {
	workTree := t.TempDir()
	if err := os.WriteFile(filepath.Join(workTree, "go.mod"), []byte("module example.com/editor\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	mainPath := filepath.Join(workTree, "main.go")
	original := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(mainPath, []byte(original), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	patch := strings.Join([]string{
		"--- a/main.go",
		"+++ b/main.go",
		"@@ -1,3 +1,3 @@",
		" package main",
		" ",
		"-func main() {}",
		"+func main( {}",
		"",
	}, "\n")

	_, err := ApplyUnifiedDiffPatch(workTree, patch, VerifyModeSyntax)
	if err == nil {
		t.Fatal("expected verification failure")
	}

	content, readErr := os.ReadFile(mainPath)
	if readErr != nil {
		t.Fatalf("read main.go: %v", readErr)
	}
	if string(content) != original {
		t.Fatalf("expected rollback to restore original file; got %q", string(content))
	}

	history, historyErr := EditorHistory(workTree, 10)
	if historyErr != nil {
		t.Fatalf("EditorHistory: %v", historyErr)
	}
	if strings.Contains(history, "post-apply") {
		t.Fatalf("unexpected post-apply checkpoint after rollback: %q", history)
	}
}
