package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/chat"
	"github.com/bilbilaki/ai2go/internal/config"
)

func HandleCommand(cmd string, history *chat.History, cfg *config.Config, apiClient *api.Client) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}
	
	command := parts[0]

	switch command {
	case "/models", "/model":
		HandleModelSelection(cfg, apiClient)
		
	case "/current":
		fmt.Printf("Current model: \033[36m%s\033[0m\n", cfg.CurrentModel)
		
	case "/clear":
		history.Clear(cfg.CurrentModel)
		fmt.Println("\033[32mConversation history cleared.\033[0m")
		
	case "/change_url":
		fmt.Print("Enter new Base URL: ")
		reader := bufio.NewReader(os.Stdin)
		newUrl, _ := reader.ReadString('\n')
		cfg.SetBaseURL(strings.TrimSpace(newUrl))
		fmt.Println("Base URL updated!")
		
	case "/change_apikey":
		fmt.Print("Enter new API Key: ")
		reader := bufio.NewReader(os.Stdin)
		newKey, _ := reader.ReadString('\n')
		cfg.SetAPIKey(strings.TrimSpace(newKey))
		fmt.Println("API Key updated!")
		
	case "/proxy":
		fmt.Print("Enter proxy URL (leave blank to disable): ")
		reader := bufio.NewReader(os.Stdin)
		newProxy, _ := reader.ReadString('\n')
		cfg.SetProxyURL(strings.TrimSpace(newProxy))
		if cfg.ProxyURL == "" {
			fmt.Println("Proxy disabled!")
		} else {
			fmt.Println("Proxy updated!")
		}
		
	case "/autoaccept":
		cfg.ToggleAutoAccept()
		status := "OFF"
		if cfg.AutoAccept {
			status = "ON"
		}
		fmt.Printf("Auto-accept commands is now: %s\n", status)
		
	case "/help":
		ShowHelp()
		
	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", command)
	}
}
