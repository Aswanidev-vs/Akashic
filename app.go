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
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{}

	// Initialize modules
	app.EventBus = NewEventBus()
	app.SettingsManager = NewSettingsManager()
	app.FileManager = NewFileManager(app)

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

// GetInstalledModels returns list of installed Ollama models
func (a *App) GetInstalledModels() []OllamaModel {
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

// StartOllamaServer starts the Ollama server
func (a *App) StartOllamaServer() error {
	// Check if already running by making a request to the API
	_, err := http.Get("http://localhost:11434/api/tags")
	if err == nil {
		return fmt.Errorf("Ollama server is already running")
	}

	// Start the server
	cmd := exec.Command("ollama", "serve")
	cmd.SysProcAttr = hideConsoleWindows()

	// Start in background
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start Ollama server: %v", err)
	}

	// Wait a moment for server to start
	time.Sleep(2 * time.Second)

	// Verify it's running
	_, err = http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return fmt.Errorf("Ollama server failed to start properly")
	}

	return nil
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

// GenerateWithOllama sends a prompt to Ollama and returns the response
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
