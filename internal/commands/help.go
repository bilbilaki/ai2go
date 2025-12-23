package commands

import "fmt"

func ShowHelp() {
	fmt.Println("Commands:")
	fmt.Println("  \033[36m/models\033[0m         - Show available models and switch")
	fmt.Println("  \033[36m/current\033[0m        - Show current model")
	fmt.Println("  \033[36m/clear\033[0m          - Clear conversation history")
	fmt.Println("  \033[36m/file\033[0m           - add file content into chat")
	fmt.Println("  \033[36m/change_url\033[0m     - Change base URL")
	fmt.Println("  \033[36m/change_apikey\033[0m  - Change API key")
	fmt.Println("  \033[36m/proxy\033[0m          - Set proxy URL")
	fmt.Println("  \033[36m/autoaccept\033[0m     - Toggle auto-accept for commands")
	fmt.Println("  \033[36m/help\033[0m           - Show available commands")
	fmt.Println("  \033[36mexit/quit\033[0m       - Exit program")
}
