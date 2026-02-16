package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func extractBackupID(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "backup_id="); idx >= 0 {
			return strings.TrimSpace(line[idx+len("backup_id="):])
		}
	}
	return ""
}

func TestFileBackupRestoreAndDiff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	backupOut, err := CreateFileBackup(path)
	if err != nil {
		t.Fatalf("CreateFileBackup: %v", err)
	}
	backupID := extractBackupID(backupOut)
	if backupID == "" {
		t.Fatalf("missing backup id in output: %q", backupOut)
	}

	if err := os.WriteFile(path, []byte("one\nTWO\n"), 0644); err != nil {
		t.Fatalf("write modified file: %v", err)
	}

	diffOut, err := ShowFileDiff(path, "", backupID)
	if err != nil {
		t.Fatalf("ShowFileDiff: %v", err)
	}
	if !strings.Contains(diffOut, "+two") && !strings.Contains(diffOut, "-TWO") {
		t.Fatalf("unexpected diff output: %q", diffOut)
	}

	restoreOut, err := RestoreFileBackup(path, backupID)
	if err != nil {
		t.Fatalf("RestoreFileBackup: %v", err)
	}
	if !strings.Contains(restoreOut, "Restored") {
		t.Fatalf("unexpected restore output: %q", restoreOut)
	}

	finalBlob, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(finalBlob) != "one\ntwo\n" {
		t.Fatalf("unexpected restored content: %q", string(finalBlob))
	}
}

func TestCompareMergeAndTypeDetect(t *testing.T) {
	dir := t.TempDir()
	left := filepath.Join(dir, "left.txt")
	right := filepath.Join(dir, "right.txt")
	base := filepath.Join(dir, "base.txt")
	if err := os.WriteFile(left, []byte("aa\nbb\ncc\n"), 0644); err != nil {
		t.Fatalf("write left: %v", err)
	}
	if err := os.WriteFile(right, []byte("aa\nBB\ncc\n"), 0644); err != nil {
		t.Fatalf("write right: %v", err)
	}
	if err := os.WriteFile(base, []byte("aa\nbb\ncc\n"), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}

	cmpOut, err := CompareFilesSideBySide(left, right, 80)
	if err != nil {
		t.Fatalf("CompareFilesSideBySide: %v", err)
	}
	if !strings.Contains(cmpOut, "|") {
		t.Fatalf("expected change marker in side-by-side output: %q", cmpOut)
	}

	mergeOut, err := MergeFiles(base, left, right, "")
	if err != nil {
		t.Fatalf("MergeFiles: %v", err)
	}
	if !strings.Contains(mergeOut, "conflicts=0") {
		t.Fatalf("unexpected merge output: %q", mergeOut)
	}

	typeOut, err := DetectFileType(left)
	if err != nil {
		t.Fatalf("DetectFileType: %v", err)
	}
	if !strings.Contains(typeOut, "mime:") {
		t.Fatalf("unexpected detect output: %q", typeOut)
	}
}
