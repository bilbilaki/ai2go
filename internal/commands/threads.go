package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bilbilaki/ai2go/internal/chat"
	"github.com/bilbilaki/ai2go/internal/config"
	"github.com/bilbilaki/ai2go/internal/ui"
)

func handleThreadsList(parts []string, store *chat.ThreadStore) {
	query, sortBy, order := parseQuerySortOrder(parts[1:])
	threads := store.ListThreads(query, sortBy, order)
	if len(threads) == 0 {
		fmt.Println("No threads matched.")
		return
	}

	activeID := store.ActiveThreadID()
	fmt.Printf("Threads (%d):\n", len(threads))
	for i, t := range threads {
		marker := " "
		if t.ID == activeID {
			marker = "*"
		}
		fmt.Printf("%s %2d. %s\n", marker, i+1, ui.Thread(t.Title))
		fmt.Printf("    id=%s messages=%d updated=%s\n", t.ID, len(t.Messages), t.UpdatedAt.Local().Format(time.RFC822))
	}
}

func handleThreadCommand(parts []string, history *chat.History, store *chat.ThreadStore, cfg *config.Config) {
	if len(parts) < 2 {
		fmt.Println("Usage: /thread [new|open|rename|current] ...")
		return
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "new":
		title := ""
		if len(parts) > 2 {
			title = strings.Join(parts[2:], " ")
		}
		thread, err := store.NewThread(cfg.CurrentModel, title, history)
		if err != nil {
			fmt.Printf("\033[31mError creating thread: %v\033[0m\n", err)
			return
		}
		fmt.Printf("\033[32mCreated thread:\033[0m %s (%s)\n", ui.Thread(thread.Title), thread.ID)
	case "open":
		if len(parts) < 3 {
			fmt.Println("Usage: /thread open <id|index>")
			return
		}
		thread, err := store.OpenThread(parts[2], history, cfg.CurrentModel)
		if err != nil {
			fmt.Printf("\033[31mError opening thread: %v\033[0m\n", err)
			return
		}
		fmt.Printf("\033[32mSwitched to thread:\033[0m %s (%s)\n", ui.Thread(thread.Title), thread.ID)
	case "rename":
		if len(parts) < 4 {
			fmt.Println("Usage: /thread rename <id|current> <new title>")
			return
		}
		thread, err := store.RenameThread(parts[2], strings.Join(parts[3:], " "))
		if err != nil {
			fmt.Printf("\033[31mError renaming thread: %v\033[0m\n", err)
			return
		}
		fmt.Printf("\033[32mRenamed thread:\033[0m %s (%s)\n", ui.Thread(thread.Title), thread.ID)
	case "current":
		fmt.Printf("Current thread: %s (%s)\n", ui.Thread(store.ActiveThreadTitle()), store.ActiveThreadID())
	default:
		fmt.Println("Usage: /thread [new|open|rename|current] ...")
	}
}

func handleSearch(parts []string, store *chat.ThreadStore) {
	if len(parts) < 2 {
		fmt.Println("Usage: /search <query> [--sort=updated|title|role|index] [--order=asc|desc]")
		return
	}

	query, sortBy, order := parseQuerySortOrder(parts[1:])
	if strings.TrimSpace(query) == "" {
		fmt.Println("Usage: /search <query> [--sort=updated|title|role|index] [--order=asc|desc]")
		return
	}

	results := store.Search(query, sortBy, order)
	if len(results) == 0 {
		fmt.Println("No history results matched.")
		return
	}

	if len(results) > 50 {
		results = results[:50]
	}

	fmt.Printf("History search results (%d):\n", len(results))
	for i, r := range results {
		idx := "thread"
		if r.MessageIdx >= 0 {
			idx = strconv.Itoa(r.MessageIdx)
		}
		fmt.Printf("%2d. [%s] %s | %s | msg=%s\n", i+1, r.ThreadID, ui.Thread(r.ThreadTitle), r.Role, idx)
		fmt.Printf("    %s\n", r.Snippet)
	}
}

func parseQuerySortOrder(args []string) (query string, sortBy string, order string) {
	queryParts := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "--sort=") {
			sortBy = strings.TrimSpace(strings.TrimPrefix(arg, "--sort="))
			continue
		}
		if strings.HasPrefix(arg, "--order=") {
			order = strings.TrimSpace(strings.TrimPrefix(arg, "--order="))
			continue
		}
		queryParts = append(queryParts, arg)
	}
	return strings.Join(queryParts, " "), sortBy, order
}
