package commands

import "github.com/chzyer/readline"

func NewAutoCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("/help"),
		readline.PcItem("/models"),
		readline.PcItem("/model"),
		readline.PcItem("/current"),
		readline.PcItem("/clear"),
		readline.PcItem("/summarize"),
		readline.PcItem("/setup"),
		readline.PcItem("/autoaccept"),
		readline.PcItem("/subagent_experimental"),
		readline.PcItem("/change_url"),
		readline.PcItem("/change_apikey"),
		readline.PcItem("/proxy"),
		readline.PcItem("/search"),
		readline.PcItem("/threads"),
		readline.PcItem("/thread",
			readline.PcItem("new"),
			readline.PcItem("open"),
			readline.PcItem("rename"),
			readline.PcItem("current"),
		),
	)
}
