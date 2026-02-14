package storage

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/bilbilaki/ai2go/internal/api"
)

type Store struct {
	path string
}

type ChatSummary struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	LastMessage string    `json:"last_message"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type chatSummaryRow struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	LastMessage string `json:"last_message"`
	UpdatedAt   string `json:"updated_at"`
}

type messageRow struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func Open(path string) (*Store, error) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return nil, fmt.Errorf("sqlite3 binary not found in PATH")
	}
	store := &Store{path: path}
	if err := store.initSchema(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) initSchema() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS chats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(chat_id) REFERENCES chats(id)
		);`,
	}
	_, err := s.execSQL(strings.Join(statements, "\n"))
	return err
}

func (s *Store) CreateChat(title string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	query := fmt.Sprintf(
		"INSERT INTO chats (title, created_at, updated_at) VALUES (%s, %s, %s); SELECT last_insert_rowid();",
		quote(title),
		quote(now),
		quote(now),
	)
	output, err := s.execSQL(query)
	if err != nil {
		return 0, err
	}
	var id int64
	if _, err := fmt.Sscanf(strings.TrimSpace(output), "%d", &id); err != nil {
		return 0, fmt.Errorf("failed to parse chat id: %w", err)
	}
	return id, nil
}

func (s *Store) UpdateChatTitle(chatID int64, title string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := fmt.Sprintf(
		"UPDATE chats SET title = %s, updated_at = %s WHERE id = %d;",
		quote(title),
		quote(now),
		chatID,
	)
	_, err := s.execSQL(query)
	return err
}

func (s *Store) TouchChat(chatID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := fmt.Sprintf("UPDATE chats SET updated_at = %s WHERE id = %d;", quote(now), chatID)
	_, err := s.execSQL(query)
	return err
}

func (s *Store) SaveMessage(chatID int64, role, content string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := fmt.Sprintf(
		"BEGIN; INSERT INTO messages (chat_id, role, content, created_at) VALUES (%d, %s, %s, %s); UPDATE chats SET updated_at = %s WHERE id = %d; COMMIT;",
		chatID,
		quote(role),
		quote(content),
		quote(now),
		quote(now),
		chatID,
	)
	_, err := s.execSQL(query)
	return err
}

func (s *Store) ListChats() ([]ChatSummary, error) {
	query := `SELECT c.id, c.title, c.updated_at,
		COALESCE((
			SELECT m.content FROM messages m
			WHERE m.chat_id = c.id
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT 1
		), '') AS last_message
	FROM chats c
	ORDER BY c.updated_at DESC, c.id DESC;`
	output, err := s.execJSON(query)
	if err != nil {
		return nil, err
	}
	var rows []chatSummaryRow
	if err := json.Unmarshal(output, &rows); err != nil {
		return nil, err
	}
	chats := make([]ChatSummary, 0, len(rows))
	for _, row := range rows {
		updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)
		chats = append(chats, ChatSummary{
			ID:          row.ID,
			Title:       row.Title,
			LastMessage: row.LastMessage,
			UpdatedAt:   updatedAt,
		})
	}
	return chats, nil
}

func (s *Store) GetChatTitle(chatID int64) (string, error) {
	query := fmt.Sprintf("SELECT title FROM chats WHERE id = %d;", chatID)
	output, err := s.execSQL(query)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (s *Store) LoadChatMessages(chatID int64) ([]api.Message, error) {
	query := fmt.Sprintf(
		"SELECT role, content FROM messages WHERE chat_id = %d ORDER BY created_at ASC, id ASC;",
		chatID,
	)
	output, err := s.execJSON(query)
	if err != nil {
		return nil, err
	}
	var rows []messageRow
	if err := json.Unmarshal(output, &rows); err != nil {
		return nil, err
	}
	messages := make([]api.Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, api.Message{
			Role:    row.Role,
			Content: row.Content,
		})
	}
	return messages, nil
}

func (s *Store) execSQL(query string) (string, error) {
	cmd := exec.Command("sqlite3", s.path, query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("sqlite3 error: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *Store) execJSON(query string) ([]byte, error) {
	cmd := exec.Command("sqlite3", "-json", s.path, query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sqlite3 error: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func quote(value string) string {
	escaped := strings.ReplaceAll(value, "'", "''")
	return "'" + escaped + "'"
}
