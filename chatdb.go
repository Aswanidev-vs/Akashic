package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Chat represents a chat session
type Chat struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	ModelName string `json:"modelName"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Message represents a chat message
type Message struct {
	ID        int64  `json:"id"`
	ChatID    int64  `json:"chatId"`
	Role      string `json:"role"` // "user" or "assistant"
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

// ChatDB manages the SQLite database for chat history
type ChatDB struct {
	db *sql.DB
}

// NewChatDB creates a new ChatDB instance
func NewChatDB() (*ChatDB, error) {
	// Get app data directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	// Create app directory if it doesn't exist
	appDir := filepath.Join(homeDir, ".akashic")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create app directory: %v", err)
	}

	// Open database
	dbPath := filepath.Join(appDir, "chat_history.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	chatDB := &ChatDB{db: db}
	if err := chatDB.initTables(); err != nil {
		db.Close()
		return nil, err
	}

	return chatDB, nil
}

// initTables creates the necessary tables
func (c *ChatDB) initTables() error {
	// Create chats table
	_, err := c.db.Exec(`
		CREATE TABLE IF NOT EXISTS chats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			model_name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create chats table: %v", err)
	}

	// Create messages table
	_, err = c.db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id INTEGER NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('user', 'assistant')),
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create messages table: %v", err)
	}

	// Create index for faster queries
	_, err = c.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id)
	`)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	return nil
}

// CreateChat creates a new chat session
func (c *ChatDB) CreateChat(title, modelName string) (*Chat, error) {
	result, err := c.db.Exec(
		"INSERT INTO chats (title, model_name) VALUES (?, ?)",
		title, modelName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get chat ID: %v", err)
	}

	return c.GetChat(id)
}

// GetChat retrieves a chat by ID
func (c *ChatDB) GetChat(id int64) (*Chat, error) {
	var chat Chat
	err := c.db.QueryRow(
		"SELECT id, title, model_name, created_at, updated_at FROM chats WHERE id = ?",
		id,
	).Scan(&chat.ID, &chat.Title, &chat.ModelName, &chat.CreatedAt, &chat.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("chat not found")
		}
		return nil, fmt.Errorf("failed to get chat: %v", err)
	}

	return &chat, nil
}

// GetAllChats retrieves all chat sessions ordered by most recent
func (c *ChatDB) GetAllChats() ([]Chat, error) {
	rows, err := c.db.Query(
		"SELECT id, title, model_name, created_at, updated_at FROM chats ORDER BY updated_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats: %v", err)
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var chat Chat
		err := rows.Scan(&chat.ID, &chat.Title, &chat.ModelName, &chat.CreatedAt, &chat.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat: %v", err)
		}
		chats = append(chats, chat)
	}

	return chats, nil
}

// UpdateChatTitle updates the title of a chat
func (c *ChatDB) UpdateChatTitle(id int64, title string) error {
	_, err := c.db.Exec(
		"UPDATE chats SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		title, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update chat title: %v", err)
	}
	return nil
}

// UpdateChatModel updates the model of a chat
func (c *ChatDB) UpdateChatModel(id int64, modelName string) error {
	_, err := c.db.Exec(
		"UPDATE chats SET model_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		modelName, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update chat model: %v", err)
	}
	return nil
}

// DeleteChat deletes a chat and all its messages
func (c *ChatDB) DeleteChat(id int64) error {
	_, err := c.db.Exec("DELETE FROM chats WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete chat: %v", err)
	}
	return nil
}

// DeleteAllChats deletes all chats and messages
func (c *ChatDB) DeleteAllChats() error {
	_, err := c.db.Exec("DELETE FROM chats")
	if err != nil {
		return fmt.Errorf("failed to delete all chats: %v", err)
	}
	return nil
}

// AddMessage adds a message to a chat
func (c *ChatDB) AddMessage(chatID int64, role, content string) (*Message, error) {
	result, err := c.db.Exec(
		"INSERT INTO messages (chat_id, role, content) VALUES (?, ?, ?)",
		chatID, role, content,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add message: %v", err)
	}

	// Update chat's updated_at timestamp
	_, err = c.db.Exec(
		"UPDATE chats SET updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		chatID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update chat timestamp: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get message ID: %v", err)
	}

	return c.GetMessage(id)
}

// GetMessage retrieves a message by ID
func (c *ChatDB) GetMessage(id int64) (*Message, error) {
	var msg Message
	err := c.db.QueryRow(
		"SELECT id, chat_id, role, content, created_at FROM messages WHERE id = ?",
		id,
	).Scan(&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &msg.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %v", err)
	}

	return &msg, nil
}

// GetChatMessages retrieves all messages for a chat
func (c *ChatDB) GetChatMessages(chatID int64) ([]Message, error) {
	rows, err := c.db.Query(
		"SELECT id, chat_id, role, content, created_at FROM messages WHERE chat_id = ? ORDER BY created_at ASC",
		chatID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %v", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &msg.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %v", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetRecentMessages retrieves the last N messages for context
func (c *ChatDB) GetRecentMessages(chatID int64, limit int) ([]Message, error) {
	rows, err := c.db.Query(
		`SELECT id, chat_id, role, content, created_at FROM messages 
		WHERE chat_id = ? 
		ORDER BY created_at DESC 
		LIMIT ?`,
		chatID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent messages: %v", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &msg.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %v", err)
		}
		messages = append(messages, msg)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// BuildContext builds a context string from recent messages
func (c *ChatDB) BuildContext(chatID int64, maxMessages int) (string, error) {
	messages, err := c.GetRecentMessages(chatID, maxMessages)
	if err != nil {
		return "", err
	}

	if len(messages) == 0 {
		return "", nil
	}

	var context string
	for _, msg := range messages {
		role := "User"
		if msg.Role == "assistant" {
			role = "Assistant"
		}
		context += fmt.Sprintf("%s: %s\n\n", role, msg.Content)
	}

	return context, nil
}

// Close closes the database connection
func (c *ChatDB) Close() error {
	return c.db.Close()
}

// SearchChats searches chats by title
func (c *ChatDB) SearchChats(query string) ([]Chat, error) {
	rows, err := c.db.Query(
		`SELECT id, title, model_name, created_at, updated_at FROM chats 
		WHERE title LIKE ? 
		ORDER BY updated_at DESC`,
		"%"+query+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search chats: %v", err)
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var chat Chat
		err := rows.Scan(&chat.ID, &chat.Title, &chat.ModelName, &chat.CreatedAt, &chat.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat: %v", err)
		}
		chats = append(chats, chat)
	}

	return chats, nil
}

// RenameChat renames a chat based on first message content
func (c *ChatDB) RenameChatFromFirstMessage(chatID int64) error {
	// Get first user message
	var firstMessage string
	err := c.db.QueryRow(
		`SELECT content FROM messages 
		WHERE chat_id = ? AND role = 'user' 
		ORDER BY created_at ASC 
		LIMIT 1`,
		chatID,
	).Scan(&firstMessage)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil // No messages yet, keep default title
		}
		return fmt.Errorf("failed to get first message: %v", err)
	}

	// Truncate to create title (max 50 chars)
	title := firstMessage
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	return c.UpdateChatTitle(chatID, title)
}

// GetChatCount returns the total number of chats
func (c *ChatDB) GetChatCount() (int, error) {
	var count int
	err := c.db.QueryRow("SELECT COUNT(*) FROM chats").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count chats: %v", err)
	}
	return count, nil
}

// GetMessageCount returns the total number of messages for a chat
func (c *ChatDB) GetMessageCount(chatID int64) (int, error) {
	var count int
	err := c.db.QueryRow("SELECT COUNT(*) FROM messages WHERE chat_id = ?", chatID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %v", err)
	}
	return count, nil
}

// ExportChat exports a chat as a formatted string
func (c *ChatDB) ExportChat(chatID int64) (string, error) {
	chat, err := c.GetChat(chatID)
	if err != nil {
		return "", err
	}

	messages, err := c.GetChatMessages(chatID)
	if err != nil {
		return "", err
	}

	var export string
	export += fmt.Sprintf("Chat: %s\n", chat.Title)
	export += fmt.Sprintf("Model: %s\n", chat.ModelName)
	export += fmt.Sprintf("Created: %s\n", chat.CreatedAt)
	export += fmt.Sprintf("Updated: %s\n\n", chat.UpdatedAt)
	export += "========================================\n\n"

	for _, msg := range messages {
		role := "User"
		if msg.Role == "assistant" {
			role = "Assistant"
		}
		export += fmt.Sprintf("[%s] %s\n\n%s\n\n", msg.CreatedAt, role, msg.Content)
	}

	return export, nil
}
