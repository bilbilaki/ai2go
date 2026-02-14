package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/chat"
	"github.com/bilbilaki/ai2go/internal/config"
)

func HandleSummarizeCommand(parts []string, history *chat.History, store *chat.ThreadStore, cfg *config.Config, apiClient *api.Client) {
	if len(parts) == 1 || strings.EqualFold(parts[1], "now") {
		if err := runSummary(context.Background(), history, store, cfg, apiClient, "manual"); err != nil {
			fmt.Printf("\033[31mError generating summary: %v\033[0m\n", err)
		}
		return
	}

	if strings.EqualFold(parts[1], "auto") {
		handleSummarizeAuto(parts[2:], cfg)
		return
	}

	fmt.Println("Usage: /summarize [now|auto on|auto off|auto status|auto threshold <tokens>]")
}

func handleSummarizeAuto(args []string, cfg *config.Config) {
	if len(args) == 0 || strings.EqualFold(args[0], "status") {
		status := "OFF"
		if cfg.AutoSummarize {
			status = "ON"
		}
		fmt.Printf("Auto summarize: %s (threshold=%d tokens)\n", status, cfg.AutoSummaryThreshold)
		return
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "on":
		if !cfg.AutoSummarize {
			cfg.ToggleAutoSummarize()
		}
		fmt.Printf("Auto summarize: ON (threshold=%d tokens)\n", cfg.AutoSummaryThreshold)
	case "off":
		if cfg.AutoSummarize {
			cfg.ToggleAutoSummarize()
		}
		fmt.Println("Auto summarize: OFF")
	case "threshold":
		if len(args) < 2 {
			fmt.Println("Usage: /summarize auto threshold <tokens>")
			return
		}
		n, err := strconv.Atoi(strings.TrimSpace(args[1]))
		if err != nil || n < 1000 {
			fmt.Println("Invalid threshold. Use an integer >= 1000.")
			return
		}
		cfg.SetAutoSummaryThreshold(n)
		fmt.Printf("Auto summarize threshold set to %d tokens.\n", cfg.AutoSummaryThreshold)
	default:
		fmt.Println("Usage: /summarize auto [on|off|status|threshold <tokens>]")
	}
}

func TryAutoSummarize(history *chat.History, store *chat.ThreadStore, cfg *config.Config, apiClient *api.Client) {
	if history == nil || store == nil || cfg == nil || apiClient == nil {
		return
	}
	if !cfg.AutoSummarize {
		return
	}
	if history.GetTotalTokens() < int64(cfg.AutoSummaryThreshold) {
		return
	}

	fmt.Printf("\n\033[33m[Auto Summary] Token threshold reached (%d >= %d). Compressing thread...\033[0m\n", history.GetTotalTokens(), cfg.AutoSummaryThreshold)
	if err := runSummary(context.Background(), history, store, cfg, apiClient, "auto"); err != nil {
		fmt.Printf("\033[31m[Auto Summary] Failed: %v\033[0m\n", err)
	}
}

func runSummary(ctx context.Context, history *chat.History, store *chat.ThreadStore, cfg *config.Config, apiClient *api.Client, mode string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	fmt.Println("\n\033[33mGenerating session summary (multi-pass)...\033[0m")
	result, err := chat.SummarizeHistoryMultiPass(ctx, history, apiClient, cfg.CurrentModel)
	if err != nil {
		return err
	}

	history.ReplaceWithSummary(cfg.CurrentModel, result.Summary)
	if err := store.SyncActiveHistory(history); err != nil {
		return fmt.Errorf("summary created but failed to persist thread: %w", err)
	}

	fmt.Printf("\033[32mHistory summarized (%s mode). passes=%d chunks=%d\033[0m\n", mode, result.Passes, result.Chunks)
	return nil
}
