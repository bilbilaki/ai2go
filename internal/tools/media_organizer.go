package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type MediaOrganizeInput struct {
	Directory         string   `json:"directory"`
	Recursive         bool     `json:"recursive"`
	DryRun            bool     `json:"dry_run"`
	MinGroupSize      int      `json:"min_group_size"`
	MaxPreviewActions int      `json:"max_preview_actions"`
	Extensions        []string `json:"extensions"`
}

type mediaMoveAction struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Group string `json:"group"`
}

type mediaOrganizeResult struct {
	Directory       string            `json:"directory"`
	DryRun          bool              `json:"dry_run"`
	Recursive       bool              `json:"recursive"`
	TotalMediaFiles int               `json:"total_media_files"`
	DetectedGroups  int               `json:"detected_groups"`
	EligibleGroups  int               `json:"eligible_groups"`
	PlannedMoves    int               `json:"planned_moves"`
	Moved           int               `json:"moved"`
	Skipped         int               `json:"skipped"`
	TopGroups       []groupCount      `json:"top_groups"`
	OmittedGroups   int               `json:"omitted_groups"`
	PreviewActions  []mediaMoveAction `json:"preview_actions"`
	Warnings        []string          `json:"warnings,omitempty"`
}

type groupCount struct {
	Group string `json:"group"`
	Count int    `json:"count"`
}

type mediaFile struct {
	absPath  string
	baseName string
}

var (
	reBracketContent = regexp.MustCompile(`[\[\(\{].*?[\]\)\}]`)
	reMultiSpace     = regexp.MustCompile(`\s+`)
	reSeasonEpisode  = regexp.MustCompile(`(?i)\bS\d{1,2}E\d{1,3}\b`)
	reEpisodeOnly    = regexp.MustCompile(`(?i)\bE[Pp]?\s?\d{1,4}\b`)
	reResolution     = regexp.MustCompile(`(?i)\b(360p|480p|720p|1080p|1440p|2160p|4k|8k)\b`)
	reCodec          = regexp.MustCompile(`(?i)\b(x264|x265|h\.?264|h\.?265|hevc|av1|aac|flac|bdrip|webrip|bluray|dvdrip|web[-_. ]?dl|10bit|8bit)\b`)
	reYear           = regexp.MustCompile(`\b(19|20)\d{2}\b`)
	reLooseNumber    = regexp.MustCompile(`\b\d{1,4}\b`)
)

func OrganizeMediaFiles(raw string) (string, error) {
	in := MediaOrganizeInput{
		DryRun:            true,
		MinGroupSize:      2,
		MaxPreviewActions: 200,
	}
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return "", fmt.Errorf("invalid arguments JSON: %w", err)
	}

	in.Directory = strings.TrimSpace(in.Directory)
	if in.Directory == "" {
		return "", fmt.Errorf("directory is required")
	}
	if in.MinGroupSize <= 0 {
		in.MinGroupSize = 2
	}
	if in.MaxPreviewActions <= 0 {
		in.MaxPreviewActions = 200
	}
	if in.MaxPreviewActions > 1000 {
		in.MaxPreviewActions = 1000
	}

	root, err := filepath.Abs(in.Directory)
	if err != nil {
		return "", fmt.Errorf("failed to resolve directory: %w", err)
	}
	st, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("failed to stat directory: %w", err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", root)
	}

	allowedExt := buildAllowedExtMap(in.Extensions)
	files, err := collectMediaFiles(root, in.Recursive, allowedExt)
	if err != nil {
		return "", err
	}

	grouped := map[string][]mediaFile{}
	skipped := 0
	for _, f := range files {
		group := inferMediaGroupName(f.baseName)
		if group == "" {
			skipped++
			continue
		}
		grouped[group] = append(grouped[group], f)
	}

	res := mediaOrganizeResult{
		Directory:       root,
		DryRun:          in.DryRun,
		Recursive:       in.Recursive,
		TotalMediaFiles: len(files),
		DetectedGroups:  len(grouped),
		TopGroups:       make([]groupCount, 0, 80),
		PreviewActions:  make([]mediaMoveAction, 0),
		Warnings:        make([]string, 0),
	}

	groupNames := make([]string, 0, len(grouped))
	for g := range grouped {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)
	for i, g := range groupNames {
		if i < 80 {
			res.TopGroups = append(res.TopGroups, groupCount{Group: g, Count: len(grouped[g])})
			continue
		}
		res.OmittedGroups++
	}

	for _, g := range groupNames {
		items := grouped[g]
		if len(items) < in.MinGroupSize {
			skipped += len(items)
			continue
		}
		res.EligibleGroups++

		targetDir := filepath.Join(root, sanitizeFolderName(g))
		for _, item := range items {
			target := filepath.Join(targetDir, item.baseName)
			if samePath(item.absPath, target) {
				skipped++
				continue
			}
			res.PlannedMoves++
			if len(res.PreviewActions) < in.MaxPreviewActions {
				res.PreviewActions = append(res.PreviewActions, mediaMoveAction{
					From:  item.absPath,
					To:    target,
					Group: g,
				})
			}

			if in.DryRun {
				continue
			}
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("mkdir failed for %s: %v", targetDir, err))
				skipped++
				continue
			}

			finalTarget := resolveCollisionPath(target)
			if err := os.Rename(item.absPath, finalTarget); err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("move failed: %s -> %s (%v)", item.absPath, finalTarget, err))
				skipped++
				continue
			}
			res.Moved++
		}
	}

	res.Skipped += skipped

	blob, _ := json.Marshal(res)
	return string(blob), nil
}

func collectMediaFiles(root string, recursive bool, allowedExt map[string]struct{}) ([]mediaFile, error) {
	out := make([]mediaFile, 0, 256)
	if recursive {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if _, ok := allowedExt[ext]; !ok {
				return nil
			}
			out = append(out, mediaFile{absPath: path, baseName: d.Name()})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scan directory: %w", err)
		}
		return out, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if _, ok := allowedExt[ext]; !ok {
			continue
		}
		out = append(out, mediaFile{absPath: filepath.Join(root, e.Name()), baseName: e.Name()})
	}
	return out, nil
}

func buildAllowedExtMap(exts []string) map[string]struct{} {
	if len(exts) == 0 {
		exts = []string{
			".mkv", ".mp4", ".avi", ".mov", ".m4v", ".wmv", ".flv", ".webm", ".ts",
		}
	}
	m := make(map[string]struct{}, len(exts))
	for _, e := range exts {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		m[e] = struct{}{}
	}
	return m
}

func inferMediaGroupName(fileName string) string {
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, ".", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = reBracketContent.ReplaceAllString(base, " ")
	base = reSeasonEpisode.ReplaceAllString(base, " ")
	base = reEpisodeOnly.ReplaceAllString(base, " ")
	base = reResolution.ReplaceAllString(base, " ")
	base = reCodec.ReplaceAllString(base, " ")
	base = reYear.ReplaceAllString(base, " ")
	base = reLooseNumber.ReplaceAllString(base, " ")
	base = strings.ToLower(base)
	base = reMultiSpace.ReplaceAllString(strings.TrimSpace(base), " ")
	if base == "" {
		return ""
	}

	stopWords := map[string]struct{}{
		"episode": {}, "ep": {}, "season": {}, "s": {}, "the": {}, "a": {}, "an": {},
		"dual": {}, "audio": {}, "sub": {}, "subs": {}, "dub": {}, "uncensored": {},
	}

	words := strings.Fields(base)
	filtered := make([]string, 0, len(words))
	for _, w := range words {
		if _, skip := stopWords[w]; skip {
			continue
		}
		if len(w) == 1 {
			continue
		}
		filtered = append(filtered, w)
		if len(filtered) >= 6 {
			break
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	return toTitleCase(strings.Join(filtered, " "))
}

func toTitleCase(s string) string {
	words := strings.Fields(strings.TrimSpace(s))
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

func sanitizeFolderName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "Unsorted"
	}
	repl := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, r := range repl {
		s = strings.ReplaceAll(s, r, " ")
	}
	s = reMultiSpace.ReplaceAllString(strings.TrimSpace(s), " ")
	if s == "" {
		return "Unsorted"
	}
	return s
}

func resolveCollisionPath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; i <= 9999; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return path
}

func samePath(a, b string) bool {
	aa := filepath.Clean(a)
	bb := filepath.Clean(b)
	return aa == bb
}
