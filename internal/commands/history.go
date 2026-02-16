package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bilbilaki/ai2go/internal/chat"
	"github.com/bilbilaki/ai2go/internal/session"
	"github.com/bilbilaki/ai2go/internal/storage"
)

const (
	defaultChatTitle = "Untitled"
	maxPreviewLen    = 80
)

func HandleHistory(store *storage.Store) {
	if store == nil {
		fmt.Println("History is unavailable (storage not initialized).")
		return
	}
	chats, err := store.ListChats()
	if err != nil {
		fmt.Printf("Failed to load history: %v\n", err)
		return
	}
	if len(chats) == 0 {
		fmt.Println("No chat history found.")
		return
	}

	fmt.Println("\nChat History:")
	for _, chat := range chats {
		updated := chat.UpdatedAt.Local().Format(time.RFC3339)
		lastMessage := strings.TrimSpace(chat.LastMessage)
		if len(lastMessage) > maxPreviewLen {
			lastMessage = lastMessage[:maxPreviewLen] + "..."
		}
		fmt.Printf("[%d] %s | %s\n", chat.ID, updated, chat.Title)
		if lastMessage != "" {
			fmt.Printf("     %s\n", lastMessage)
		}
	}
}

func ResumeChat(history *chat.History, currentModel string, store *storage.Store, state *session.State, chatID int64) error {
	if store == nil {
		return fmt.Errorf("history is unavailable (storage not initialized)")
	}
	messages, err := store.LoadChatMessages(chatID)
	if err != nil {
		return err
	}
	title, err := store.GetChatTitle(chatID)
	if err != nil {
		return err
	}
	history.LoadMessages(messages, currentModel)
	if state != nil {
		state.ChatID = chatID
		state.Title = title
		state.HasMessages = len(messages) > 0
	}
	fmt.Printf("Resumed chat %d (%s).\n", chatID, title)
	return nil
}

func StartNewChat(history *chat.History, currentModel string, store *storage.Store, state *session.State) error {
	history.Clear(currentModel)
	if store == nil {
		if state != nil {
			state.ChatID = 0
			state.Title = defaultChatTitle
			state.HasMessages = false
		}
		fmt.Println("Started a new chat (history storage unavailable).")
		return nil
	}
	chatID, err := store.CreateChat(defaultChatTitle)
	if err != nil {
		return err
	}
	if state != nil {
		state.ChatID = chatID
		state.Title = defaultChatTitle
		state.HasMessages = false
	}
	fmt.Printf("Started new chat %d.\n", chatID)
	return nil
}
