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

	out, err := ReadFileWithLines(path)
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
