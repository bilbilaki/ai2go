package tools

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxReadFileLines = 1000
	maxReadFileChars = 15000
)

func looksBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	limit := len(data)
	if limit > 8192 {
		limit = 8192
	}

	controlCount := 0
	for _, b := range data[:limit] {
		if b == 0 {
			return true
		}
		if b < 0x09 || (b > 0x0D && b < 0x20) {
			controlCount++
		}
	}

	return float64(controlCount)/float64(limit) > 0.10
}

func sanitizeText(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "?")
}

func truncationNotice(path string, totalLines, totalChars int, lineLimited, charLimited bool) string {
	reasons := make([]string, 0, 2)
	if lineLimited {
		reasons = append(reasons, fmt.Sprintf("line limit (%d)", maxReadFileLines))
	}
	if charLimited {
		reasons = append(reasons, fmt.Sprintf("char limit (%d)", maxReadFileChars))
	}
	return fmt.Sprintf("\n... [READ TRUNCATED: %s | file=%s | read_lines=%d read_chars=%d] ...\n", strings.Join(reasons, ", "), path, totalLines, totalChars)
}
