package tools

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)
func SearchFilesWrapper(ctx context.Context, cfg *Config) (string, error) {
	// Redirect stdout to capture the results
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// Execute original search
	RunSearch(ctx, cfg)

	// Restore and close
	w.Close()
	os.Stdout = old
	return <-outC, nil
}

// ListTreeWrapper adapts the RunTreeGenerator logic for the AI Processor.
func ListTreeWrapper(cfg *Config) (string, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	RunTreeGenerator(cfg)

	w.Close()
	os.Stdout = old
	return <-outC, nil
}

// ======================================================================================
// CONFIGURATION & CONSTANTS
// ======================================================================================

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// Config holds all CLI arguments and compiled regexes
type Config struct {
	RootPath          string
	Extensions        []string
	IncludePathRegex  []*regexp.Regexp
	ExcludePathRegex  []*regexp.Regexp
	IncludeNameRegex  []*regexp.Regexp
	ExcludeNameRegex  []*regexp.Regexp
	ContentRegex      *regexp.Regexp
	ContentInclude    bool
	ShowTree          bool
	WorkerCount       int
	Verbose           bool
}

// Global buffer pool to reduce GC pressure during content reading
var bufferPool = sync.Pool{
	New: func() interface{} {
		// Allocate 32KB buffers
		return make([]byte, 32*1024)
	},
}

// ======================================================================================
// MAIN ENTRY POINT
// ======================================================================================

// func main() {
// 	// 1. Parse CLI Flags
// 	config := parseFlags()

// 	// 2. Setup Signal Handling (Ctrl+C)
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()
// 	setupSignalHandler(cancel)

// 	// 3. Mode Selection
// 	if config.ShowTree {
// 		runTreeGenerator(config)
// 	} else {
// 		runSearch(ctx, config)
// 	}
// }

// ======================================================================================
// FLAG PARSING & SETUP
// ======================================================================================

func ParseFlags() *Config {
	rootDir := flag.String("dir", ".", "Directory to search")
	exts := flag.String("ext", "", "Comma-separated extensions (e.g. .go,.dart,.json)")
	incPath := flag.String("inc-path", "", "Include path patterns (wildcards allowed: src/*)")
	excPath := flag.String("exc-path", "", "Exclude path patterns (wildcards allowed: .git/*)")
	incName := flag.String("inc-name", "", "Include filename patterns (wildcards allowed: *_test.go)")
	excName := flag.String("exc-name", "", "Exclude filename patterns")
	content := flag.String("content", "", "Regex content filter")
	contentExc := flag.Bool("content-exclude", false, "Exclude files matching content regex instead of including")
	tree := flag.Bool("tree", false, "Generate file tree instead of searching")
	workers := flag.Int("workers", runtime.NumCPU(), "Number of concurrent workers")
	verbose := flag.Bool("v", false, "Verbose output")

	flag.Parse()

	absRoot, err := filepath.Abs(*rootDir)
	if err != nil {
		fatalf("Failed to resolve absolute path: %v", err)
	}

	// Parse Extensions
	var extList []string
	if *exts != "" {
		parts := strings.Split(*exts, ",")
		for _, p := range parts {
			ext := strings.TrimSpace(p)
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			extList = append(extList, strings.ToLower(ext))
		}
	}

	cfg := &Config{
		RootPath:       absRoot,
		Extensions:     extList,
		ContentInclude: !*contentExc,
		ShowTree:       *tree,
		WorkerCount:    *workers,
		Verbose:        *verbose,
	}

	// Compile Regex Patterns
	cfg.IncludePathRegex = compilePatterns(*incPath)
	cfg.ExcludePathRegex = compilePatterns(*excPath)
	cfg.IncludeNameRegex = compilePatterns(*incName)
	cfg.ExcludeNameRegex = compilePatterns(*excName)

	if *content != "" {
		re, err := regexp.Compile(*content) // Use standard regex for content
		if err != nil {
			fatalf("Invalid content regex: %v", err)
		}
		cfg.ContentRegex = re
	}

	return cfg
}

func SetupSignalHandler(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Println("\n" + ColorRed + "Cancelling operation..." + ColorReset)
		cancel()
	}()
}

// ======================================================================================
// PATTERN MATCHING ENGINE
// ======================================================================================

// compilePatterns replicates the Dart logic:
// 1. Escape regex chars
// 2. Replace '*' with '.*'
// 3. Handle path prefixes/suffixes
func compilePatterns(input string) []*regexp.Regexp {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	var results []*regexp.Regexp
	parts := strings.Split(input, ",")
	for _, p := range parts {
		pattern := strings.TrimSpace(p)
		if pattern == "" {
			continue
		}

		// Escape special regex characters first
		regexStr := regexp.QuoteMeta(pattern)

		// Dart logic: If starts with /, anchor start. If ends with /, anchor end.
		// Note: We use filepath separator handling in Go, so we normalize.
		if strings.HasPrefix(pattern, "/") || strings.HasPrefix(pattern, "\\") {
			regexStr = "^" + regexStr
		}
		if strings.HasSuffix(pattern, "/") || strings.HasSuffix(pattern, "\\") {
			regexStr = regexStr + "$"
		}

		// Un-escape the '*' we just escaped, and turn it into '.*'
		// QuoteMeta turns '*' into '\*'
		regexStr = strings.ReplaceAll(regexStr, "\\*", ".*")

		re, err := regexp.Compile("(?i)" + regexStr) // Case insensitive
		if err != nil {
			fmt.Printf(ColorYellow+"Warning: Invalid pattern '%s': %v\n"+ColorReset, pattern, err)
			continue
		}
		results = append(results, re)
	}
	return results
}

func matchesAny(s string, regexes []*regexp.Regexp) bool {
	for _, re := range regexes {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// ======================================================================================
// SEARCH LOGIC (CONCURRENT PIPELINE)
// ======================================================================================

func RunSearch(ctx context.Context, cfg *Config) {
	start := time.Now()
	fmt.Printf(ColorBlue+"Starting search in: %s\n"+ColorReset, cfg.RootPath)
	if len(cfg.Extensions) > 0 {
		fmt.Printf("Extensions: %v\n", cfg.Extensions)
	}

	pathsCh := make(chan string, 1000)
	resultsCh := make(chan string, 1000)
	var wg sync.WaitGroup

	// 1. Worker Pool
	for i := 0; i < cfg.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, cfg, pathsCh, resultsCh)
		}()
	}

	// 2. Result Collector (separate goroutine)
	totalFound := 0
	doneCh := make(chan struct{})
	go func() {
		for res := range resultsCh {
			fmt.Println(res)
			totalFound++
		}
		close(doneCh)
	}()

	// 3. File Walker (Producer)
	go func() {
		defer close(pathsCh)
		err := filepath.WalkDir(cfg.RootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if cfg.Verbose {
					fmt.Fprintf(os.Stderr, "Access error: %v\n", err)
				}
				return nil // Continue walking
			}
			if ctx.Err() != nil {
				return filepath.SkipAll
			}

			normalizedPath := filepath.ToSlash(path)

			// --- Directory Filtering ---
			if d.IsDir() {
				// If we have path exclude patterns, we might want to skip entire dirs
				// This matches the Flutter logic "if entity is Directory ... continue"
				if len(cfg.ExcludePathRegex) > 0 && matchesAny(normalizedPath, cfg.ExcludePathRegex) {
					return filepath.SkipDir
				}
				return nil
			}

			// --- Path Filtering (Pre-check) ---
			// Check excludes first
			if len(cfg.ExcludePathRegex) > 0 && matchesAny(normalizedPath, cfg.ExcludePathRegex) {
				return nil
			}
			// Check includes (if defined)
			if len(cfg.IncludePathRegex) > 0 && !matchesAny(normalizedPath, cfg.IncludePathRegex) {
				return nil
			}

			pathsCh <- path
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, ColorRed+"Walk error: %v\n"+ColorReset, err)
		}
	}()

	wg.Wait()      // Wait for workers
	close(resultsCh) // Signal collector
	<-doneCh       // Wait for collector

	fmt.Printf(ColorGreen+"\nSearch completed in %v. Found %d files.\n"+ColorReset, time.Since(start), totalFound)
}

func worker(ctx context.Context, cfg *Config, jobs <-chan string, results chan<- string) {
	for path := range jobs {
		if ctx.Err() != nil {
			return
		}

		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		fileName := info.Name()
		ext := strings.ToLower(filepath.Ext(fileName))

		// 1. Extension Filter
		if len(cfg.Extensions) > 0 {
			found := false
			for _, allowed := range cfg.Extensions {
				if allowed == ext {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// 2. Filename Pattern Filter
		if len(cfg.ExcludeNameRegex) > 0 && matchesAny(fileName, cfg.ExcludeNameRegex) {
			continue
		}
		if len(cfg.IncludeNameRegex) > 0 && !matchesAny(fileName, cfg.IncludeNameRegex) {
			continue
		}

		// 3. Content Filter (Heavy I/O)
		if cfg.ContentRegex != nil {
			matched, err := CheckFileContent(path, cfg.ContentRegex)
			if err != nil {
				if cfg.Verbose {
					fmt.Fprintf(os.Stderr, "Read error %s: %v\n", path, err)
				}
				continue
			}

			// Logic:
			// If IncludeMode: matched=true -> Keep
			// If ExcludeMode: matched=true -> Drop
			shouldKeep := false
			if cfg.ContentInclude {
				shouldKeep = matched
			} else {
				shouldKeep = !matched
			}

			if !shouldKeep {
				continue
			}
		}

		// Format output (relative path for cleaner display)
		relPath, _ := filepath.Rel(cfg.RootPath, path)
		results <- fmt.Sprintf("%s%s%s (%s)", ColorCyan, relPath, ColorReset, ReadableSize(info.Size()))
	}
}

// checkFileContent reads file using buffer pool to minimize allocations.
// It stops reading as soon as a match is found (if in include mode).
func CheckFileContent(path string, re *regexp.Regexp) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Use buffered reader
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	// Since regexp works on streams/bytes, we can scan the file.
	// However, standard regexp doesn't support io.Reader natively in a streaming fashion nicely for "Any Match".
	// For large files, reading chunk by chunk might miss matches crossing buffer boundaries.
	// For "0% bugs" correctness, we should read the whole file or use a sliding window.
	// Given typical source code files, reading fully is acceptable, but let's be memory safe and cap it.
	// Limitation: Files > 10MB are skipped for content search to prevent OOM in massive repos.

	stat, _ := f.Stat()
	if stat.Size() > 10*1024*1024 { // 10MB limit
		return false, nil
	}

	// Read full content (safe for source code)
	// We use the bufferPool buffer if it fits, otherwise ReadAll
	var data []byte
	if stat.Size() <= int64(len(buf)) {
		n, err := f.Read(buf)
		if err != nil && err.Error() != "EOF" {
			return false, err
		}
		data = buf[:n]
	} else {
		// Fallback for larger files
		data, err = os.ReadFile(path)
		if err != nil {
			return false, err
		}
	}

	return re.Match(data), nil
}

// ======================================================================================
// TREE GENERATOR
// ======================================================================================

func RunTreeGenerator(cfg *Config) {
	fmt.Printf(ColorBlue+"Generating File Tree for: %s\n"+ColorReset, cfg.RootPath)
	fmt.Println(".")
	err := generateTree(cfg.RootPath, "", cfg)
	if err != nil {
		fatalf("Tree generation failed: %v", err)
	}
}

func generateTree(dir string, prefix string, cfg *Config) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Filter entries before processing to know which is "last"
	var filtered []fs.DirEntry
	for _, e := range entries {
		// Basic visibility filter (skip dotfiles unless explicit? Keeping simple for now)
		filtered = append(filtered, e)
	}

	for i, e := range filtered {
		isLast := i == len(filtered)-1
		
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		name := e.Name()
		displayName := name
		if e.IsDir() {
			displayName = ColorBlue + name + ColorReset
		} else {
			displayName = ColorGreen + name + ColorReset
		}

		fmt.Printf("%s%s%s\n", prefix, connector, displayName)

		if e.IsDir() {
			extension := "│   "
			if isLast {
				extension = "    "
			}
			// Recursive call
			_ = generateTree(filepath.Join(dir, name), prefix+extension, cfg)
		}
	}
	return nil
}

// ======================================================================================
// UTILS
// ======================================================================================

func ReadableSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func fatalf(format string, args ...interface{}) {
	fmt.Printf(ColorRed+"FATAL: "+format+"\n"+ColorReset, args...)
	os.Exit(1)
}