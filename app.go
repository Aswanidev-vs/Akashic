package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// hideConsoleWindows returns the correct SysProcAttr for the current OS
// to hide console windows when running external commands
func hideConsoleWindows() *syscall.SysProcAttr {
	if runtime.GOOS == "windows" {
		return &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
	}
	return nil
}

// App struct
type App struct {
	ctx             context.Context
	FileManager     *FileManager
	SettingsManager *SettingsManager
	EventBus        *EventBus
	ChatDB          *ChatDB
	activeRequests  map[string]context.CancelFunc
	ollamaProcess   *exec.Cmd
	ollamaMutex     sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{
		activeRequests: make(map[string]context.CancelFunc),
	}

	// Initialize modules
	app.EventBus = NewEventBus()
	app.SettingsManager = NewSettingsManager()
	app.FileManager = NewFileManager(app)

	// Initialize chat database
	var err error
	app.ChatDB, err = NewChatDB()
	if err != nil {
		fmt.Printf("Failed to initialize chat database: %v\n", err)
		// Continue without chat history if DB fails
	}

	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize settings
	if err := a.SettingsManager.Load(); err != nil {
		fmt.Printf("Failed to load settings: %v\n", err)
	}

	// Initialize file manager
	a.FileManager.Startup()

	// Publish startup event
	a.EventBus.Publish("app.startup", nil)
}

// ============================================
// Chat History API
// ============================================

// CreateChat creates a new chat session
func (a *App) CreateChat(title, modelName string) (*Chat, error) {
	if a.ChatDB == nil {
		return nil, fmt.Errorf("chat database not initialized")
	}
	return a.ChatDB.CreateChat(title, modelName)
}

// GetChats returns all chat sessions
func (a *App) GetChats() ([]Chat, error) {
	if a.ChatDB == nil {
		return []Chat{}, nil
	}
	return a.ChatDB.GetAllChats()
}

// GetChatMessages returns all messages for a chat
func (a *App) GetChatMessages(chatID int64) ([]Message, error) {
	if a.ChatDB == nil {
		return []Message{}, nil
	}
	return a.ChatDB.GetChatMessages(chatID)
}

// DeleteChat deletes a chat and all its messages
func (a *App) DeleteChat(chatID int64) error {
	if a.ChatDB == nil {
		return fmt.Errorf("chat database not initialized")
	}
	return a.ChatDB.DeleteChat(chatID)
}

// DeleteAllChats deletes all chats
func (a *App) DeleteAllChats() error {
	if a.ChatDB == nil {
		return fmt.Errorf("chat database not initialized")
	}
	return a.ChatDB.DeleteAllChats()
}

// UpdateChatTitle updates a chat's title
func (a *App) UpdateChatTitle(chatID int64, title string) error {
	if a.ChatDB == nil {
		return fmt.Errorf("chat database not initialized")
	}
	return a.ChatDB.UpdateChatTitle(chatID, title)
}

// AddMessage adds a message to a chat
func (a *App) AddMessage(chatID int64, role, content string) (*Message, error) {
	if a.ChatDB == nil {
		return nil, fmt.Errorf("chat database not initialized")
	}
	return a.ChatDB.AddMessage(chatID, role, content)
}

// GetChatContext builds context from recent messages for AI memory
func (a *App) GetChatContext(chatID int64, maxMessages int) (string, error) {
	if a.ChatDB == nil {
		return "", nil
	}
	return a.ChatDB.BuildContext(chatID, maxMessages)
}

// RenameChatFromFirstMessage auto-renames a chat based on first message
func (a *App) RenameChatFromFirstMessage(chatID int64) error {
	if a.ChatDB == nil {
		return fmt.Errorf("chat database not initialized")
	}
	return a.ChatDB.RenameChatFromFirstMessage(chatID)
}

// SearchChats searches chats by title
func (a *App) SearchChats(query string) ([]Chat, error) {
	if a.ChatDB == nil {
		return []Chat{}, nil
	}
	return a.ChatDB.SearchChats(query)
}

// ExportChat exports a chat as formatted text
func (a *App) ExportChat(chatID int64) (string, error) {
	if a.ChatDB == nil {
		return "", fmt.Errorf("chat database not initialized")
	}
	return a.ChatDB.ExportChat(chatID)
}

// GetSettings returns the current application settings
func (a *App) GetSettings() *Settings {
	return a.SettingsManager.Get()
}

// UpdateSettings updates the application settings
func (a *App) UpdateSettings(settings *Settings) error {
	return a.SettingsManager.Update(settings)
}

// NewFile creates a new empty file
func (a *App) NewFile() *FileInfo {
	fileInfo := a.FileManager.NewFile()
	a.EventBus.Publish(EventFileNew, FileEventData{FileInfo: fileInfo})
	return fileInfo
}

// FileOpenResult contains file info and content for opening files
type FileOpenResult struct {
	FileInfo *FileInfo `json:"fileInfo"`
	Content  string    `json:"content"`
}

// OpenFile shows open dialog and reads selected file
func (a *App) OpenFile() (*FileOpenResult, error) {
	filePath, err := a.FileManager.OpenFileDialog()
	if err != nil {
		return nil, err
	}
	if filePath == "" {
		return nil, nil // User cancelled
	}

	fileInfo, content, err := a.FileManager.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	a.EventBus.Publish(EventFileOpen, FileEventData{FileInfo: fileInfo, Content: content})
	return &FileOpenResult{FileInfo: fileInfo, Content: content}, nil
}

// OpenFileByPath opens a specific file path
func (a *App) OpenFileByPath(filePath string) (*FileOpenResult, error) {
	fileInfo, content, err := a.FileManager.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	a.EventBus.Publish(EventFileOpen, FileEventData{FileInfo: fileInfo, Content: content})
	return &FileOpenResult{FileInfo: fileInfo, Content: content}, nil
}

// SaveFile saves content to an existing file path
func (a *App) SaveFile(filePath string, content string, lineEnding string) (*FileInfo, error) {
	fileInfo, err := a.FileManager.WriteFile(filePath, content, lineEnding)
	if err != nil {
		return nil, err
	}

	a.EventBus.Publish(EventFileSave, FileEventData{FileInfo: fileInfo})
	return fileInfo, nil
}

// SaveFileAs shows save dialog and writes file
func (a *App) SaveFileAs(defaultName string, content string, lineEnding string) (*FileInfo, error) {
	filePath, err := a.FileManager.SaveFileDialog(defaultName)
	if err != nil {
		return nil, err
	}
	if filePath == "" {
		return nil, nil // User cancelled
	}

	fileInfo, err := a.FileManager.WriteFile(filePath, content, lineEnding)
	if err != nil {
		return nil, err
	}

	a.EventBus.Publish(EventFileSave, FileEventData{FileInfo: fileInfo})
	return fileInfo, nil
}

// GetRecentFiles returns the list of recent files
func (a *App) GetRecentFiles() []string {
	return a.FileManager.GetRecentFiles()
}

// ClearRecentFiles clears the recent files list
func (a *App) ClearRecentFiles() {
	a.FileManager.ClearRecentFiles()
}

// OnFileChange is called when file content changes in the editor
func (a *App) OnFileChange(filePath string, isDirty bool) {
	a.EventBus.Publish(EventFileChange, map[string]interface{}{
		"filePath": filePath,
		"isDirty":  isDirty,
	})
}

// OnEditorEvent publishes editor events (selection change, cursor move, etc.)
func (a *App) OnEditorEvent(eventType string, data EditorEventData) {
	a.EventBus.Publish(eventType, data)
}

// Greet returns a greeting for the given name (legacy method)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, Welcome to Akashic!", name)
}

// ============================================
// Ollama Integration
// ============================================

// OllamaStatus represents the installation status of Ollama
type OllamaStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Message   string `json:"message"`
}

// OllamaModel represents an installed Ollama model
type OllamaModel struct {
	Name       string `json:"name"`
	Size       string `json:"size"`
	Modified   string `json:"modified"`
	Parameters string `json:"parameters"`
}

// CheckOllamaInstalled checks if Ollama is installed on the system
func (a *App) CheckOllamaInstalled() OllamaStatus {
	// Try to run ollama --version
	cmd := exec.Command("ollama", "--version")
	cmd.SysProcAttr = hideConsoleWindows()
	output, err := cmd.CombinedOutput()

	if err != nil {
		return OllamaStatus{
			Installed: false,
			Version:   "",
			Message:   "Ollama is not installed. Please download from https://ollama.com/download",
		}
	}

	version := strings.TrimSpace(string(output))
	return OllamaStatus{
		Installed: true,
		Version:   version,
		Message:   "Ollama is installed and ready",
	}
}

// CheckOllamaServerRunning checks if the Ollama server is running via HTTP API
func (a *App) CheckOllamaServerRunning() bool {
	_, err := http.Get("http://localhost:11434/api/tags")
	return err == nil
}

// GetInstalledModels returns list of installed Ollama models
// Tries API first, falls back to CLI command
func (a *App) GetInstalledModels() []OllamaModel {
	// First try the HTTP API (works if server is running)
	if a.CheckOllamaServerRunning() {
		models, err := a.getModelsFromAPI()
		if err == nil && len(models) > 0 {
			return models
		}
	}

	// Fallback to CLI command
	return a.getModelsFromCLI()
}

// getModelsFromAPI fetches models from Ollama HTTP API
func (a *App) getModelsFromAPI() ([]OllamaModel, error) {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name       string    `json:"name"`
			Size       int64     `json:"size"`
			ModifiedAt time.Time `json:"modified_at"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var models []OllamaModel
	for _, m := range result.Models {
		model := OllamaModel{
			Name:     m.Name,
			Size:     formatBytes(m.Size),
			Modified: m.ModifiedAt.Format("2006-01-02 15:04:05"),
		}
		// Try to extract parameters from name (e.g., llama3:8b -> 8B)
		if idx := strings.Index(m.Name, ":"); idx != -1 {
			tag := m.Name[idx+1:]
			if strings.Contains(tag, "b") {
				model.Parameters = strings.ToUpper(tag)
			}
		}
		models = append(models, model)
	}

	return models, nil
}

// getModelsFromCLI fetches models using ollama list command
func (a *App) getModelsFromCLI() []OllamaModel {
	cmd := exec.Command("ollama", "list")
	cmd.SysProcAttr = hideConsoleWindows()
	output, err := cmd.CombinedOutput()

	if err != nil {
		return []OllamaModel{}
	}

	var models []OllamaModel
	lines := strings.Split(string(output), "\n")

	// Skip header line
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			model := OllamaModel{
				Name:     fields[0],
				Size:     fields[1] + " " + fields[2],
				Modified: strings.Join(fields[3:], " "),
			}
			// Try to extract parameters from name (e.g., llama3:8b -> 8B)
			if idx := strings.Index(fields[0], ":"); idx != -1 {
				tag := fields[0][idx+1:]
				if strings.Contains(tag, "b") {
					model.Parameters = strings.ToUpper(tag)
				}
			}
			models = append(models, model)
		}
	}

	return models
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// StartOllamaServer starts the Ollama server with proper process tracking
func (a *App) StartOllamaServer() error {
	a.ollamaMutex.Lock()
	defer a.ollamaMutex.Unlock()

	// Check if already running by making a request to the API
	if a.CheckOllamaServerRunning() {
		return fmt.Errorf("Ollama server is already running")
	}

	// Check if we already have a tracked process
	if a.ollamaProcess != nil && a.ollamaProcess.Process != nil {
		// Try to see if it's still running
		if err := a.ollamaProcess.Process.Signal(syscall.Signal(0)); err == nil {
			return fmt.Errorf("Ollama server is already running")
		}
		// Process is dead, clear it
		a.ollamaProcess = nil
	}

	// Start the server
	cmd := exec.Command("ollama", "serve")
	cmd.SysProcAttr = hideConsoleWindows()

	// Start in background
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start Ollama server: %v", err)
	}

	// Store process reference for later cleanup
	a.ollamaProcess = cmd

	// Wait for server to be ready with polling (max 30 seconds)
	client := &http.Client{Timeout: 2 * time.Second}
	startTime := time.Now()
	maxWait := 30 * time.Second
	checkInterval := 500 * time.Millisecond

	for time.Since(startTime) < maxWait {
		resp, err := client.Get("http://localhost:11434/api/tags")
		if err == nil {
			resp.Body.Close()
			return nil // Server is ready
		}
		time.Sleep(checkInterval)
	}

	// Server didn't start in time, kill the process
	if a.ollamaProcess != nil && a.ollamaProcess.Process != nil {
		a.ollamaProcess.Process.Kill()
		a.ollamaProcess = nil
	}

	return fmt.Errorf("Ollama server failed to start within %v", maxWait)
}

// StopOllamaServer stops the tracked Ollama server process
func (a *App) StopOllamaServer() error {
	a.ollamaMutex.Lock()
	defer a.ollamaMutex.Unlock()

	if a.ollamaProcess == nil || a.ollamaProcess.Process == nil {
		return nil // Nothing to stop
	}

	// Try graceful shutdown first
	if err := a.ollamaProcess.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, force kill
		if err := a.ollamaProcess.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop Ollama server: %v", err)
		}
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- a.ollamaProcess.Wait()
	}()

	select {
	case <-done:
		// Process exited
	case <-time.After(5 * time.Second):
		// Timeout, force kill
		a.ollamaProcess.Process.Kill()
	}

	a.ollamaProcess = nil
	return nil
}

// Shutdown performs cleanup when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	// Stop any active AI generation requests
	for requestID, cancel := range a.activeRequests {
		cancel()
		delete(a.activeRequests, requestID)
	}

	// Stop Ollama server if we started it
	a.StopOllamaServer()

	// Close chat database
	if a.ChatDB != nil {
		a.ChatDB.Close()
	}

	// Publish shutdown event
	if a.EventBus != nil {
		a.EventBus.Publish("app.shutdown", nil)
	}
}

// OllamaGenerateRequest represents a request to generate text
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaGenerateResponse represents the response from Ollama
type OllamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

// GenerateWithOllama sends a prompt to Ollama and returns the response (non-streaming)
func (a *App) GenerateWithOllama(model string, prompt string) (string, error) {
	// First check if server is running
	_, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return "", fmt.Errorf("Ollama server is not running. Please start it first.")
	}

	// Prepare request
	reqBody := OllamaGenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make request to Ollama API
	resp, err := http.Post("http://localhost:11434/api/generate",
		"application/json",
		bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// Parse response
	var result OllamaGenerateResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("Ollama error: %s", result.Error)
	}

	return result.Response, nil
}

// GenerateWithOllamaStream sends a prompt to Ollama and streams the response via events
// The frontend listens for "ai.stream.chunk" and "ai.stream.done" events
func (a *App) GenerateWithOllamaStream(requestID string, model string, prompt string, promptContext string) error {
	// First check if server is running
	_, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return fmt.Errorf("Ollama server is not running. Please start it first.")
	}

	// Build full prompt with context if provided
	fullPrompt := prompt
	if promptContext != "" {
		fullPrompt = fmt.Sprintf("Context:\n%s\n\nUser request: %s", promptContext, prompt)
	}

	// Prepare request with streaming enabled
	reqBody := OllamaGenerateRequest{
		Model:  model,
		Prompt: fullPrompt,
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	a.activeRequests[requestID] = cancel

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 0, // No timeout for streaming
	}

	// Create request with cancellable context
	req, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		delete(a.activeRequests, requestID)
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request in goroutine
	go func() {
		defer delete(a.activeRequests, requestID)

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() == context.Canceled {
				a.EventBus.Publish("ai.stream.done", map[string]string{
					"requestID": requestID,
					"reason":    "cancelled",
				})
				return
			}
			a.EventBus.Publish("ai.stream.error", map[string]string{
				"requestID": requestID,
				"error":     fmt.Sprintf("Error: %v", err),
			})
			return
		}
		defer resp.Body.Close()

		// Read streaming response line by line
		decoder := json.NewDecoder(resp.Body)
		for {
			select {
			case <-ctx.Done():
				// Request was cancelled
				a.EventBus.Publish("ai.stream.done", map[string]string{
					"requestID": requestID,
					"reason":    "cancelled",
				})
				return
			default:
				var chunk OllamaGenerateResponse
				if err := decoder.Decode(&chunk); err != nil {
					if err == io.EOF {
						// Stream completed successfully
						a.EventBus.Publish("ai.stream.done", map[string]string{
							"requestID": requestID,
						})
						return
					}
					if ctx.Err() == context.Canceled {
						a.EventBus.Publish("ai.stream.done", map[string]string{
							"requestID": requestID,
							"reason":    "cancelled",
						})
						return
					}
					a.EventBus.Publish("ai.stream.error", map[string]string{
						"requestID": requestID,
						"error":     fmt.Sprintf("[Error reading response: %v]", err),
					})
					return
				}

				if chunk.Error != "" {
					a.EventBus.Publish("ai.stream.error", map[string]string{
						"requestID": requestID,
						"error":     fmt.Sprintf("[Ollama error: %s]", chunk.Error),
					})
					return
				}

				// Publish chunk
				a.EventBus.Publish("ai.stream.chunk", map[string]string{
					"requestID": requestID,
					"chunk":     chunk.Response,
				})

				if chunk.Done {
					// Stream completed
					a.EventBus.Publish("ai.stream.done", map[string]string{
						"requestID": requestID,
					})
					return
				}
			}
		}
	}()

	return nil
}

// StopGeneration cancels an active generation request
func (a *App) StopGeneration(requestID string) {
	if cancel, exists := a.activeRequests[requestID]; exists {
		cancel()
		delete(a.activeRequests, requestID)
	}
}

// PullModel downloads a model from Ollama
func (a *App) PullModel(modelName string) error {
	cmd := exec.Command("ollama", "pull", modelName)
	cmd.SysProcAttr = hideConsoleWindows()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull model: %v\nOutput: %s", err, string(output))
	}
	return nil
}
