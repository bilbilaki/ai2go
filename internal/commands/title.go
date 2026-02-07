package commands

import (
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
)

func GenerateChatTitle(apiClient *api.Client, model, content string) (string, error) {
	prompt := "Generate a concise chat title (max 6 words) for this message. Respond with only the title."
	msgs := []api.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: content},
	}
	resp, err := apiClient.RunCompletion(msgs, nil, model)
	if err != nil {
		return "", err
	}
	title := strings.TrimSpace(resp.Content)
	title = strings.Trim(title, "\"")
	return title, nil
}
