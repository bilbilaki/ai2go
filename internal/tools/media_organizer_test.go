package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInferMediaGroupName(t *testing.T) {
	got := inferMediaGroupName("[SubsPlease] One.Piece.-.1089.1080p.WEB-DL.mkv")
	if got == "" {
		t.Fatal("expected non-empty group")
	}
	if !strings.Contains(strings.ToLower(got), "one") || !strings.Contains(strings.ToLower(got), "piece") {
		t.Fatalf("unexpected group: %q", got)
	}
}

func TestOrganizeMediaFilesDryRun(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"My.Hero.Academia.S01E01.1080p.mkv",
		"My.Hero.Academia.S01E02.1080p.mkv",
		"Random.Movie.2020.1080p.mkv",
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}
	}

	raw := `{"directory":"` + strings.ReplaceAll(dir, `\`, `\\`) + `","dry_run":true,"min_group_size":2}`
	out, err := OrganizeMediaFiles(raw)
	if err != nil {
		t.Fatalf("OrganizeMediaFiles failed: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if int(res["planned_moves"].(float64)) < 2 {
		t.Fatalf("expected planned moves >= 2, got %v", res["planned_moves"])
	}
	_, err = os.Stat(filepath.Join(dir, "My Hero Academia"))
	if !os.IsNotExist(err) {
		t.Fatalf("dry run should not create directories, stat err=%v", err)
	}
}
