package tools

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxFileDiffOutputLines = 500
	defaultCompareWidth    = 120
)

type fileBackupMeta struct {
	BackupID   string `json:"backup_id"`
	SourcePath string `json:"source_path"`
	BackupPath string `json:"backup_path"`
	CreatedAt  string `json:"created_at"`
	SizeBytes  int64  `json:"size_bytes"`
}

func readLinesNoEOL(path string) ([]string, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	txt := strings.ReplaceAll(string(blob), "\r\n", "\n")
	if strings.HasSuffix(txt, "\n") {
		txt = strings.TrimSuffix(txt, "\n")
	}
	if txt == "" {
		return []string{}, nil
	}
	return strings.Split(txt, "\n"), nil
}

func commonPrefixLines(a, b []string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func commonSuffixLines(a, b []string, prefix int) int {
	i := len(a) - 1
	j := len(b) - 1
	count := 0
	for i >= prefix && j >= prefix {
		if a[i] != b[j] {
			break
		}
		count++
		i--
		j--
	}
	return count
}

func BuildSimpleUnifiedDiff(fromLabel, toLabel string, fromLines, toLines []string) string {
	prefix := commonPrefixLines(fromLines, toLines)
	suffix := commonSuffixLines(fromLines, toLines, prefix)

	fromStart := prefix
	toStart := prefix
	fromEnd := len(fromLines) - suffix
	toEnd := len(toLines) - suffix
	if fromEnd < fromStart {
		fromEnd = fromStart
	}
	if toEnd < toStart {
		toEnd = toStart
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("--- %s\n", fromLabel))
	out.WriteString(fmt.Sprintf("+++ %s\n", toLabel))
	out.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", fromStart+1, fromEnd-fromStart, toStart+1, toEnd-toStart))

	written := 0
	for i := fromStart; i < fromEnd; i++ {
		out.WriteString("-")
		out.WriteString(fromLines[i])
		out.WriteString("\n")
		written++
		if written >= maxFileDiffOutputLines {
			out.WriteString("... [DIFF TRUNCATED] ...\n")
			return strings.TrimRight(out.String(), "\n")
		}
	}
	for i := toStart; i < toEnd; i++ {
		out.WriteString("+")
		out.WriteString(toLines[i])
		out.WriteString("\n")
		written++
		if written >= maxFileDiffOutputLines {
			out.WriteString("... [DIFF TRUNCATED] ...\n")
			return strings.TrimRight(out.String(), "\n")
		}
	}

	if fromStart == fromEnd && toStart == toEnd {
		out.WriteString("(No differences)\n")
	}
	return strings.TrimRight(out.String(), "\n")
}

func resolveBackupRoot() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(cacheDir) == "" {
		cacheDir = os.TempDir()
	}
	root := filepath.Join(cacheDir, "ai2go", "file_backups")
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}
	return root, nil
}

func backupPathID(sourcePath string) string {
	sum := sha1.Sum([]byte(sourcePath))
	return hex.EncodeToString(sum[:])[:10]
}

func sanitizeFileName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "file"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	s = replacer.Replace(s)
	if len(s) > 80 {
		s = s[len(s)-80:]
	}
	return s
}

func CreateFileBackup(path string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return "", fmt.Errorf("path is required")
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	blob, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	root, err := resolveBackupRoot()
	if err != nil {
		return "", err
	}

	base := sanitizeFileName(filepath.Base(abs))
	id := fmt.Sprintf("%s__%s__%d", base, backupPathID(abs), time.Now().UTC().UnixNano())
	backupPath := filepath.Join(root, id+".bak")
	metaPath := filepath.Join(root, id+".meta.json")

	if err := os.WriteFile(backupPath, blob, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}
	meta := fileBackupMeta{
		BackupID:   id,
		SourcePath: abs,
		BackupPath: backupPath,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		SizeBytes:  int64(len(blob)),
	}
	metaBlob, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaPath, metaBlob, 0644)
	return fmt.Sprintf("Backup created. backup_id=%s\nbackup_file=%s", id, backupPath), nil
}

func resolveBackupByID(backupID string) (fileBackupMeta, error) {
	id := strings.TrimSpace(backupID)
	if id == "" {
		return fileBackupMeta{}, fmt.Errorf("backup_id is required")
	}
	root, err := resolveBackupRoot()
	if err != nil {
		return fileBackupMeta{}, err
	}
	metaPath := filepath.Join(root, id+".meta.json")
	metaBlob, err := os.ReadFile(metaPath)
	if err == nil {
		meta := fileBackupMeta{}
		if uErr := json.Unmarshal(metaBlob, &meta); uErr == nil {
			if strings.TrimSpace(meta.BackupPath) != "" {
				return meta, nil
			}
		}
	}

	backupPath := filepath.Join(root, id+".bak")
	if _, statErr := os.Stat(backupPath); statErr == nil {
		return fileBackupMeta{BackupID: id, BackupPath: backupPath}, nil
	}
	return fileBackupMeta{}, fmt.Errorf("backup id not found: %s", id)
}

func RestoreFileBackup(path, backupID string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return "", fmt.Errorf("path is required")
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	meta, err := resolveBackupByID(backupID)
	if err != nil {
		return "", err
	}
	blob, err := os.ReadFile(meta.BackupPath)
	if err != nil {
		return "", fmt.Errorf("failed to read backup file: %w", err)
	}
	if err := os.WriteFile(abs, blob, 0644); err != nil {
		return "", fmt.Errorf("failed to restore backup: %w", err)
	}
	return fmt.Sprintf("Restored %s from backup_id=%s", abs, meta.BackupID), nil
}

func ShowFileDiff(path, comparePath, backupID string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return "", fmt.Errorf("path is required")
	}
	absPath, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	lhs, err := readLinesNoEOL(absPath)
	if err != nil {
		return "", err
	}

	var rhs []string
	fromLabel := absPath
	toLabel := ""

	if strings.TrimSpace(backupID) != "" {
		meta, err := resolveBackupByID(backupID)
		if err != nil {
			return "", err
		}
		rhs, err = readLinesNoEOL(meta.BackupPath)
		if err != nil {
			return "", err
		}
		toLabel = meta.BackupPath
	} else {
		other := strings.TrimSpace(comparePath)
		if other == "" {
			return "", fmt.Errorf("compare_path or backup_id is required")
		}
		absOther, err := filepath.Abs(other)
		if err != nil {
			return "", fmt.Errorf("failed to resolve compare_path: %w", err)
		}
		rhs, err = readLinesNoEOL(absOther)
		if err != nil {
			return "", err
		}
		toLabel = absOther
	}

	return BuildSimpleUnifiedDiff(fromLabel, toLabel, lhs, rhs), nil
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func padRightRunes(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return string(r[:width])
	}
	return s + strings.Repeat(" ", width-len(r))
}

func CompareFilesSideBySide(leftPath, rightPath string, width int) (string, error) {
	leftAbs, err := filepath.Abs(strings.TrimSpace(leftPath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve left_path: %w", err)
	}
	rightAbs, err := filepath.Abs(strings.TrimSpace(rightPath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve right_path: %w", err)
	}
	left, err := readLinesNoEOL(leftAbs)
	if err != nil {
		return "", err
	}
	right, err := readLinesNoEOL(rightAbs)
	if err != nil {
		return "", err
	}

	if width <= 0 {
		width = defaultCompareWidth
	}
	if width < 60 {
		width = 60
	}
	colWidth := (width - 7) / 2
	if colWidth < 20 {
		colWidth = 20
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("LEFT: %s\nRIGHT: %s\n", leftAbs, rightAbs))
	b.WriteString(strings.Repeat("-", colWidth*2+7) + "\n")

	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	for i := 0; i < n; i++ {
		l := ""
		r := ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		marker := " "
		if l != r {
			marker = "|"
		}
		l = padRightRunes(truncateRunes(l, colWidth), colWidth)
		r = padRightRunes(truncateRunes(r, colWidth), colWidth)
		b.WriteString(fmt.Sprintf("%4d %s %s %s\n", i+1, l, marker, r))
		if i+1 >= maxFileDiffOutputLines {
			b.WriteString("... [COMPARISON TRUNCATED] ...\n")
			break
		}
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func MergeFiles(basePath, leftPath, rightPath, outputPath string) (string, error) {
	baseAbs, err := filepath.Abs(strings.TrimSpace(basePath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve base_path: %w", err)
	}
	leftAbs, err := filepath.Abs(strings.TrimSpace(leftPath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve left_path: %w", err)
	}
	rightAbs, err := filepath.Abs(strings.TrimSpace(rightPath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve right_path: %w", err)
	}

	base, err := readLinesNoEOL(baseAbs)
	if err != nil {
		return "", err
	}
	left, err := readLinesNoEOL(leftAbs)
	if err != nil {
		return "", err
	}
	right, err := readLinesNoEOL(rightAbs)
	if err != nil {
		return "", err
	}

	n := len(base)
	if len(left) > n {
		n = len(left)
	}
	if len(right) > n {
		n = len(right)
	}

	pick := func(lines []string, idx int) string {
		if idx >= 0 && idx < len(lines) {
			return lines[idx]
		}
		return ""
	}

	merged := make([]string, 0, n)
	conflicts := 0
	for i := 0; i < n; i++ {
		b := pick(base, i)
		l := pick(left, i)
		r := pick(right, i)

		switch {
		case l == r:
			merged = append(merged, l)
		case l == b:
			merged = append(merged, r)
		case r == b:
			merged = append(merged, l)
		default:
			conflicts++
			merged = append(merged,
				"<<<<<<< LEFT",
				l,
				"||||||| BASE",
				b,
				"=======",
				r,
				">>>>>>> RIGHT",
			)
		}
	}

	out := strings.TrimSpace(outputPath)
	if out == "" {
		out = baseAbs + ".merged"
	}
	outAbs, err := filepath.Abs(out)
	if err != nil {
		return "", fmt.Errorf("failed to resolve output_path: %w", err)
	}

	content := strings.Join(merged, "\n")
	if len(merged) > 0 {
		content += "\n"
	}
	if err := os.WriteFile(outAbs, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write merge output: %w", err)
	}
	return fmt.Sprintf("Merge completed. output=%s conflicts=%d", outAbs, conflicts), nil
}

func DetectFileType(path string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return "", fmt.Errorf("path is required")
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	blob, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	sample := blob
	if len(sample) > 512 {
		sample = sample[:512]
	}
	mime := http.DetectContentType(sample)
	isBinary := looksBinary(sample)
	encoding := "utf-8"
	if isBinary {
		encoding = "binary"
	} else if !utf8.Valid(blob) {
		encoding = "non-utf8-text"
	}

	newline := "none"
	s := string(blob)
	switch {
	case strings.Contains(s, "\r\n"):
		newline = "crlf"
	case strings.Contains(s, "\n"):
		newline = "lf"
	}

	res := map[string]any{
		"path":          abs,
		"size_bytes":    len(blob),
		"extension":     strings.ToLower(filepath.Ext(abs)),
		"mime":          mime,
		"is_binary":     isBinary,
		"encoding":      encoding,
		"newline_style": newline,
	}
	keys := make([]string, 0, len(res))
	for k := range res {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("%s: %v\n", k, res[k]))
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func ExecuteFileManagementTool(name string, rawArgs string) (handled bool, output string) {
	args := map[string]any{}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return true, fmt.Sprintf("Error: invalid arguments for %s: %v", name, err)
	}

	getStr := func(key string) string {
		v, _ := args[key].(string)
		return strings.TrimSpace(v)
	}
	getInt := func(key string, def int) int {
		raw, ok := args[key]
		if !ok || raw == nil {
			return def
		}
		switch v := raw.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			n, err := strconv.Atoi(strings.TrimSpace(v))
			if err == nil {
				return n
			}
		}
		return def
	}

	switch name {
	case "show_file_diff":
		out, err := ShowFileDiff(getStr("path"), getStr("compare_path"), getStr("backup_id"))
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out
	case "compare_files_side_by_side":
		left := getStr("left_path")
		right := getStr("right_path")
		if left == "" || right == "" {
			return true, "Error: compare_files_side_by_side requires non-empty 'left_path' and 'right_path'."
		}
		out, err := CompareFilesSideBySide(left, right, getInt("width", defaultCompareWidth))
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out
	case "create_file_backup":
		out, err := CreateFileBackup(getStr("path"))
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out
	case "restore_file_backup":
		out, err := RestoreFileBackup(getStr("path"), getStr("backup_id"))
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out
	case "merge_files":
		out, err := MergeFiles(getStr("base_path"), getStr("left_path"), getStr("right_path"), getStr("output_path"))
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out
	case "detect_file_type":
		out, err := DetectFileType(getStr("path"))
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out
	default:
		return false, ""
	}
}
