package subagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/tools"
)

const (
	statusNoError         = "NOERROR"
	statusFailed          = "FAILED"
	statusUnknown         = "UNKNOWN"
	maxSubagentIterations = 16
	maxSubagentDepth      = 2
	defaultTimeoutSec     = 600
	defaultConcurrency    = 3
	finalizeWindow        = 20 * time.Second
)

type FactoryInput struct {
	TaskListName    string
	MegaPrompt      string
	SplitSymbol     string
	SplitRegex      string
	BaseInstruction string
	MaxConcurrency  int
	TimeoutSec      int
	TTLSeconds      int
	OutputDir       string
	Model           string
}

type TaskBrief struct {
	TaskID        string `json:"task_id"`
	PromptPreview string `json:"prompt_preview"`
	OutputHash    string `json:"output_hash,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
	OutputFile    string `json:"output_file,omitempty"`
}

type BatchReport struct {
	BatchID        string      `json:"batch_id"`
	TaskListName   string      `json:"task_list_name"`
	StartedAt      time.Time   `json:"started_at"`
	FinishedAt     time.Time   `json:"finished_at"`
	TotalStarted   int         `json:"total_started"`
	NoError        []TaskBrief `json:"noerror"`
	Failed         []TaskBrief `json:"failed"`
	Unknown        []TaskBrief `json:"unknown"`
	ReportFilePath string      `json:"report_file_path"`
	OutputDir      string      `json:"output_dir"`
}

type TaskContext struct {
	TaskID            string
	BatchID           string
	TaskListName      string
	Prompt            string
	PromptPreview     string
	Instruction       string
	Status            string
	ErrorCode         string
	ErrorMessage      string
	Output            string
	OutputFile        string
	OutputHash        string
	StartedAt         time.Time
	FinishedAt        time.Time
	DurationMs        int64
	TokenApprox       int
	LastOutputSnippet string
	ExpiresAt         time.Time
}

type Manager struct {
	mu       sync.RWMutex
	tasks    map[string]TaskContext
	reports  map[string]BatchReport
	ticker   *time.Ticker
	stopChan chan struct{}
}

var (
	defaultManager *Manager
	managerOnce    sync.Once
)

func DefaultManager() *Manager {
	managerOnce.Do(func() {
		defaultManager = NewManager()
	})
	return defaultManager
}

func NewManager() *Manager {
	m := &Manager{
		tasks:    make(map[string]TaskContext),
		reports:  make(map[string]BatchReport),
		ticker:   time.NewTicker(30 * time.Second),
		stopChan: make(chan struct{}),
	}
	go m.scrubExpiredLoop()
	return m
}

func (m *Manager) Close() {
	close(m.stopChan)
	m.ticker.Stop()
}

func (m *Manager) RunFactory(ctx context.Context, client *api.Client, defaultModel, systemPrompt string, input FactoryInput, experimentalEnabled bool) (BatchReport, error) {
	return m.runFactoryWithDepth(ctx, client, defaultModel, systemPrompt, input, experimentalEnabled, 1)
}

func (m *Manager) runFactoryWithDepth(ctx context.Context, client *api.Client, defaultModel, systemPrompt string, input FactoryInput, experimentalEnabled bool, depth int) (BatchReport, error) {
	if depth < 1 {
		depth = 1
	}
	if client == nil {
		return BatchReport{}, fmt.Errorf("api client is required")
	}
	if strings.TrimSpace(input.MegaPrompt) == "" {
		return BatchReport{}, fmt.Errorf("mega_prompt is required")
	}

	tasks, err := splitTasks(input)
	if err != nil {
		return BatchReport{}, err
	}
	if len(tasks) == 0 {
		return BatchReport{}, fmt.Errorf("no tasks were found after splitting")
	}

	taskListName := sanitizeName(input.TaskListName)
	if taskListName == "" {
		taskListName = "tasklist"
	}

	maxConc := input.MaxConcurrency
	if maxConc <= 0 {
		maxConc = defaultConcurrency
	}
	if maxConc > 200 {
		maxConc = 200
	}

	timeoutSec := input.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = defaultTimeoutSec
	}
	if timeoutSec > 3600 {
		timeoutSec = 3600
	}

	ttl := input.TTLSeconds
	if ttl <= 0 {
		ttl = 600
	}
	if ttl > 86400 {
		ttl = 86400
	}

	model := strings.TrimSpace(input.Model)
	if model == "" {
		model = strings.TrimSpace(defaultModel)
	}
	if model == "" {
		return BatchReport{}, fmt.Errorf("model is empty")
	}

	batchID := fmt.Sprintf("batch_%d", time.Now().UnixNano())
	baseOutputDir := strings.TrimSpace(input.OutputDir)
	if baseOutputDir == "" {
		baseOutputDir = filepath.Join(".ai2go", "subagents", batchID)
	}
	if err := os.MkdirAll(baseOutputDir, 0755); err != nil {
		return BatchReport{}, fmt.Errorf("failed to create output dir: %w", err)
	}

	bp := Blueprint{
		Client:              client,
		SystemPrompt:        systemPrompt,
		Model:               model,
		Manager:             m,
		ExperimentalEnabled: experimentalEnabled,
		Depth:               depth,
	}

	type taskResult struct {
		ctx TaskContext
	}

	startedAt := time.Now().UTC()
	suffix := shortBatchID(batchID)
	sem := make(chan struct{}, maxConc)
	results := make(chan taskResult, len(tasks))
	wg := sync.WaitGroup{}

	for idx, taskPrompt := range tasks {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(i int, prompt string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			taskID := fmt.Sprintf("%03d_%s_%s", i+1, taskListName, suffix)
			preview := firstTwoLines(prompt)
			started := time.Now().UTC()
			instruction := strings.TrimSpace(input.BaseInstruction)

			tctx := TaskContext{
				TaskID:        taskID,
				BatchID:       batchID,
				TaskListName:  taskListName,
				Prompt:        prompt,
				PromptPreview: preview,
				Instruction:   instruction,
				Status:        statusUnknown,
				StartedAt:     started,
				ExpiresAt:     started.Add(time.Duration(ttl) * time.Second),
			}

			runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
			defer cancel()

			agent := bp.Clone(taskID)
			out, runErr := agent.Run(runCtx, prompt, instruction)
			finished := time.Now().UTC()
			tctx.FinishedAt = finished
			tctx.DurationMs = finished.Sub(started).Milliseconds()
			tctx.Output = out
			tctx.TokenApprox = approxTokens(out)
			tctx.LastOutputSnippet = snippet(out, 220)

			if runErr != nil {
				tctx.Status = statusFailed
				tctx.ErrorMessage = runErr.Error()
				tctx.ErrorCode = classifyRunErr(runErr)
			} else if strings.TrimSpace(out) == "" {
				tctx.Status = statusUnknown
				tctx.ErrorCode = "empty_output"
			} else {
				tctx.Status = statusNoError
			}

			outputFile := filepath.Join(baseOutputDir, taskID+".txt")
			fileBody := buildTaskOutputFile(tctx)
			if writeErr := os.WriteFile(outputFile, []byte(fileBody), 0644); writeErr == nil {
				tctx.OutputFile = outputFile
				tctx.OutputHash = hashString(fileBody)
			}

			m.storeTask(tctx)
			results <- taskResult{ctx: tctx}
		}(idx, taskPrompt)
	}

	wg.Wait()
	close(results)

	all := make([]TaskContext, 0, len(tasks))
	for item := range results {
		all = append(all, item.ctx)
	}

	sort.SliceStable(all, func(i, j int) bool {
		return all[i].TaskID < all[j].TaskID
	})

	report := BatchReport{
		BatchID:      batchID,
		TaskListName: taskListName,
		StartedAt:    startedAt,
		FinishedAt:   time.Now().UTC(),
		TotalStarted: len(all),
		NoError:      make([]TaskBrief, 0),
		Failed:       make([]TaskBrief, 0),
		Unknown:      make([]TaskBrief, 0),
		OutputDir:    baseOutputDir,
	}

	for _, item := range all {
		brief := TaskBrief{
			TaskID:        item.TaskID,
			PromptPreview: item.PromptPreview,
			OutputHash:    item.OutputHash,
			ErrorCode:     item.ErrorCode,
			OutputFile:    item.OutputFile,
		}

		switch item.Status {
		case statusNoError:
			report.NoError = append(report.NoError, brief)
		case statusFailed:
			report.Failed = append(report.Failed, brief)
		default:
			report.Unknown = append(report.Unknown, brief)
		}
	}

	reportFile := filepath.Join(baseOutputDir, "report.json")
	blob, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(reportFile, blob, 0644); err == nil {
		report.ReportFilePath = reportFile
	}

	m.mu.Lock()
	m.reports[report.BatchID] = report
	m.mu.Unlock()

	return report, nil
}

func (m *Manager) GetTaskContextSummary(taskID string, consume bool) (string, error) {
	id := strings.TrimSpace(taskID)
	if id == "" {
		return "", fmt.Errorf("task_id is required")
	}

	m.mu.RLock()
	task, ok := m.tasks[id]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("task context not found: %s", id)
	}

	summary := fmt.Sprintf(
		"TaskID: %s\nBatchID: %s\nStatus: %s\nErrorCode: %s\nDurationMs: %d\nPromptPreview: %s\nOutputFile: %s\nOutputHash: %s\nLastOutputSnippet: %s\nErrorMessage: %s",
		task.TaskID,
		task.BatchID,
		task.Status,
		task.ErrorCode,
		task.DurationMs,
		task.PromptPreview,
		task.OutputFile,
		task.OutputHash,
		task.LastOutputSnippet,
		task.ErrorMessage,
	)

	if consume {
		m.mu.Lock()
		delete(m.tasks, id)
		m.mu.Unlock()
	}

	return summary, nil
}

func (m *Manager) storeTask(task TaskContext) {
	m.mu.Lock()
	m.tasks[task.TaskID] = task
	m.mu.Unlock()
}

func (m *Manager) scrubExpiredLoop() {
	for {
		select {
		case <-m.ticker.C:
			now := time.Now().UTC()
			m.mu.Lock()
			for id, task := range m.tasks {
				if !task.ExpiresAt.IsZero() && now.After(task.ExpiresAt) {
					delete(m.tasks, id)
				}
			}
			m.mu.Unlock()
		case <-m.stopChan:
			return
		}
	}
}

func splitTasks(input FactoryInput) ([]string, error) {
	raw := input.MegaPrompt
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	parts := []string{}
	splitRegex := strings.TrimSpace(input.SplitRegex)
	splitSymbol := strings.TrimSpace(input.SplitSymbol)

	if splitRegex != "" {
		re, err := regexp.Compile(splitRegex)
		if err != nil {
			return nil, fmt.Errorf("invalid split_regex: %w", err)
		}
		parts = re.Split(raw, -1)
	} else {
		if splitSymbol == "" {
			splitSymbol = "---TASK---"
		}
		parts = strings.Split(raw, splitSymbol)
	}

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}

	if len(out) == 0 && strings.TrimSpace(raw) != "" {
		out = append(out, strings.TrimSpace(raw))
	}

	return out, nil
}

func ParseFactoryInput(raw string) (FactoryInput, error) {
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return FactoryInput{}, fmt.Errorf("invalid arguments JSON: %w", err)
	}

	in := FactoryInput{
		TaskListName:    getString(obj, "task_list_name", ""),
		MegaPrompt:      getString(obj, "mega_prompt", ""),
		SplitSymbol:     getString(obj, "split_symbol", "---TASK---"),
		SplitRegex:      getString(obj, "split_regex", ""),
		BaseInstruction: getString(obj, "base_instruction", ""),
		MaxConcurrency:  getInt(obj, "max_concurrency", defaultConcurrency),
		TimeoutSec:      getInt(obj, "timeout_sec", defaultTimeoutSec),
		TTLSeconds:      getInt(obj, "ttl_seconds", 600),
		OutputDir:       getString(obj, "output_dir", ""),
		Model:           getString(obj, "model", ""),
	}
	return in, nil
}

func ParseContextProviderInput(raw string) (taskID string, consume bool, err error) {
	var obj map[string]any
	if parseErr := json.Unmarshal([]byte(raw), &obj); parseErr != nil {
		return "", false, fmt.Errorf("invalid arguments JSON: %w", parseErr)
	}

	taskID = strings.TrimSpace(getString(obj, "task_id", ""))
	if taskID == "" {
		return "", false, fmt.Errorf("task_id is required")
	}
	consume = getBool(obj, "consume", true)
	return taskID, consume, nil
}

func FormatBatchReport(report BatchReport) string {
	return fmt.Sprintf(
		"Subagent batch finished.\nBatchID: %s\nTaskList: %s\nStarted: %d\nNOERROR: %d\nFAILED: %d\nUNKNOWN: %d\nOutputDir: %s\nReportFile: %s\nUse tool 'subagent_context_provider' with task_id to inspect failed/unknown tasks.",
		report.BatchID,
		report.TaskListName,
		report.TotalStarted,
		len(report.NoError),
		len(report.Failed),
		len(report.Unknown),
		report.OutputDir,
		report.ReportFilePath,
	)
}

func buildTaskOutputFile(task TaskContext) string {
	return fmt.Sprintf(
		"TaskID: %s\nBatchID: %s\nTaskList: %s\nStatus: %s\nErrorCode: %s\nErrorMessage: %s\nStartedAt: %s\nFinishedAt: %s\nDurationMs: %d\nPromptPreview:\n%s\n\nInstruction:\n%s\n\nTaskPrompt:\n%s\n\nOutput:\n%s\n",
		task.TaskID,
		task.BatchID,
		task.TaskListName,
		task.Status,
		task.ErrorCode,
		task.ErrorMessage,
		task.StartedAt.Format(time.RFC3339),
		task.FinishedAt.Format(time.RFC3339),
		task.DurationMs,
		task.PromptPreview,
		task.Instruction,
		task.Prompt,
		task.Output,
	)
}

func classifyRunErr(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "run_error"
}

func firstTwoLines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return ""
	}
	if len(lines) == 1 {
		return strings.TrimSpace(lines[0])
	}
	return strings.TrimSpace(lines[0]) + " | " + strings.TrimSpace(lines[1])
}

func snippet(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func approxTokens(s string) int {
	if s == "" {
		return 0
	}
	r := len([]rune(s))
	if r < 4 {
		return 1
	}
	return r / 4
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, " ", "_")
	re := regexp.MustCompile(`[^a-z0-9_-]+`)
	s = re.ReplaceAllString(s, "")
	if s == "" {
		return "tasklist"
	}
	return s
}

func shortBatchID(batchID string) string {
	if strings.TrimSpace(batchID) == "" {
		return "batch"
	}
	parts := strings.Split(batchID, "_")
	last := parts[len(parts)-1]
	if len(last) > 6 {
		return last[len(last)-6:]
	}
	return last
}

func getString(m map[string]any, key, def string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch val := v.(type) {
	case string:
		if strings.TrimSpace(val) == "" {
			return def
		}
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return def
	}
}

func getInt(m map[string]any, key string, def int) int {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err == nil {
			return n
		}
	}
	return def
}

func getBool(m map[string]any, key string, def bool) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(val))
		return trimmed == "1" || trimmed == "true" || trimmed == "yes" || trimmed == "y"
	case float64:
		return val != 0
	default:
		return def
	}
}

type Blueprint struct {
	Client              *api.Client
	SystemPrompt        string
	Model               string
	Manager             *Manager
	ExperimentalEnabled bool
	Depth               int
}

type Agent struct {
	id                  string
	client              *api.Client
	systemPrompt        string
	model               string
	manager             *Manager
	experimentalEnabled bool
	depth               int
}

func (b Blueprint) Clone(id string) Agent {
	return Agent{
		id:                  id,
		client:              b.Client,
		systemPrompt:        b.SystemPrompt,
		model:               b.Model,
		manager:             b.Manager,
		experimentalEnabled: b.ExperimentalEnabled,
		depth:               b.Depth,
	}
}

func (a Agent) Run(ctx context.Context, taskPrompt, instruction string) (string, error) {
	workerSystem := strings.TrimSpace(a.systemPrompt)
	if workerSystem == "" {
		workerSystem = "You are a reliable coding subagent."
	}

	workerSystem += "\n\nYou are a subagent worker. Complete only the assigned task. Keep the scope narrow. Avoid broad refactors and avoid unrelated files. Do not run long/global commands (full build/test) unless the task explicitly requires it. Use concise tool calls, then return a final answer quickly with changed files and what remains."

	userPrompt := strings.TrimSpace(taskPrompt)
	if strings.TrimSpace(instruction) != "" {
		userPrompt = fmt.Sprintf("Instruction:\n%s\n\nTask:\n%s", strings.TrimSpace(instruction), strings.TrimSpace(taskPrompt))
	}

	msgs := []api.Message{
		{Role: "system", Content: workerSystem},
		{Role: "user", Content: userPrompt},
	}
	progress := strings.Builder{}

	toolList := []api.Tool{
		tools.GetCLITool(),
		tools.GetReadFileTool(),
		tools.GetPatchFileTool(),
		tools.GetApplyUnifiedDiffPatchTool(),
		tools.GetCreateCheckpointTool(),
		tools.GetUndoCheckpointsTool(),
		tools.GetEditorHistoryTool(),
		tools.GetCPUUsageSampleTool(),
		tools.GetProcessSignalTool(),
		tools.GetPageSizeTool(),
		tools.GetSubagentContextProviderTool(),
	}
	if a.experimentalEnabled {
		toolList = append(toolList, tools.GetSubagentFactoryTool())
	}

	for i := 0; i < maxSubagentIterations; i++ {
		if shouldFinalizeNow(ctx, finalizeWindow) {
			finalText, err := a.forceFinalizeNoTools(ctx, msgs)
			if err == nil && strings.TrimSpace(finalText) != "" {
				if progress.Len() > 0 {
					return strings.TrimSpace(progress.String()) + "\n\n" + finalText, nil
				}
				return finalText, nil
			}
			if progress.Len() > 0 {
				return strings.TrimSpace(progress.String()), context.DeadlineExceeded
			}
			return "", context.DeadlineExceeded
		}

		resp, err := a.client.RunCompletionOnce(ctx, msgs, toolList, a.model)
		if err != nil {
			if progress.Len() > 0 {
				return strings.TrimSpace(progress.String()), err
			}
			return "", err
		}
		msgs = append(msgs, resp)
		if txt := strings.TrimSpace(resp.Content); txt != "" {
			progress.WriteString("Assistant:\n")
			progress.WriteString(txt)
			progress.WriteString("\n\n")
		}
		if len(resp.ToolCalls) == 0 {
			if progress.Len() > 0 {
				return strings.TrimSpace(progress.String()), nil
			}
			return strings.TrimSpace(resp.Content), nil
		}

		for _, tc := range resp.ToolCalls {
			progress.WriteString(fmt.Sprintf("ToolCall: %s\n", tc.Function.Name))
			toolOutput := a.executeToolCall(ctx, tc)
			if snip := snippet(toolOutput, 600); strings.TrimSpace(snip) != "" {
				progress.WriteString("ToolOutput:\n")
				progress.WriteString(snip)
				progress.WriteString("\n\n")
			}
			msgs = append(msgs, api.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    toolOutput,
			})
		}
	}

	if progress.Len() > 0 {
		return strings.TrimSpace(progress.String()), fmt.Errorf("subagent exceeded maximum tool iterations (%d)", maxSubagentIterations)
	}
	return "", fmt.Errorf("subagent exceeded maximum tool iterations (%d)", maxSubagentIterations)
}

func shouldFinalizeNow(ctx context.Context, threshold time.Duration) bool {
	if threshold <= 0 {
		return false
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return false
	}
	return time.Until(deadline) <= threshold
}

func (a Agent) forceFinalizeNoTools(ctx context.Context, msgs []api.Message) (string, error) {
	finalizePrompt := "Time budget is almost finished. Stop using tools now. Return a concise final report with: 1) what was completed, 2) exact files changed, 3) unresolved items."
	msgs = append(msgs, api.Message{Role: "user", Content: finalizePrompt})

	finalCtx := ctx
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 2*time.Second {
			budget := 8 * time.Second
			if remaining < budget {
				budget = remaining - time.Second
			}
			if budget > 0 {
				var cancel context.CancelFunc
				finalCtx, cancel = context.WithTimeout(ctx, budget)
				defer cancel()
			}
		}
	}

	resp, err := a.client.RunCompletionOnce(finalCtx, msgs, nil, a.model)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

func (a Agent) executeToolCall(ctx context.Context, tc api.ToolCall) string {
	switch tc.Function.Name {
	case "run_command":
		var args map[string]string
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Error: invalid arguments for run_command: %v", err)
		}
		cmd := strings.TrimSpace(args["command"])
		if cmd == "" {
			return "Error: run_command requires a non-empty 'command' argument."
		}
		out, err := tools.ExecuteShellCommand(ctx, cmd)
		if err != nil && strings.TrimSpace(out) == "" {
			return fmt.Sprintf("Error: %v", err)
		}
		if err != nil {
			return fmt.Sprintf("%s\n\nError: %v", out, err)
		}
		return out
	case "read_file":
		var args map[string]string
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Error: invalid arguments for read_file: %v", err)
		}
		path := strings.TrimSpace(args["path"])
		if path == "" {
			return "Error: read_file requires a non-empty 'path' argument."
		}
		out, err := tools.ReadFileWithLines(path, strings.TrimSpace(args["line_range"]))
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return out
	case "patch_file":
		var args map[string]string
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Error: invalid arguments for patch_file: %v", err)
		}
		path := strings.TrimSpace(args["path"])
		patch := args["patch"]
		if path == "" {
			return "Error: patch_file requires a non-empty 'path' argument."
		}
		if strings.TrimSpace(patch) == "" {
			return "Error: patch_file requires a non-empty 'patch' argument."
		}
		out, err := tools.ApplyFilePatch(path, patch)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return out
	case "apply_unified_diff_patch":
		var args map[string]string
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Error: invalid arguments for apply_unified_diff_patch: %v", err)
		}
		workTree := strings.TrimSpace(args["work_tree"])
		patch := args["patch"]
		verifyMode := tools.VerifyMode(strings.TrimSpace(args["verify_mode"]))
		if verifyMode == "" {
			verifyMode = tools.VerifyModeNone
		}
		if workTree == "" {
			return "Error: apply_unified_diff_patch requires a non-empty 'work_tree' argument."
		}
		if strings.TrimSpace(patch) == "" {
			return "Error: apply_unified_diff_patch requires a non-empty 'patch' argument."
		}
		out, err := tools.ApplyUnifiedDiffPatch(workTree, patch, verifyMode)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return out
	case "create_checkpoint":
		var args map[string]string
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Error: invalid arguments for create_checkpoint: %v", err)
		}
		workTree := strings.TrimSpace(args["work_tree"])
		if workTree == "" {
			return "Error: create_checkpoint requires a non-empty 'work_tree' argument."
		}
		head, err := tools.CreateCheckpoint(workTree, strings.TrimSpace(args["file_path"]), strings.TrimSpace(args["message"]))
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return fmt.Sprintf("Checkpoint created: %s", head)
	case "undo_checkpoints":
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Error: invalid arguments for undo_checkpoints: %v", err)
		}
		workTree, _ := args["work_tree"].(string)
		workTree = strings.TrimSpace(workTree)
		if workTree == "" {
			return "Error: undo_checkpoints requires a non-empty 'work_tree' argument."
		}
		steps := 1
		if raw, ok := args["steps"]; ok {
			switch v := raw.(type) {
			case float64:
				steps = int(v)
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
					steps = n
				}
			}
		}
		head, err := tools.UndoLastCheckpoints(workTree, steps)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return fmt.Sprintf("Undo complete. HEAD=%s", head)
	case "editor_history":
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Error: invalid arguments for editor_history: %v", err)
		}
		workTree, _ := args["work_tree"].(string)
		workTree = strings.TrimSpace(workTree)
		if workTree == "" {
			return "Error: editor_history requires a non-empty 'work_tree' argument."
		}
		limit := 10
		if raw, ok := args["limit"]; ok {
			switch v := raw.(type) {
			case float64:
				limit = int(v)
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
					limit = n
				}
			}
		}
		history, err := tools.EditorHistory(workTree, limit)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return history
	case "get_process_cpu_usage_sample":
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Error: invalid arguments for get_process_cpu_usage_sample: %v", err)
		}
		rawPids, _ := args["pids"].([]any)
		if len(rawPids) == 0 {
			return "Error: get_process_cpu_usage_sample requires non-empty 'pids'."
		}
		pids := make([]int, 0, len(rawPids))
		for _, p := range rawPids {
			if f, ok := p.(float64); ok {
				pids = append(pids, int(f))
			}
		}
		asInteger, _ := args["as_integer"].(bool)
		if asInteger {
			vals, err := tools.GetProcessCPUUsageSimple(pids)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			blob, _ := json.Marshal(vals)
			return string(blob)
		}
		vals, err := tools.GetProcessCPUUsage(pids)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		blob, _ := json.Marshal(vals)
		return string(blob)
	case "send_process_signal":
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Error: invalid arguments for send_process_signal: %v", err)
		}
		pidF, ok := args["pid"].(float64)
		if !ok {
			return "Error: send_process_signal requires integer 'pid'."
		}
		pid := int(pidF)
		signalName, _ := args["signal"].(string)
		signalName = strings.TrimSpace(signalName)
		if signalName == "" {
			signalName = "TERM"
		}
		grace := 0
		if g, ok := args["graceful_timeout"].(float64); ok {
			grace = int(g)
		}
		force, _ := args["force"].(bool)
		if err := tools.KillProcessTreeWithTimeout(pid, signalName, grace, force); err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return fmt.Sprintf("Signal handling completed for pid=%d", pid)
	case "get_page_size":
		return fmt.Sprintf("%d", os.Getpagesize())
	case "subagent_context_provider":
		taskID, consume, err := ParseContextProviderInput(tc.Function.Arguments)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		if a.manager == nil {
			return "Error: subagent manager is unavailable."
		}
		out, err := a.manager.GetTaskContextSummary(taskID, consume)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return out
	case "subagent_factory":
		if !a.experimentalEnabled {
			return "Error: subagent_factory is disabled for subagents. Enable experimental mode first."
		}
		if a.depth >= maxSubagentDepth {
			return fmt.Sprintf("Error: nested subagent depth limit reached (%d).", maxSubagentDepth)
		}
		if a.manager == nil {
			return "Error: subagent manager is unavailable."
		}
		input, err := ParseFactoryInput(tc.Function.Arguments)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		report, err := a.manager.runFactoryWithDepth(ctx, a.client, a.model, a.systemPrompt, input, a.experimentalEnabled, a.depth+1)
		if err != nil {
			return fmt.Sprintf("Error: subagent_factory failed: %v", err)
		}
		return FormatBatchReport(report)
	default:
		return fmt.Sprintf("Error: unsupported tool '%s'", tc.Function.Name)
	}
}
