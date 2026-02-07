package commands

import (
	"fmt"

	"github.com/bilbilaki/ai2go/internal/ui"
)

func ShowHelp() {
	fmt.Println("Commands:")
	fmt.Println("  " + ui.HelpCommand("/models", "Show available models and switch"))
	fmt.Println("  " + ui.HelpCommand("/current", "Show current model"))
	fmt.Println("  " + ui.HelpCommand("/clear", "Clear conversation history"))
	fmt.Println("  " + ui.HelpCommand("/threads", "List threads (supports query, --sort, --order)"))
	fmt.Println("  " + ui.HelpCommand("/thread", "Thread ops: new/open/rename/current"))
	fmt.Println("  " + ui.HelpCommand("/search", "Search across thread titles and messages"))
	fmt.Println("  " + ui.HelpCommand("/file", "add file content into chat"))
	fmt.Println("  " + ui.HelpCommand("/change_url", "Change base URL"))
	fmt.Println("  " + ui.HelpCommand("/change_apikey", "Change API key"))
	fmt.Println("  " + ui.HelpCommand("/proxy", "Set proxy URL"))
	fmt.Println("  " + ui.HelpCommand("/autoaccept", "Toggle auto-accept for commands"))
	fmt.Println("  " + ui.HelpCommand("/subagent_experimental", "Toggle experimental subagent tool execution"))
	fmt.Println("  " + ui.HelpCommand("/summarize", "Summarize current thread and compact context"))
	fmt.Println("  " + ui.HelpCommand("/help", "Show available commands"))
	fmt.Println("  " + ui.HelpCommand("exit/quit", "Exit program"))
}
