package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// EditorSettings contains editor configuration
type EditorSettings struct {
	FontFamily          string  `json:"fontFamily"`
	FontSize            int     `json:"fontSize"`
	LineHeight          float64 `json:"lineHeight"`
	TabSize             int     `json:"tabSize"`
	UseSpaces           bool    `json:"useSpaces"`
	WordWrap            bool    `json:"wordWrap"`
	LineNumbers         bool    `json:"lineNumbers"`
	Minimap             bool    `json:"minimap"`
	AutoSave            bool    `json:"autoSave"`
	AutoSaveDelay       int     `json:"autoSaveDelay"` // seconds
	ShowWhitespace      bool    `json:"showWhitespace"`
	HighlightActiveLine bool    `json:"highlightActiveLine"`
}

// UISettings contains UI configuration
type UISettings struct {
	Theme           string `json:"theme"`
	DarkMode        bool   `json:"darkMode"`
	ShowStatusBar   bool   `json:"showStatusBar"`
	CompactMode     bool   `json:"compactMode"`
	SidebarPosition string `json:"sidebarPosition"` // "left" or "right"
}

// AISettings contains AI service configuration
type AISettings struct {
	Enabled         bool     `json:"enabled"`
	Endpoint        string   `json:"endpoint"`
	DefaultModel    string   `json:"defaultModel"`
	Temperature     float64  `json:"temperature"`
	MaxTokens       int      `json:"maxTokens"`
	AvailableModels []string `json:"availableModels"`
}

// Settings is the main configuration structure
type Settings struct {
	Editor EditorSettings `json:"editor"`
	UI     UISettings     `json:"ui"`
	AI     AISettings     `json:"ai"`
}

// DefaultSettings returns the default configuration
func DefaultSettings() *Settings {
	return &Settings{
		Editor: EditorSettings{
			FontFamily:          "Consolas, 'Courier New', monospace",
			FontSize:            14,
			LineHeight:          1.5,
			TabSize:             4,
			UseSpaces:           true,
			WordWrap:            true,
			LineNumbers:         true,
			Minimap:             false,
			AutoSave:            false,
			AutoSaveDelay:       5,
			ShowWhitespace:      false,
			HighlightActiveLine: true,
		},
		UI: UISettings{
			Theme:           "default-dark",
			DarkMode:        true,
			ShowStatusBar:   true,
			CompactMode:     false,
			SidebarPosition: "right",
		},
		AI: AISettings{
			Enabled:         true,
			Endpoint:        "http://localhost:11434",
			DefaultModel:    "mistral",
			Temperature:     0.7,
			MaxTokens:       2048,
			AvailableModels: []string{"mistral", "llama3", "gemma", "deepseek-coder"},
		},
	}
}

// SettingsManager handles configuration persistence
type SettingsManager struct {
	settings    *Settings
	settingsDir string
	configFile  string
}

// NewSettingsManager creates a new SettingsManager
func NewSettingsManager() *SettingsManager {
	settingsDir := getSettingsDir()
	return &SettingsManager{
		settings:    DefaultSettings(),
		settingsDir: settingsDir,
		configFile:  filepath.Join(settingsDir, "settings.json"),
	}
}

// Load reads settings from disk or creates defaults
func (sm *SettingsManager) Load() error {
	// Ensure directory exists
	if err := os.MkdirAll(sm.settingsDir, 0755); err != nil {
		return err
	}

	// Try to load existing settings
	data, err := os.ReadFile(sm.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default settings file
			return sm.Save()
		}
		return err
	}

	// Parse settings
	var loadedSettings Settings
	if err := json.Unmarshal(data, &loadedSettings); err != nil {
		return err
	}

	sm.settings = &loadedSettings
	return nil
}

// Save writes current settings to disk
func (sm *SettingsManager) Save() error {
	if err := os.MkdirAll(sm.settingsDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(sm.settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sm.configFile, data, 0644)
}

// Get returns the current settings
func (sm *SettingsManager) Get() *Settings {
	return sm.settings
}

// Update updates settings and saves to disk
func (sm *SettingsManager) Update(newSettings *Settings) error {
	sm.settings = newSettings
	return sm.Save()
}

// UpdateEditor updates editor settings
func (sm *SettingsManager) UpdateEditor(editor EditorSettings) error {
	sm.settings.Editor = editor
	return sm.Save()
}

// UpdateUI updates UI settings
func (sm *SettingsManager) UpdateUI(ui UISettings) error {
	sm.settings.UI = ui
	return sm.Save()
}

// UpdateAI updates AI settings
func (sm *SettingsManager) UpdateAI(ai AISettings) error {
	sm.settings.AI = ai
	return sm.Save()
}

// ResetToDefaults resets all settings to defaults
func (sm *SettingsManager) ResetToDefaults() error {
	sm.settings = DefaultSettings()
	return sm.Save()
}
