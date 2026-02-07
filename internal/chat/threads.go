package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/bilbilaki/ai2go/internal/api"
)

const (
	appDataDir  = ".config/ai2go"
	threadsFile = "threads.json"
)

type Thread struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	AutoTitle bool          `json:"auto_title"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Messages  []api.Message `json:"messages"`
}

type SearchResult struct {
	ThreadID    string
	ThreadTitle string
	MessageIdx  int
	Role        string
	Snippet     string
	UpdatedAt   time.Time
}

type threadStoreData struct {
	ActiveThreadID string   `json:"active_thread_id"`
	Threads        []Thread `json:"threads"`
}

type ThreadStore struct {
	path string
	data threadStoreData
}

func NewThreadStore(currentModel string) (*ThreadStore, *History, error) {
	path, err := getThreadsPath()
	if err != nil {
		return nil, nil, err
	}

	store := &ThreadStore{path: path}

	if content, readErr := os.ReadFile(path); readErr == nil {
		if unmarshalErr := json.Unmarshal(content, &store.data); unmarshalErr != nil {
			return nil, nil, fmt.Errorf("failed to parse thread store: %w", unmarshalErr)
		}
	} else if !os.IsNotExist(readErr) {
		return nil, nil, fmt.Errorf("failed to read thread store: %w", readErr)
	}

	if len(store.data.Threads) == 0 {
		h := NewHistory(currentModel)
		thread := Thread{
			ID:        newThreadID(),
			Title:     fmt.Sprintf("New Thread %s", time.Now().Format("2006-01-02 15:04")),
			AutoTitle: true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Messages:  cloneMessages(h.GetMessages()),
		}
		store.data.Threads = []Thread{thread}
		store.data.ActiveThreadID = thread.ID
		if err := store.save(); err != nil {
			return nil, nil, err
		}
	}

	if store.findThreadIndex(store.data.ActiveThreadID) == -1 {
		store.data.ActiveThreadID = store.data.Threads[0].ID
		if err := store.save(); err != nil {
			return nil, nil, err
		}
	}

	history := NewHistory(currentModel)
	active := store.GetActiveThread()
	history.LoadMessages(active.Messages, currentModel)

	return store, history, nil
}

func (s *ThreadStore) ActiveThreadID() string {
	return s.data.ActiveThreadID
}

func (s *ThreadStore) ActiveThreadTitle() string {
	if t := s.GetActiveThread(); t != nil {
		return t.Title
	}
	return ""
}

func (s *ThreadStore) GetActiveThread() *Thread {
	idx := s.findThreadIndex(s.data.ActiveThreadID)
	if idx == -1 {
		return nil
	}
	return &s.data.Threads[idx]
}

func (s *ThreadStore) SyncActiveHistory(history *History) error {
	thread := s.GetActiveThread()
	if thread == nil {
		return fmt.Errorf("no active thread")
	}

	thread.Messages = cloneMessages(history.GetMessages())
	thread.UpdatedAt = time.Now().UTC()
	if thread.AutoTitle {
		if title := generateTitleFromMessages(thread.Messages); title != "" {
			thread.Title = title
		}
	}

	return s.save()
}

func (s *ThreadStore) NewThread(currentModel, explicitTitle string, history *History) (*Thread, error) {
	title := strings.TrimSpace(explicitTitle)
	auto := title == ""
	if auto {
		title = fmt.Sprintf("New Thread %s", time.Now().Format("2006-01-02 15:04"))
	}

	history.Clear(currentModel)
	now := time.Now().UTC()
	thread := Thread{
		ID:        newThreadID(),
		Title:     title,
		AutoTitle: auto,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  cloneMessages(history.GetMessages()),
	}

	s.data.Threads = append(s.data.Threads, thread)
	s.data.ActiveThreadID = thread.ID
	if err := s.save(); err != nil {
		return nil, err
	}

	return &thread, nil
}

func (s *ThreadStore) OpenThread(identifier string, history *History, currentModel string) (*Thread, error) {
	idx, err := s.resolveThreadIdentifier(identifier)
	if err != nil {
		return nil, err
	}

	s.data.ActiveThreadID = s.data.Threads[idx].ID
	history.LoadMessages(s.data.Threads[idx].Messages, currentModel)
	if err := s.save(); err != nil {
		return nil, err
	}
	return &s.data.Threads[idx], nil
}

func (s *ThreadStore) RenameThread(identifier, newTitle string) (*Thread, error) {
	title := strings.TrimSpace(newTitle)
	if title == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}

	idx, err := s.resolveThreadIdentifier(identifier)
	if err != nil {
		return nil, err
	}

	s.data.Threads[idx].Title = title
	s.data.Threads[idx].AutoTitle = false
	s.data.Threads[idx].UpdatedAt = time.Now().UTC()
	if err := s.save(); err != nil {
		return nil, err
	}

	return &s.data.Threads[idx], nil
}

func (s *ThreadStore) ListThreads(query, sortBy, order string) []Thread {
	q := strings.ToLower(strings.TrimSpace(query))
	items := make([]Thread, 0, len(s.data.Threads))
	for _, thread := range s.data.Threads {
		if q == "" || strings.Contains(strings.ToLower(thread.Title), q) || strings.Contains(strings.ToLower(thread.ID), q) {
			items = append(items, thread)
		}
	}

	sortBy = strings.ToLower(strings.TrimSpace(sortBy))
	if sortBy == "" {
		sortBy = "updated"
	}
	order = strings.ToLower(strings.TrimSpace(order))
	desc := order == "" || order == "desc"

	sort.SliceStable(items, func(i, j int) bool {
		var cmp int
		switch sortBy {
		case "created":
			cmp = items[i].CreatedAt.Compare(items[j].CreatedAt)
		case "title":
			cmp = strings.Compare(strings.ToLower(items[i].Title), strings.ToLower(items[j].Title))
		case "messages":
			if len(items[i].Messages) < len(items[j].Messages) {
				cmp = -1
			} else if len(items[i].Messages) > len(items[j].Messages) {
				cmp = 1
			}
		default:
			cmp = items[i].UpdatedAt.Compare(items[j].UpdatedAt)
		}
		if cmp == 0 {
			return items[i].ID < items[j].ID
		}
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})

	return items
}

func (s *ThreadStore) Search(query, sortBy, order string) []SearchResult {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}

	results := make([]SearchResult, 0)
	for _, thread := range s.data.Threads {
		if strings.Contains(strings.ToLower(thread.Title), q) {
			results = append(results, SearchResult{
				ThreadID:    thread.ID,
				ThreadTitle: thread.Title,
				MessageIdx:  -1,
				Role:        "thread",
				Snippet:     thread.Title,
				UpdatedAt:   thread.UpdatedAt,
			})
		}

		for i, msg := range thread.Messages {
			text := strings.TrimSpace(msg.Content)
			if text == "" {
				continue
			}
			if strings.Contains(strings.ToLower(text), q) {
				results = append(results, SearchResult{
					ThreadID:    thread.ID,
					ThreadTitle: thread.Title,
					MessageIdx:  i,
					Role:        msg.Role,
					Snippet:     buildSnippet(text, q),
					UpdatedAt:   thread.UpdatedAt,
				})
			}
		}
	}

	sortBy = strings.ToLower(strings.TrimSpace(sortBy))
	if sortBy == "" {
		sortBy = "updated"
	}
	order = strings.ToLower(strings.TrimSpace(order))
	desc := order == "" || order == "desc"

	sort.SliceStable(results, func(i, j int) bool {
		var cmp int
		switch sortBy {
		case "title":
			cmp = strings.Compare(strings.ToLower(results[i].ThreadTitle), strings.ToLower(results[j].ThreadTitle))
		case "role":
			cmp = strings.Compare(strings.ToLower(results[i].Role), strings.ToLower(results[j].Role))
		case "index":
			if results[i].MessageIdx < results[j].MessageIdx {
				cmp = -1
			} else if results[i].MessageIdx > results[j].MessageIdx {
				cmp = 1
			}
		default:
			cmp = results[i].UpdatedAt.Compare(results[j].UpdatedAt)
		}
		if cmp == 0 {
			if results[i].ThreadID == results[j].ThreadID {
				return results[i].MessageIdx < results[j].MessageIdx
			}
			return results[i].ThreadID < results[j].ThreadID
		}
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})

	return results
}

func (s *ThreadStore) findThreadIndex(id string) int {
	for i := range s.data.Threads {
		if s.data.Threads[i].ID == id {
			return i
		}
	}
	return -1
}

func (s *ThreadStore) resolveThreadIdentifier(identifier string) (int, error) {
	id := strings.TrimSpace(identifier)
	if id == "" || id == "current" {
		idx := s.findThreadIndex(s.data.ActiveThreadID)
		if idx == -1 {
			return -1, fmt.Errorf("active thread not found")
		}
		return idx, nil
	}

	if n, err := strconv.Atoi(id); err == nil {
		if n < 1 || n > len(s.data.Threads) {
			return -1, fmt.Errorf("thread index out of range")
		}
		return n - 1, nil
	}

	exact := -1
	prefixMatches := []int{}
	for i, thread := range s.data.Threads {
		if thread.ID == id {
			exact = i
			break
		}
		if strings.HasPrefix(thread.ID, id) {
			prefixMatches = append(prefixMatches, i)
		}
	}
	if exact != -1 {
		return exact, nil
	}
	if len(prefixMatches) == 1 {
		return prefixMatches[0], nil
	}
	if len(prefixMatches) > 1 {
		return -1, fmt.Errorf("thread id prefix is ambiguous")
	}

	return -1, fmt.Errorf("thread not found")
}

func (s *ThreadStore) save() error {
	content, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal thread store: %w", err)
	}
	if err := os.WriteFile(s.path, content, 0644); err != nil {
		return fmt.Errorf("failed to write thread store: %w", err)
	}
	return nil
}

func getThreadsPath() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return "", fmt.Errorf("cannot determine home directory")
	}

	dir := filepath.Join(home, appDataDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create app data dir: %w", err)
	}

	return filepath.Join(dir, threadsFile), nil
}

func cloneMessages(src []api.Message) []api.Message {
	out := make([]api.Message, len(src))
	copy(out, src)
	return out
}

func newThreadID() string {
	return fmt.Sprintf("th_%d", time.Now().UnixNano())
}

func generateTitleFromMessages(messages []api.Message) string {
	for _, msg := range messages {
		if msg.Role != "user" {
			continue
		}
		clean := strings.TrimSpace(msg.Content)
		if clean == "" {
			continue
		}
		clean = strings.Join(strings.Fields(clean), " ")
		if clean == "" {
			continue
		}

		runes := []rune(clean)
		if len(runes) > 96 {
			runes = runes[:96]
		}
		clean = strings.TrimSpace(string(runes))

		words := strings.Fields(clean)
		if len(words) > 10 {
			clean = strings.Join(words[:10], " ")
		}
		clean = strings.TrimFunc(clean, func(r rune) bool {
			return unicode.IsPunct(r) || unicode.IsSpace(r)
		})

		if clean != "" {
			return clean
		}
	}
	return ""
}

func buildSnippet(content, q string) string {
	plain := strings.Join(strings.Fields(content), " ")
	if plain == "" {
		return ""
	}

	lower := strings.ToLower(plain)
	idx := strings.Index(lower, q)
	if idx == -1 {
		runes := []rune(plain)
		if len(runes) > 80 {
			return string(runes[:80]) + "..."
		}
		return plain
	}

	start := idx - 30
	if start < 0 {
		start = 0
	}
	end := idx + len(q) + 50
	if end > len(plain) {
		end = len(plain)
	}
	runes := []rune(plain)
	runeStart := len([]rune(plain[:start]))
	runeEnd := len([]rune(plain[:end]))
	if runeStart < 0 {
		runeStart = 0
	}
	if runeEnd > len(runes) {
		runeEnd = len(runes)
	}

	snippet := string(runes[runeStart:runeEnd])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(plain) {
		snippet += "..."
	}
	return snippet
}
