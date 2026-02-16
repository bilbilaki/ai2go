package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type LineRange struct {
	Start int
	End   int
}

type BatchLineOperation struct {
	Op      string `json:"op"`
	Line    int    `json:"line"`
	EndLine int    `json:"end_line"`
	Text    string `json:"text"`
}

func parseRangeSpec(spec string) (LineRange, error) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return LineRange{}, fmt.Errorf("empty range")
	}
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		if len(parts) != 2 {
			return LineRange{}, fmt.Errorf("invalid range format: %s", spec)
		}
		start, errStart := strconv.Atoi(strings.TrimSpace(parts[0]))
		end, errEnd := strconv.Atoi(strings.TrimSpace(parts[1]))
		if errStart != nil || errEnd != nil || start < 1 || end < start {
			return LineRange{}, fmt.Errorf("invalid range values: %s", spec)
		}
		return LineRange{Start: start, End: end}, nil
	}
	line, err := strconv.Atoi(s)
	if err != nil || line < 1 {
		return LineRange{}, fmt.Errorf("invalid line number: %s", spec)
	}
	return LineRange{Start: line, End: line}, nil
}

func readTextLines(path string) ([]string, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read file: %w", err)
	}
	hadTrailingNewline := strings.HasSuffix(string(content), "\n")
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, hadTrailingNewline, nil
}

func writeTextLines(path string, lines []string, hadTrailingNewline bool) error {
	out := strings.Join(lines, "\n")
	if hadTrailingNewline || len(lines) > 0 {
		out += "\n"
	}
	if err := os.WriteFile(path, []byte(out), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func normalizeRange(start, end, max int) (int, int, error) {
	if start < 1 || end < start {
		return 0, 0, fmt.Errorf("invalid line range %d-%d", start, end)
	}
	if max == 0 {
		return 0, 0, fmt.Errorf("target file is empty")
	}
	if start > max {
		return 0, 0, fmt.Errorf("range start %d exceeds file length %d", start, max)
	}
	if end > max {
		end = max
	}
	return start, end, nil
}

func RemoveLineRanges(path string, ranges []LineRange) (string, error) {
	if len(ranges) == 0 {
		return "Error: at least one range is required.", nil
	}
	lines, hadTrailingNewline, err := readTextLines(path)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "No changes. File is empty.", nil
	}

	del := make(map[int]struct{})
	for _, r := range ranges {
		start, end, err := normalizeRange(r.Start, r.End, len(lines))
		if err != nil {
			return "", err
		}
		for i := start; i <= end; i++ {
			del[i-1] = struct{}{}
		}
	}

	newLines := make([]string, 0, len(lines)-len(del))
	for i, line := range lines {
		if _, ok := del[i]; ok {
			continue
		}
		newLines = append(newLines, line)
	}

	if err := writeTextLines(path, newLines, hadTrailingNewline); err != nil {
		return "", err
	}
	return fmt.Sprintf("Removed %d line(s) in %s.", len(del), path), nil
}

func ReplaceLineRange(path string, start, end int, replacement string) (string, error) {
	lines, hadTrailingNewline, err := readTextLines(path)
	if err != nil {
		return "", err
	}
	start, end, err = normalizeRange(start, end, len(lines))
	if err != nil {
		return "", err
	}

	newBlock := []string{}
	if replacement != "" {
		replacement = strings.ReplaceAll(replacement, "\r\n", "\n")
		newBlock = strings.Split(replacement, "\n")
	}

	newLines := make([]string, 0, len(lines)-(end-start+1)+len(newBlock))
	newLines = append(newLines, lines[:start-1]...)
	newLines = append(newLines, newBlock...)
	if end < len(lines) {
		newLines = append(newLines, lines[end:]...)
	}

	if err := writeTextLines(path, newLines, hadTrailingNewline); err != nil {
		return "", err
	}
	return fmt.Sprintf("Replaced lines %d-%d in %s.", start, end, path), nil
}

func ApplyBatchLineOperations(path string, ops []BatchLineOperation) (string, error) {
	if len(ops) == 0 {
		return "Error: operations cannot be empty.", nil
	}
	lines, hadTrailingNewline, err := readTextLines(path)
	if err != nil {
		return "", err
	}

	applyInsert := func(before bool, line int, text string) error {
		if line < 1 || line > len(lines) {
			return fmt.Errorf("invalid insert line %d for file length %d", line, len(lines))
		}
		parts := []string{}
		if text != "" {
			text = strings.ReplaceAll(text, "\r\n", "\n")
			parts = strings.Split(text, "\n")
		}
		if len(parts) == 0 {
			return nil
		}
		idx := line - 1
		if !before {
			idx = line
		}
		if idx >= len(lines) {
			lines = append(lines, parts...)
			return nil
		}
		prefix := append([]string{}, lines[:idx]...)
		suffix := append([]string{}, lines[idx:]...)
		prefix = append(prefix, parts...)
		lines = append(prefix, suffix...)
		return nil
	}

	for i, op := range ops {
		name := strings.ToLower(strings.TrimSpace(op.Op))
		switch name {
		case "delete":
			end := op.EndLine
			if end == 0 {
				end = op.Line
			}
			start, end, err := normalizeRange(op.Line, end, len(lines))
			if err != nil {
				return "", fmt.Errorf("operation %d: %w", i+1, err)
			}
			lines = append(lines[:start-1], lines[end:]...)
		case "replace":
			end := op.EndLine
			if end == 0 {
				end = op.Line
			}
			start, end, err := normalizeRange(op.Line, end, len(lines))
			if err != nil {
				return "", fmt.Errorf("operation %d: %w", i+1, err)
			}
			repl := []string{}
			if op.Text != "" {
				txt := strings.ReplaceAll(op.Text, "\r\n", "\n")
				repl = strings.Split(txt, "\n")
			}
			lines = append(lines[:start-1], append(repl, lines[end:]...)...)
		case "insert_before":
			if err := applyInsert(true, op.Line, op.Text); err != nil {
				return "", fmt.Errorf("operation %d: %w", i+1, err)
			}
		case "insert_after":
			if err := applyInsert(false, op.Line, op.Text); err != nil {
				return "", fmt.Errorf("operation %d: %w", i+1, err)
			}
		default:
			return "", fmt.Errorf("operation %d: unsupported op %q", i+1, op.Op)
		}
	}

	if err := writeTextLines(path, lines, hadTrailingNewline); err != nil {
		return "", err
	}
	return fmt.Sprintf("Applied %d batch operation(s) to %s.", len(ops), path), nil
}

func DeleteLinesByPattern(path, pattern string, caseSensitive bool) (string, error) {
	if strings.TrimSpace(pattern) == "" {
		return "Error: pattern is required.", nil
	}
	lines, hadTrailingNewline, err := readTextLines(path)
	if err != nil {
		return "", err
	}
	if !caseSensitive {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	newLines := make([]string, 0, len(lines))
	removed := 0
	for _, line := range lines {
		if re.MatchString(line) {
			removed++
			continue
		}
		newLines = append(newLines, line)
	}

	if err := writeTextLines(path, newLines, hadTrailingNewline); err != nil {
		return "", err
	}
	return fmt.Sprintf("Removed %d line(s) matching pattern in %s.", removed, path), nil
}

func ExtractLineRange(path string, start, end int) (string, error) {
	lines, _, err := readTextLines(path)
	if err != nil {
		return "", err
	}
	start, end, err = normalizeRange(start, end, len(lines))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i := start; i <= end; i++ {
		b.WriteString(fmt.Sprintf("%d | %s\n", i, lines[i-1]))
	}
	if b.Len() == 0 {
		return "(No lines extracted)", nil
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func ReorderLineRange(path string, start, end, targetLine int) (string, error) {
	lines, hadTrailingNewline, err := readTextLines(path)
	if err != nil {
		return "", err
	}
	start, end, err = normalizeRange(start, end, len(lines))
	if err != nil {
		return "", err
	}
	if targetLine < 1 || targetLine > len(lines)+1 {
		return "", fmt.Errorf("target_line %d is out of range 1..%d", targetLine, len(lines)+1)
	}
	if targetLine >= start && targetLine <= end+1 {
		return "No changes. target_line already points inside the selected range.", nil
	}

	block := append([]string{}, lines[start-1:end]...)
	rest := append([]string{}, lines[:start-1]...)
	rest = append(rest, lines[end:]...)

	insertIdx := targetLine - 1
	if targetLine > end {
		insertIdx -= (end - start + 1)
	}
	if insertIdx < 0 {
		insertIdx = 0
	}
	if insertIdx > len(rest) {
		insertIdx = len(rest)
	}

	reordered := append([]string{}, rest[:insertIdx]...)
	reordered = append(reordered, block...)
	reordered = append(reordered, rest[insertIdx:]...)

	if err := writeTextLines(path, reordered, hadTrailingNewline); err != nil {
		return "", err
	}
	return fmt.Sprintf("Moved lines %d-%d to before line %d in %s.", start, end, targetLine, path), nil
}

func RemoveDuplicateLines(path string, caseSensitive bool, ignoreBlank bool) (string, error) {
	lines, hadTrailingNewline, err := readTextLines(path)
	if err != nil {
		return "", err
	}
	seen := make(map[string]struct{}, len(lines))
	removed := 0
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if ignoreBlank && strings.TrimSpace(line) == "" {
			result = append(result, line)
			continue
		}
		key := line
		if !caseSensitive {
			key = strings.ToLower(key)
		}
		if _, ok := seen[key]; ok {
			removed++
			continue
		}
		seen[key] = struct{}{}
		result = append(result, line)
	}
	if err := writeTextLines(path, result, hadTrailingNewline); err != nil {
		return "", err
	}
	return fmt.Sprintf("Removed %d duplicate line(s) from %s.", removed, path), nil
}

func ExecuteLineTool(name string, rawArgs string) (handled bool, output string) {
	args := map[string]any{}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return true, fmt.Sprintf("Error: invalid arguments for %s: %v", name, err)
	}

	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return true, fmt.Sprintf("Error: %s requires a non-empty 'path' argument.", name)
	}

	switch name {
	case "remove_lines":
		rawRanges, _ := args["ranges"].([]any)
		ranges := make([]LineRange, 0, len(rawRanges))
		for _, item := range rawRanges {
			spec, _ := item.(string)
			r, err := parseRangeSpec(spec)
			if err != nil {
				return true, fmt.Sprintf("Error: invalid range %q: %v", spec, err)
			}
			ranges = append(ranges, r)
		}
		if len(ranges) == 0 {
			if startF, ok := args["start_line"].(float64); ok {
				end := int(startF)
				if endF, ok := args["end_line"].(float64); ok {
					end = int(endF)
				}
				ranges = append(ranges, LineRange{Start: int(startF), End: end})
			}
		}
		out, err := RemoveLineRanges(path, ranges)
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out

	case "replace_line_range":
		startF, okStart := args["start_line"].(float64)
		endF, okEnd := args["end_line"].(float64)
		replacement, _ := args["replacement"].(string)
		if !okStart || !okEnd {
			return true, "Error: replace_line_range requires numeric 'start_line' and 'end_line'."
		}
		out, err := ReplaceLineRange(path, int(startF), int(endF), replacement)
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out

	case "batch_line_operations":
		rawOps, _ := args["operations"].([]any)
		if len(rawOps) == 0 {
			return true, "Error: batch_line_operations requires non-empty 'operations'."
		}
		ops := make([]BatchLineOperation, 0, len(rawOps))
		for i, rawOp := range rawOps {
			obj, ok := rawOp.(map[string]any)
			if !ok {
				return true, fmt.Sprintf("Error: operation %d is not an object.", i+1)
			}
			op := BatchLineOperation{}
			if v, ok := obj["op"].(string); ok {
				op.Op = v
			}
			if v, ok := obj["line"].(float64); ok {
				op.Line = int(v)
			}
			if v, ok := obj["end_line"].(float64); ok {
				op.EndLine = int(v)
			}
			if v, ok := obj["text"].(string); ok {
				op.Text = v
			}
			ops = append(ops, op)
		}
		out, err := ApplyBatchLineOperations(path, ops)
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out

	case "delete_lines_by_pattern":
		pattern, _ := args["pattern"].(string)
		caseSensitive, _ := args["case_sensitive"].(bool)
		out, err := DeleteLinesByPattern(path, pattern, caseSensitive)
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out

	case "extract_line_range":
		startF, okStart := args["start_line"].(float64)
		endF, okEnd := args["end_line"].(float64)
		if !okStart || !okEnd {
			return true, "Error: extract_line_range requires numeric 'start_line' and 'end_line'."
		}
		out, err := ExtractLineRange(path, int(startF), int(endF))
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out

	case "reorder_line_range":
		startF, okStart := args["start_line"].(float64)
		endF, okEnd := args["end_line"].(float64)
		targetF, okTarget := args["target_line"].(float64)
		if !okStart || !okEnd || !okTarget {
			return true, "Error: reorder_line_range requires numeric 'start_line', 'end_line', and 'target_line'."
		}
		out, err := ReorderLineRange(path, int(startF), int(endF), int(targetF))
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out

	case "remove_duplicate_lines":
		caseSensitive, _ := args["case_sensitive"].(bool)
		ignoreBlank, _ := args["ignore_blank"].(bool)
		out, err := RemoveDuplicateLines(path, caseSensitive, ignoreBlank)
		if err != nil {
			return true, fmt.Sprintf("Error: %v", err)
		}
		return true, out
	default:
		return false, ""
	}
}

func parseRangesCSV(csv string) ([]LineRange, error) {
	items := strings.Split(csv, ",")
	out := make([]LineRange, 0, len(items))
	for _, item := range items {
		r, err := parseRangeSpec(item)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Start == out[j].Start {
			return out[i].End < out[j].End
		}
		return out[i].Start < out[j].Start
	})
	return out, nil
}

func ParseLineRanges(raw string) ([]LineRange, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("ranges cannot be empty")
	}
	raw = strings.ReplaceAll(raw, "\n", ",")
	raw = strings.ReplaceAll(raw, ";", ",")
	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Split(bufio.ScanLines)
	chunks := make([]string, 0)
	for scanner.Scan() {
		chunks = append(chunks, scanner.Text())
	}
	joined := strings.Join(chunks, ",")
	return parseRangesCSV(joined)
}
