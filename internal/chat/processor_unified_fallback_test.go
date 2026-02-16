package chat

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bilbilaki/ai2go/internal/api"
	"github.com/bilbilaki/ai2go/internal/config"
	"github.com/bilbilaki/ai2go/internal/tools"
)

func TestProcessConversationUnifiedDiffFailureSuggestsPatchFile(t *testing.T) {
	workTree := t.TempDir()

	var requestCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}

		n := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		switch n {
		case 1:
			args := fmt.Sprintf(`{"work_tree":"%s","patch":"@@ malformed patch fragment"}`, filepath.ToSlash(workTree))
			chunk := fmt.Sprintf(`{"choices":[{"delta":{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"apply_unified_diff_patch","arguments":%q}}]}}]}`, args)
			_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"done\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		}
	}))
	defer srv.Close()

	cfg := &config.Config{
		APIKey:       "test-key",
		BaseURL:      srv.URL,
		CurrentModel: "test-model",
		AutoAccept:   true,
	}
	client := api.NewClient(cfg)
	history := NewHistory(cfg.CurrentModel)
	history.AddUserMessage("edit one file only")

	toolsList := []api.Tool{
		tools.GetApplyUnifiedDiffPatchTool(),
		tools.GetPatchFileTool(),
		tools.GetReadFileTool(),
	}

	ProcessConversation(context.Background(), history, toolsList, cfg, client, NewPauseController())

	messages := history.GetMessages()
	foundHint := false
	for _, msg := range messages {
		if msg.Role != "tool" {
			continue
		}
		if strings.Contains(msg.Content, "Hint: unified diff parsing failed. Re-read target files and use patch_file for this edit.") {
			foundHint = true
			break
		}
	}
	if !foundHint {
		t.Fatalf("expected unified diff fallback hint in tool response; messages=%#v", messages)
	}
}

