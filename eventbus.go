package main

import (
	"sync"
)

// EventHandler is a function that handles events
type EventHandler func(data interface{})

// EventBus provides pub/sub functionality for decoupled communication
type EventBus struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// NewEventBus creates a new EventBus
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Subscribe registers a handler for an event type
func (eb *EventBus) Subscribe(event string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[event] = append(eb.handlers[event], handler)
}

// Unsubscribe removes a handler for an event type
func (eb *EventBus) Unsubscribe(event string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers := eb.handlers[event]
	for i, h := range handlers {
		// Compare function pointers (this is a simplification)
		// In production, you'd use a token-based system
		if &h == &handler {
			eb.handlers[event] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Publish emits an event to all subscribers
func (eb *EventBus) Publish(event string, data interface{}) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	handlers := eb.handlers[event]
	for _, handler := range handlers {
		// Run handlers in goroutines to prevent blocking
		go handler(data)
	}
}

// Event types for Akashic
const (
	// File events
	EventFileNew    = "file.new"
	EventFileOpen   = "file.open"
	EventFileSave   = "file.save"
	EventFileClose  = "file.close"
	EventFileChange = "file.change"

	// Editor events
	EventEditorChange    = "editor.change"
	EventEditorSelection = "editor.selection"
	EventEditorCursor    = "editor.cursor"
	EventEditorScroll    = "editor.scroll"
	EventEditorFocus     = "editor.focus"
	EventEditorBlur      = "editor.blur"

	// UI events
	EventThemeChange    = "theme.change"
	EventSettingsChange = "settings.change"
	EventZoomChange     = "zoom.change"

	// AI events
	EventAIRequest     = "ai.request"
	EventAIResponse    = "ai.response"
	EventAIError       = "ai.error"
	EventAIModelChange = "ai.model.change"

	// Extension events
	EventExtensionLoad    = "extension.load"
	EventExtensionUnload  = "extension.unload"
	EventExtensionEnable  = "extension.enable"
	EventExtensionDisable = "extension.disable"
)

// EventData structures
type FileEventData struct {
	FileInfo *FileInfo `json:"fileInfo"`
	Content  string    `json:"content,omitempty"`
}

type EditorEventData struct {
	FilePath   string `json:"filePath"`
	Selection  string `json:"selection,omitempty"`
	CursorLine int    `json:"cursorLine"`
	CursorCol  int    `json:"cursorCol"`
}

type AIEventData struct {
	Prompt      string `json:"prompt"`
	Model       string `json:"model"`
	Response    string `json:"response,omitempty"`
	Error       string `json:"error,omitempty"`
	IsStreaming bool   `json:"isStreaming"`
}
