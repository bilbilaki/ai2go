package commands

import (
	"context"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
)

func GenerateChatTitle(apiClient *api.Client, model, content string) (string, error) {
	prompt := "Generate a concise chat title (max 6 words) for this message. Respond with only the title."
	msgs := []api.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: content},
	}
	resp, err := apiClient.RunCompletionOnce(context.Background(), msgs, nil, model)
	if err != nil {
		return "", err
	}
	title := strings.TrimSpace(resp.Content)
	title = strings.Trim(title, "\"")
	return title, nil
}
