package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FileInfo represents metadata about an open file
type FileInfo struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	Encoding   string `json:"encoding"`
	LineEnding string `json:"lineEnding"` // "CRLF" or "LF"
	IsDirty    bool   `json:"isDirty"`
	IsNewFile  bool   `json:"isNewFile"`
	LastSaved  int64  `json:"lastSaved"`
}

// FileManager handles all file operations
type FileManager struct {
	app            *App
	recentFiles    []string
	maxRecentFiles int
	settingsDir    string
}

// NewFileManager creates a new FileManager instance
func NewFileManager(app *App) *FileManager {
	return &FileManager{
		app:            app,
		recentFiles:    make([]string, 0),
		maxRecentFiles: 10,
		settingsDir:    getSettingsDir(),
	}
}

// getSettingsDir returns the directory for storing settings
func getSettingsDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./.akashic"
	}
	return filepath.Join(homeDir, ".akashic")
}

// ensureSettingsDir creates the settings directory if it doesn't exist
func (fm *FileManager) ensureSettingsDir() error {
	return os.MkdirAll(fm.settingsDir, 0755)
}

// ReadFile opens and reads a file, returning content and metadata
func (fm *FileManager) ReadFile(filePath string) (*FileInfo, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		return nil, "", fmt.Errorf("failed to stat file: %w", err)
	}

	// Detect encoding and read content
	reader := bufio.NewReader(file)
	content, encoding, lineEnding, err := fm.readWithDetection(reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	fileInfo := &FileInfo{
		Path:       filePath,
		Name:       filepath.Base(filePath),
		Encoding:   encoding,
		LineEnding: lineEnding,
		IsDirty:    false,
		IsNewFile:  false,
		LastSaved:  stat.ModTime().Unix(),
	}

	// Add to recent files
	fm.addToRecentFiles(filePath)

	return fileInfo, content, nil
}

// readWithDetection reads content and detects encoding/line endings
func (fm *FileManager) readWithDetection(reader *bufio.Reader) (string, string, string, error) {
	var content strings.Builder
	var lineEnding string
	hasCRLF := false
	hasLF := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", "", "", err
		}

		// Detect line endings
		if strings.HasSuffix(line, "\r\n") {
			hasCRLF = true
			line = strings.TrimSuffix(line, "\r\n")
		} else if strings.HasSuffix(line, "\n") {
			hasLF = true
			line = strings.TrimSuffix(line, "\n")
		}

		content.WriteString(line)
		if err != io.EOF {
			content.WriteString("\n")
		}

		if err == io.EOF {
			break
		}
	}

	// Determine line ending type
	if hasCRLF && !hasLF {
		lineEnding = "CRLF"
	} else if hasLF && !hasCRLF {
		lineEnding = "LF"
	} else {
		lineEnding = "CRLF" // Default to CRLF on Windows
	}

	// For now, assume UTF-8. Could be extended to detect BOM
	encoding := "UTF-8"

	return content.String(), encoding, lineEnding, nil
}

// WriteFile saves content to a file
func (fm *FileManager) WriteFile(filePath string, content string, lineEnding string) (*FileInfo, error) {
	// Convert line endings if needed
	var normalizedContent string
	if lineEnding == "CRLF" {
		normalizedContent = strings.ReplaceAll(content, "\n", "\r\n")
	} else {
		normalizedContent = strings.ReplaceAll(content, "\r\n", "\n")
	}

	// Write to file
	err := os.WriteFile(filePath, []byte(normalizedContent), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Get updated file info
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	fileInfo := &FileInfo{
		Path:       filePath,
		Name:       filepath.Base(filePath),
		Encoding:   "UTF-8",
		LineEnding: lineEnding,
		IsDirty:    false,
		IsNewFile:  false,
		LastSaved:  stat.ModTime().Unix(),
	}

	// Add to recent files
	fm.addToRecentFiles(filePath)

	return fileInfo, nil
}

// NewFile creates a new empty file
func (fm *FileManager) NewFile() *FileInfo {
	return &FileInfo{
		Path:       "",
		Name:       "Untitled",
		Encoding:   "UTF-8",
		LineEnding: "CRLF",
		IsDirty:    false,
		IsNewFile:  true,
		LastSaved:  0,
	}
}

// OpenFileDialog shows a file open dialog and returns selected path
func (fm *FileManager) OpenFileDialog() (string, error) {
	selection, err := runtime.OpenFileDialog(fm.app.ctx, runtime.OpenDialogOptions{
		Title: "Open File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// SaveFileDialog shows a save file dialog and returns selected path
func (fm *FileManager) SaveFileDialog(defaultName string) (string, error) {
	selection, err := runtime.SaveFileDialog(fm.app.ctx, runtime.SaveDialogOptions{
		Title:           "Save File",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// addToRecentFiles adds a file to recent files list
func (fm *FileManager) addToRecentFiles(filePath string) {
	// Remove if already exists
	for i, path := range fm.recentFiles {
		if path == filePath {
			fm.recentFiles = append(fm.recentFiles[:i], fm.recentFiles[i+1:]...)
			break
		}
	}

	// Add to front
	fm.recentFiles = append([]string{filePath}, fm.recentFiles...)

	// Trim to max
	if len(fm.recentFiles) > fm.maxRecentFiles {
		fm.recentFiles = fm.recentFiles[:fm.maxRecentFiles]
	}

	// Save to disk
	fm.saveRecentFiles()
}

// GetRecentFiles returns the list of recent files
func (fm *FileManager) GetRecentFiles() []string {
	return fm.recentFiles
}

// loadRecentFiles loads recent files from disk
func (fm *FileManager) loadRecentFiles() error {
	fm.ensureSettingsDir()

	data, err := os.ReadFile(filepath.Join(fm.settingsDir, "recent_files.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No recent files yet
		}
		return err
	}

	return json.Unmarshal(data, &fm.recentFiles)
}

// saveRecentFiles saves recent files to disk
func (fm *FileManager) saveRecentFiles() error {
	fm.ensureSettingsDir()

	data, err := json.Marshal(fm.recentFiles)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(fm.settingsDir, "recent_files.json"), data, 0644)
}

// ClearRecentFiles clears the recent files list
func (fm *FileManager) ClearRecentFiles() {
	fm.recentFiles = make([]string, 0)
	fm.saveRecentFiles()
}

// Startup initializes the file manager
func (fm *FileManager) Startup() {
	fm.loadRecentFiles()
}
