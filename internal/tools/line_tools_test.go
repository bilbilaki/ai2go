package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLineToolsCoreFlow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	initial := strings.Join([]string{
		"alpha",
		"beta",
		"gamma",
		"beta",
		"delta",
		"", // keep one blank line
		"gamma",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if _, err := RemoveLineRanges(path, []LineRange{{Start: 2, End: 2}}); err != nil {
		t.Fatalf("RemoveLineRanges: %v", err)
	}

	if _, err := ReplaceLineRange(path, 2, 2, "GAMMA"); err != nil {
		t.Fatalf("ReplaceLineRange: %v", err)
	}

	if _, err := ReorderLineRange(path, 1, 1, 4); err != nil {
		t.Fatalf("ReorderLineRange: %v", err)
	}

	extracted, err := ExtractLineRange(path, 1, 3)
	if err != nil {
		t.Fatalf("ExtractLineRange: %v", err)
	}
	if !strings.Contains(extracted, "1 | GAMMA") {
		t.Fatalf("unexpected extract output: %q", extracted)
	}

	if _, err := DeleteLinesByPattern(path, "^delta$", true); err != nil {
		t.Fatalf("DeleteLinesByPattern: %v", err)
	}

	if _, err := RemoveDuplicateLines(path, false, true); err != nil {
		t.Fatalf("RemoveDuplicateLines: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read final file: %v", err)
	}
	got := string(content)
	if strings.Contains(got, "delta") {
		t.Fatalf("expected delta removed, got %q", got)
	}
	if strings.Count(got, "gamma") > 1 {
		t.Fatalf("expected duplicate gamma removed, got %q", got)
	}
}

func TestBatchLineOperations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "batch.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	ops := []BatchLineOperation{
		{Op: "insert_before", Line: 1, Text: "zero"},
		{Op: "replace", Line: 3, EndLine: 3, Text: "TWO"},
		{Op: "delete", Line: 4, EndLine: 4},
	}
	if _, err := ApplyBatchLineOperations(path, ops); err != nil {
		t.Fatalf("ApplyBatchLineOperations: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	got := string(content)
	if got != "zero\none\nTWO\n" {
		t.Fatalf("unexpected batch result: %q", got)
	}
}
