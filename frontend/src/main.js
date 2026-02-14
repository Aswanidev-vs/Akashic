import './style.css';
import './app.css';

// Wails bindings
import {
    NewFile,
    OpenFile,
    SaveFile,
    SaveFileAs,
    GetSettings,
    OnFileChange,
    Greet,
    CheckOllamaInstalled,
    CheckOllamaServerRunning,
    GetInstalledModels,
    StartOllamaServer,
    GenerateWithOllama,
    CreateChat,
    GetChats,
    GetChatMessages,
    DeleteChat,
    DeleteAllChats,
    UpdateChatTitle,
    AddMessage,
    GetChatContext,
    RenameChatFromFirstMessage,
    SearchChats,
    ExportChat
} from '../wailsjs/go/main/App.js';

console.log('Akashic Editor Starting...');

class AkashicEditor {
    constructor() {
        this.tabs = [];
        this.activeTabId = null;
        this.tabCounter = 0;
        this.zoomLevel = 100;
        this.ollamaStatus = { installed: false, checked: false };
        this.installedModels = [];
        this.selectedModel = '';
        this.serverRunning = false;
        this.currentGeneration = null;
        this.lastAIResponse = '';
        
        // Chat history state
        this.currentChatId = null;
        this.chats = [];
        this.isLoadingChat = false;
        
        // Wait for DOM
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => this.init());
        } else {
            this.init();
        }
    }
    
    async init() {
        console.log('Initializing Akashic Editor...');
        
        // Cache DOM elements
        this.elements = {
            tabsContainer: document.getElementById('tabs-container'),
            newTabBtn: document.getElementById('new-tab-btn'),
            editorContainer: document.getElementById('editor-container'),
            aiSidebar: document.getElementById('ai-sidebar'),
            statusFile: document.getElementById('status-file'),
            statusEncoding: document.getElementById('status-encoding'),
            statusLineEnding: document.getElementById('status-line-ending'),
            statusPosition: document.getElementById('status-position'),
            statusZoom: document.getElementById('status-zoom'),
            contextMenu: document.getElementById('context-menu'),
            dialogOverlay: document.getElementById('dialog-overlay'),
            // Chat history elements
            chatHistoryPanel: document.getElementById('chat-history-panel'),
            chatList: document.getElementById('chat-list'),
            chatSearch: document.getElementById('chat-search'),
            currentChatTitle: document.getElementById('current-chat-title'),
            chatInfo: document.getElementById('chat-info')
        };
        
        // Verify critical elements
        if (!this.elements.editorContainer) {
            console.error('Editor container not found!');
            document.body.innerHTML = '<div style="color: red; padding: 20px;">Error: Editor container not found</div>';
            return;
        }
        
        this.setupMenus();
        this.setupKeyboardShortcuts();
        this.setupEventListeners();
        this.setupAIEventListeners();
        this.setupChatHistoryListeners();
        
        // Create initial tab
        this.createNewTab();
        
        // Test Wails connection
        if (Greet) {
            try {
                const result = await Greet('Developer');
                console.log('Wails connected:', result);
            } catch (e) {
                console.warn('Wails greeting failed:', e);
            }
        }
        
        // Load chat history
        await this.loadChatHistory();
        
        console.log('Editor initialized successfully');
    }
    
    // ============================================
    // Chat History Management
    // ============================================
    
    async loadChatHistory() {
        try {
            this.chats = await GetChats();
            this.renderChatList();
        } catch (err) {
            console.error('Failed to load chat history:', err);
        }
    }
    
    renderChatList() {
        if (!this.elements.chatList) return;
        
        this.elements.chatList.innerHTML = '';
        
        if (this.chats.length === 0) {
            this.elements.chatList.innerHTML = `
                <div class="chat-empty">
                    <p>No chats yet</p>
                    <span>Start a new conversation</span>
                </div>
            `;
            return;
        }
        
        this.chats.forEach(chat => {
            const chatItem = document.createElement('div');
            chatItem.className = 'chat-item';
            chatItem.dataset.chatId = chat.id;
            
            if (chat.id === this.currentChatId) {
                chatItem.classList.add('active');
            }
            
            const date = new Date(chat.updatedAt);
            const timeStr = date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
            
            chatItem.innerHTML = `
                <div class="chat-item-content">
                    <div class="chat-item-title">${this.escapeHtml(chat.title)}</div>
                    <div class="chat-item-meta">
                        <span class="chat-item-model">${chat.modelName}</span>
                        <span class="chat-item-time">${timeStr}</span>
                    </div>
                </div>
                <button class="chat-item-delete" title="Delete chat">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M18 6L6 18M6 6l12 12"/>
                    </svg>
                </button>
            `;
            
            // Click to load chat
            chatItem.querySelector('.chat-item-content').addEventListener('click', () => {
                this.loadChat(chat.id);
            });
            
            // Delete button
            chatItem.querySelector('.chat-item-delete').addEventListener('click', (e) => {
                e.stopPropagation();
                this.deleteChat(chat.id);
            });
            
            this.elements.chatList.appendChild(chatItem);
        });
    }
    
    async createNewChat() {
        try {
            const title = 'New Chat';
            const chat = await CreateChat(title, this.selectedModel || 'default');
            this.currentChatId = chat.id;
            this.chats.unshift(chat);
            
            // Clear messages
            const messagesDiv = document.getElementById('ai-messages');
            messagesDiv.innerHTML = `
                <div class="ai-welcome">
                    <div class="ai-welcome-icon">🤖</div>
                    <h3>How can I help you today?</h3>
                    <p>I can help you write, edit, explain code, summarize text, and more.</p>
                    <div class="ai-suggestions">
                        <button class="ai-suggestion" data-prompt="Explain this code">💡 Explain code</button>
                        <button class="ai-suggestion" data-prompt="Rewrite this to be more professional">✍️ Rewrite professionally</button>
                        <button class="ai-suggestion" data-prompt="Summarize this text">📝 Summarize</button>
                        <button class="ai-suggestion" data-prompt="Fix grammar and spelling">🔧 Fix grammar</button>
                    </div>
                </div>
            `;
            
            // Re-attach suggestion listeners
            messagesDiv.querySelectorAll('.ai-suggestion').forEach(btn => {
                btn.addEventListener('click', () => {
                    const prompt = btn.dataset.prompt;
                    document.getElementById('ai-prompt').value = prompt;
                    this.generateWithAI();
                });
            });
            
            this.renderChatList();
            this.updateChatInfo();
        } catch (err) {
            console.error('Failed to create chat:', err);
            this.showNotification('Failed to create chat', 'error');
        }
    }
    
    async loadChat(chatId) {
        if (this.isLoadingChat) return;
        this.isLoadingChat = true;
        
        try {
            this.currentChatId = chatId;
            const messages = await GetChatMessages(chatId);
            
            // Clear current messages
            const messagesDiv = document.getElementById('ai-messages');
            messagesDiv.innerHTML = '';
            
            if (messages.length === 0) {
                // Show welcome if no messages
                messagesDiv.innerHTML = `
                    <div class="ai-welcome">
                        <div class="ai-welcome-icon">🤖</div>
                        <h3>How can I help you today?</h3>
                        <p>I can help you write, edit, explain code, summarize text, and more.</p>
                    </div>
                `;
            } else {
                // Render messages
                messages.forEach(msg => {
                    const msgDiv = document.createElement('div');
                    msgDiv.className = `ai-message ${msg.role}`;
                    msgDiv.innerHTML = `<div class="ai-message-content">${this.escapeHtml(msg.content)}</div>`;
                    messagesDiv.appendChild(msgDiv);
                });
                
                // Scroll to bottom
                messagesDiv.scrollTop = messagesDiv.scrollHeight;
            }
            
            // Update chat info
            const chat = this.chats.find(c => c.id === chatId);
            if (chat) {
                this.selectedModel = chat.modelName;
                document.getElementById('ai-model').value = chat.modelName;
            }
            
            this.renderChatList();
            this.updateChatInfo();
        } catch (err) {
            console.error('Failed to load chat:', err);
            this.showNotification('Failed to load chat', 'error');
        } finally {
            this.isLoadingChat = false;
        }
    }
    
    async deleteChat(chatId) {
        const chat = this.chats.find(c => c.id === chatId);
        const chatTitle = chat ? chat.title : 'this chat';
        
        this.showStyledConfirmDialog(
            'Delete Chat',
            `Are you sure you want to delete "${chatTitle}"?`,
            'Delete',
            'Cancel',
            async () => {
                try {
                    await DeleteChat(chatId);
                    this.chats = this.chats.filter(c => c.id !== chatId);
                    
                    if (this.currentChatId === chatId) {
                        this.currentChatId = null;
                        // Clear messages
                        const messagesDiv = document.getElementById('ai-messages');
                        messagesDiv.innerHTML = `
                            <div class="ai-welcome">
                                <div class="ai-welcome-icon">🤖</div>
                                <h3>How can I help you today?</h3>
                            </div>
                        `;
                        this.updateChatInfo();
                    }
                    
                    this.renderChatList();
                    this.showNotification('Chat deleted', 'success');
                } catch (err) {
                    console.error('Failed to delete chat:', err);
                    this.showNotification('Failed to delete chat', 'error');
                }
            }
        );
    }
    
    async clearAllChats() {
        this.showStyledConfirmDialog(
            'Clear All Chats',
            `Are you sure you want to delete ALL ${this.chats.length} chat(s)? This action cannot be undone.`,
            'Delete All',
            'Cancel',
            async () => {
                try {
                    await DeleteAllChats();
                    this.chats = [];
                    this.currentChatId = null;
                    
                    // Clear messages
                    const messagesDiv = document.getElementById('ai-messages');
                    messagesDiv.innerHTML = `
                        <div class="ai-welcome">
                            <div class="ai-welcome-icon">🤖</div>
                            <h3>How can I help you today?</h3>
                        </div>
                    `;
                    
                    this.renderChatList();
                    this.updateChatInfo();
                    this.showNotification('All chats cleared', 'success');
                } catch (err) {
                    console.error('Failed to clear chats:', err);
                    this.showNotification('Failed to clear chats', 'error');
                }
            }
        );
    }
    
    async searchChats(query) {
        if (!query.trim()) {
            await this.loadChatHistory();
            return;
        }
        
        try {
            this.chats = await SearchChats(query);
            this.renderChatList();
        } catch (err) {
            console.error('Failed to search chats:', err);
        }
    }
    
    async renameCurrentChat() {
        if (!this.currentChatId) return;
        
        const chat = this.chats.find(c => c.id === this.currentChatId);
        if (!chat) return;
        
        this.showRenameChatDialog(chat.title, async (newTitle) => {
            if (!newTitle || newTitle === chat.title) return;
            
            try {
                await UpdateChatTitle(this.currentChatId, newTitle);
                chat.title = newTitle;
                this.renderChatList();
                this.updateChatInfo();
                this.showNotification('Chat renamed', 'success');
            } catch (err) {
                console.error('Failed to rename chat:', err);
                this.showNotification('Failed to rename chat', 'error');
            }
        });
    }
    
    showRenameChatDialog(currentTitle, onConfirm) {
        this.elements.dialogOverlay.classList.remove('hidden');
        
        const dialogId = 'dialog-rename-chat';
        let dialog = document.getElementById(dialogId);
        if (!dialog) {
            dialog = document.createElement('div');
            dialog.id = dialogId;
            dialog.className = 'dialog hidden';
            dialog.style.width = '350px';
            dialog.innerHTML = `
                <div class="dialog-header" style="display: flex; align-items: center; gap: 10px;">
                    <span style="font-size: 20px;">✏️</span>
                    <span>Rename Chat</span>
                </div>
                <div class="dialog-body" style="padding: 20px;">
                    <input type="text" id="rename-chat-input" placeholder="Enter chat name..." style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 4px; background: var(--bg-tertiary); color: var(--text-primary); font-size: 14px;">
                </div>
                <div class="dialog-footer" style="justify-content: flex-end; gap: 10px; padding: 15px 20px;">
                    <button id="rename-chat-cancel" style="padding: 6px 16px; background: transparent; color: var(--text-secondary); border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer;">Cancel</button>
                    <button id="rename-chat-save" style="padding: 6px 16px; background: var(--accent-color); color: white; border: none; border-radius: 4px; cursor: pointer;">Save</button>
                </div>
            `;
            this.elements.dialogOverlay.appendChild(dialog);
        }
        
        const input = document.getElementById('rename-chat-input');
        input.value = currentTitle;
        
        dialog.classList.remove('hidden');
        
        // Focus and select text
        setTimeout(() => {
            input.focus();
            input.select();
        }, 10);
        
        const saveBtn = document.getElementById('rename-chat-save');
        const cancelBtn = document.getElementById('rename-chat-cancel');
        
        // Remove old listeners
        const newSaveBtn = saveBtn.cloneNode(true);
        const newCancelBtn = cancelBtn.cloneNode(true);
        saveBtn.parentNode.replaceChild(newSaveBtn, saveBtn);
        cancelBtn.parentNode.replaceChild(newCancelBtn, cancelBtn);
        
        const closeDialog = () => {
            this.hideDialogs();
            dialog.classList.add('hidden');
        };
        
        newSaveBtn.addEventListener('click', () => {
            const newTitle = input.value.trim();
            closeDialog();
            if (onConfirm) onConfirm(newTitle);
        });
        
        newCancelBtn.addEventListener('click', () => {
            closeDialog();
        });
        
        // Handle Enter key
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                const newTitle = input.value.trim();
                closeDialog();
                if (onConfirm) onConfirm(newTitle);
            } else if (e.key === 'Escape') {
                closeDialog();
            }
        });
    }
    
    updateChatInfo() {
        if (!this.elements.chatInfo || !this.elements.currentChatTitle) return;
        
        if (this.currentChatId) {
            const chat = this.chats.find(c => c.id === this.currentChatId);
            if (chat) {
                this.elements.currentChatTitle.textContent = chat.title;
                this.elements.chatInfo.classList.remove('hidden');
            }
        } else {
            this.elements.chatInfo.classList.add('hidden');
        }
    }
    
    setupChatHistoryListeners() {
        // New chat button
        const newChatBtn = document.getElementById('new-chat-btn');
        if (newChatBtn) {
            newChatBtn.addEventListener('click', () => this.createNewChat());
        }
        
        // Clear all chats
        const clearAllBtn = document.getElementById('clear-all-chats');
        if (clearAllBtn) {
            clearAllBtn.addEventListener('click', () => this.clearAllChats());
        }
        
        // Search chats
        if (this.elements.chatSearch) {
            this.elements.chatSearch.addEventListener('input', (e) => {
                this.searchChats(e.target.value);
            });
        }
        
        // Rename chat button
        const renameBtn = document.getElementById('rename-chat-btn');
        if (renameBtn) {
            renameBtn.addEventListener('click', () => this.renameCurrentChat());
        }
    }
    
    // ============================================
    // Tab Management
    // ============================================
    
    createNewTab(fileInfo = null, content = '') {
        const tabId = `tab-${++this.tabCounter}`;
        console.log('Creating new tab:', tabId);
        
        const tab = {
            id: tabId,
            fileInfo: fileInfo || {
                Path: '',
                Name: 'Untitled',
                Encoding: 'UTF-8',
                LineEnding: 'CRLF',
                IsDirty: false,
                IsNewFile: true
            },
            content: content || '',
            textarea: null
        };
        
        this.tabs.push(tab);
        this.renderTab(tab);
        this.switchToTab(tabId);
        
        return tab;
    }
    
    renderTab(tab) {
        const tabElement = document.createElement('div');
        tabElement.className = 'tab';
        tabElement.dataset.tabId = tab.id;
        tabElement.innerHTML = `
            <span class="tab-dirty" style="display: none;">●</span>
            <span class="tab-name">${tab.fileInfo.Name}</span>
            <span class="tab-close">×</span>
        `;
        
        tabElement.addEventListener('click', (e) => {
            if (e.target.classList.contains('tab-close')) {
                this.closeTab(tab.id);
            } else {
                this.switchToTab(tab.id);
            }
        });
        
        this.elements.tabsContainer.appendChild(tabElement);
        tab.element = tabElement;
    }
    
    switchToTab(tabId) {
        console.log('Switching to tab:', tabId);
        
        // Save current editor content
        if (this.activeTabId) {
            const currentTab = this.tabs.find(t => t.id === this.activeTabId);
            if (currentTab && currentTab.textarea) {
                currentTab.content = currentTab.textarea.value;
                currentTab.element.classList.remove('active');
            }
        }
        
        // Activate new tab
        this.activeTabId = tabId;
        const tab = this.tabs.find(t => t.id === tabId);
        
        if (tab) {
            tab.element.classList.add('active');
            
            // Clear editor container
            this.elements.editorContainer.innerHTML = '';
            
            // Create textarea editor
            this.createTextareaEditor(tab);
            
            this.updateStatusBar();
        }
    }
    
    createTextareaEditor(tab) {
        console.log('Creating textarea editor for:', tab.id);
        
        const textarea = document.createElement('textarea');
        textarea.className = 'editor-textarea';
        textarea.value = tab.content;
        textarea.spellcheck = false;
        textarea.wrap = 'off';
        
        // Apply current zoom
        textarea.style.fontSize = `${this.zoomLevel}%`;
        
        // Event handlers
        textarea.addEventListener('input', () => {
            this.onEditorChange(tab);
        });
        
        textarea.addEventListener('click', () => {
            this.updateCursorPosition(tab);
        });
        
        textarea.addEventListener('keyup', () => {
            this.updateCursorPosition(tab);
        });
        
        textarea.addEventListener('keydown', (e) => {
            // Handle Tab key
            if (e.key === 'Tab') {
                e.preventDefault();
                const start = textarea.selectionStart;
                const end = textarea.selectionEnd;
                textarea.value = textarea.value.substring(0, start) + '\t' + textarea.value.substring(end);
                textarea.selectionStart = textarea.selectionEnd = start + 1;
                this.onEditorChange(tab);
            }
        });
        
        this.elements.editorContainer.appendChild(textarea);
        tab.textarea = textarea;
        
        // Focus the textarea
        setTimeout(() => {
            textarea.focus();
            console.log('Textarea focused');
        }, 10);
    }
    
    onEditorChange(tab) {
        tab.content = tab.textarea.value;
        
        if (!tab.fileInfo.IsDirty) {
            tab.fileInfo.IsDirty = true;
            const dirtyIndicator = tab.element.querySelector('.tab-dirty');
            if (dirtyIndicator) dirtyIndicator.style.display = 'inline';
        }
        
        // Notify backend if available
        if (OnFileChange) {
            try {
                OnFileChange(tab.fileInfo.Path, true);
            } catch (e) {
                // Ignore
            }
        }
    }
    
    updateCursorPosition(tab) {
        if (!tab.textarea) return;
        
        const text = tab.textarea.value;
        const pos = tab.textarea.selectionStart;
        const lines = text.substring(0, pos).split('\n');
        const lineNum = lines.length;
        const colNum = lines[lines.length - 1].length + 1;
        
        this.elements.statusPosition.textContent = `Ln ${lineNum}, Col ${colNum}`;
    }
    
    closeTab(tabId) {
        const tabIndex = this.tabs.findIndex(t => t.id === tabId);
        if (tabIndex === -1) return;
        
        const tab = this.tabs[tabIndex];
        
        if (tab.fileInfo.IsDirty) {
            this.showSaveConfirmDialog(
                'Unsaved Changes',
                `Save changes to ${tab.fileInfo.Name}?`,
                () => this.saveAndCloseTab(tab, tabId),
                () => this.forceCloseTab(tabId)
            );
            return;
        }
        
        this.forceCloseTab(tabId);
    }
    
    async saveAndCloseTab(tab, tabId) {
        await this.saveTab(tab);
        this.forceCloseTab(tabId);
    }
    
    forceCloseTab(tabId) {
        const tabIndex = this.tabs.findIndex(t => t.id === tabId);
        if (tabIndex === -1) return;
        
        const tab = this.tabs[tabIndex];
        tab.element.remove();
        this.tabs.splice(tabIndex, 1);
        
        if (this.activeTabId === tabId) {
            if (this.tabs.length > 0) {
                const newIndex = Math.min(tabIndex, this.tabs.length - 1);
                this.switchToTab(this.tabs[newIndex].id);
            } else {
                this.activeTabId = null;
                this.createNewTab();
            }
        }
    }
    
    // ============================================
    // File Operations
    // ============================================
    
    async newFile() {
        console.log('Creating new file');
        this.createNewTab();
    }
    
    async openFile() {
        console.log('Opening file...');
        
        if (!OpenFile) {
            this.showNotification('File dialog not available', 'error');
            return;
        }
        
        try {
            const result = await OpenFile();
            console.log('OpenFile result:', result);
            
            if (!result) {
                console.log('No file selected');
                return;
            }
            
            // Handle new FileOpenResult structure
            const fileInfo = result.fileInfo;
            const content = result.content || '';
            
            if (fileInfo) {
                console.log('File opened:', fileInfo.name);
                // Convert backend field names (lowercase) to frontend field names (PascalCase)
                const normalizedFileInfo = {
                    Path: fileInfo.path || '',
                    Name: fileInfo.name || 'Untitled',
                    Encoding: fileInfo.encoding || 'UTF-8',
                    LineEnding: fileInfo.lineEnding || 'CRLF',
                    IsDirty: fileInfo.isDirty || false,
                    IsNewFile: fileInfo.isNewFile || false
                };
                this.createNewTab(normalizedFileInfo, content);
            }
        } catch (err) {
            console.error('Failed to open file:', err);
            this.showNotification('Failed to open file: ' + (err.message || err), 'error');
        }
    }
    
    async saveTab(tab = null) {
        if (!tab) tab = this.tabs.find(t => t.id === this.activeTabId);
        if (!tab || !tab.textarea) return;
        
        const content = tab.textarea.value;
        console.log('Saving file:', tab.fileInfo.Name);
        
        if (!SaveFile || !SaveFileAs) {
            this.showNotification('Save not available', 'error');
            return;
        }
        
        try {
            let fileInfo;
            
            if (tab.fileInfo.IsNewFile || !tab.fileInfo.Path) {
                fileInfo = await SaveFileAs(tab.fileInfo.Name, content, tab.fileInfo.LineEnding);
            } else {
                fileInfo = await SaveFile(tab.fileInfo.Path, content, tab.fileInfo.LineEnding);
            }
            
            if (fileInfo) {
                // Normalize fileInfo from backend (lowercase) to frontend (PascalCase)
                const normalizedFileInfo = {
                    Path: fileInfo.path || fileInfo.Path || '',
                    Name: fileInfo.name || fileInfo.Name || 'Untitled',
                    Encoding: fileInfo.encoding || fileInfo.Encoding || 'UTF-8',
                    LineEnding: fileInfo.lineEnding || fileInfo.LineEnding || 'CRLF',
                    IsDirty: false,
                    IsNewFile: fileInfo.isNewFile || fileInfo.IsNewFile || false
                };
                tab.fileInfo = normalizedFileInfo;
                
                const nameEl = tab.element.querySelector('.tab-name');
                const dirtyEl = tab.element.querySelector('.tab-dirty');
                if (nameEl) nameEl.textContent = normalizedFileInfo.Name;
                if (dirtyEl) dirtyEl.style.display = 'none';
                
                this.updateStatusBar();
                this.showNotification('File saved: ' + normalizedFileInfo.Name, 'success');
                console.log('File saved:', normalizedFileInfo.Name);
            }
        } catch (err) {
            console.error('Failed to save:', err);
            this.showNotification('Failed to save file: ' + (err.message || err), 'error');
        }
    }
    
    async saveAs() {
        const tab = this.tabs.find(t => t.id === this.activeTabId);
        if (!tab || !tab.textarea) return;
        
        const content = tab.textarea.value;
        
        if (!SaveFileAs) {
            this.showNotification('Save As not available', 'error');
            return;
        }
        
        try {
            const fileInfo = await SaveFileAs(tab.fileInfo.Name, content, tab.fileInfo.LineEnding);
            if (fileInfo) {
                // Normalize fileInfo from backend (lowercase) to frontend (PascalCase)
                const normalizedFileInfo = {
                    Path: fileInfo.path || fileInfo.Path || '',
                    Name: fileInfo.name || fileInfo.Name || 'Untitled',
                    Encoding: fileInfo.encoding || fileInfo.Encoding || 'UTF-8',
                    LineEnding: fileInfo.lineEnding || fileInfo.LineEnding || 'CRLF',
                    IsDirty: false,
                    IsNewFile: fileInfo.isNewFile || fileInfo.IsNewFile || false
                };
                tab.fileInfo = normalizedFileInfo;
                
                const nameEl = tab.element.querySelector('.tab-name');
                const dirtyEl = tab.element.querySelector('.tab-dirty');
                if (nameEl) nameEl.textContent = normalizedFileInfo.Name;
                if (dirtyEl) dirtyEl.style.display = 'none';
                
                this.updateStatusBar();
                this.showNotification('File saved: ' + normalizedFileInfo.Name, 'success');
            }
        } catch (err) {
            console.error('Failed to save:', err);
            this.showNotification('Failed to save file: ' + (err.message || err), 'error');
        }
    }
    
    // ============================================
    // UI Setup
    // ============================================
    
    setupMenus() {
        console.log('Setting up menus...');
        
        // Menu bar clicks
        document.querySelectorAll('.menu-item').forEach(item => {
            item.addEventListener('click', (e) => {
                e.stopPropagation();
                const menuName = item.dataset.menu;
                this.toggleMenu(menuName);
            });
        });
        
        // Menu option clicks
        document.querySelectorAll('.dropdown').forEach(dropdown => {
            dropdown.addEventListener('click', (e) => {
                e.stopPropagation();
                const option = e.target.closest('.menu-option');
                if (option) {
                    const action = option.dataset.action;
                    console.log('Menu action:', action);
                    this.handleMenuAction(action);
                    this.hideAllMenus();
                }
            });
        });
        
        // Context menu
        document.querySelectorAll('.context-item').forEach(item => {
            item.addEventListener('click', (e) => {
                e.stopPropagation();
                const action = item.dataset.action;
                this.handleContextAction(action);
                this.elements.contextMenu.classList.add('hidden');
            });
        });
        
        // Close menus on outside click
        document.addEventListener('click', (e) => {
            if (!e.target.closest('#menu-bar') && !e.target.closest('.dropdown')) {
                this.hideAllMenus();
            }
            this.elements.contextMenu.classList.add('hidden');
        });
    }
    
    toggleMenu(menuName) {
        const dropdown = document.getElementById(`menu-${menuName}`);
        const menuItem = document.querySelector(`[data-menu="${menuName}"]`);
        
        if (!dropdown || !menuItem) {
            console.warn('Menu not found:', menuName);
            return;
        }
        
        const isVisible = dropdown.classList.contains('show');
        this.hideAllMenus();
        
        if (!isVisible) {
            dropdown.classList.add('show');
            const rect = menuItem.getBoundingClientRect();
            dropdown.style.left = `${rect.left}px`;
            dropdown.style.top = `${rect.bottom}px`;
        }
    }
    
    hideAllMenus() {
        document.querySelectorAll('.dropdown').forEach(d => d.classList.remove('show'));
    }
    
    handleMenuAction(action) {
        console.log('Handling menu action:', action);
        
        switch (action) {
            case 'new': this.newFile(); break;
            case 'open': this.openFile(); break;
            case 'save': this.saveTab(); break;
            case 'save-as': this.saveAs(); break;
            case 'exit': this.exitApp(); break;
            
            case 'undo': document.execCommand('undo'); break;
            case 'redo': document.execCommand('redo'); break;
            case 'cut': document.execCommand('cut'); break;
            case 'copy': document.execCommand('copy'); break;
            case 'paste': document.execCommand('paste'); break;
            case 'find': this.showFindReplaceDialog(false); break;
            case 'replace': this.showFindReplaceDialog(true); break;
            case 'go-to': this.showGoToDialog(); break;
            
            case 'zoom-in': this.zoomIn(); break;
            case 'zoom-out': this.zoomOut(); break;
            case 'reset-zoom': this.resetZoom(); break;
            case 'dark-mode': document.body.classList.toggle('light-theme'); break;
            case 'fullscreen': this.toggleFullscreen(); break;
            
            case 'ai-sidebar': this.toggleAISidebar(); break;
            
            case 'about': this.showAboutDialog(); break;
            case 'shortcuts': this.showKeyboardShortcutsDialog(); break;
            case 'word-wrap': this.toggleWordWrap(); break;
            case 'line-numbers': this.toggleLineNumbers(); break;
            case 'minimap': this.toggleMinimap(); break;
        }
    }
    
    handleContextAction(action) {
        switch (action) {
            case 'cut': document.execCommand('cut'); break;
            case 'copy': document.execCommand('copy'); break;
            case 'paste': document.execCommand('paste'); break;
        }
    }
    
    // ============================================
    // View Controls
    // ============================================
    
    zoomIn() {
        this.zoomLevel = Math.min(this.zoomLevel + 10, 200);
        this.applyZoom();
    }
    
    zoomOut() {
        this.zoomLevel = Math.max(this.zoomLevel - 10, 50);
        this.applyZoom();
    }
    
    resetZoom() {
        this.zoomLevel = 100;
        this.applyZoom();
    }
    
    applyZoom() {
        const tab = this.tabs.find(t => t.id === this.activeTabId);
        if (tab && tab.textarea) {
            tab.textarea.style.fontSize = `${this.zoomLevel}%`;
        }
        this.elements.statusZoom.textContent = `${this.zoomLevel}%`;
    }
    
    toggleFullscreen() {
        if (!document.fullscreenElement) {
            document.documentElement.requestFullscreen();
        } else {
            document.exitFullscreen();
        }
    }
    
    toggleAISidebar() {
        const isHidden = this.elements.aiSidebar.classList.contains('hidden');
        this.elements.aiSidebar.classList.toggle('hidden');
        
        // Initialize AI when opening sidebar
        if (isHidden) {
            if (!this.ollamaStatus.checked) {
                this.ollamaStatus.checked = true;
                this.checkOllamaInstallation();
                this.loadInstalledModels();
            } else {
                // Re-check server status to update UI
                this.checkServerStatus();
            }
        }
    }
    
    toggleLineEnding() {
        const tab = this.tabs.find(t => t.id === this.activeTabId);
        if (!tab) return;
        
        tab.fileInfo.LineEnding = tab.fileInfo.LineEnding === 'CRLF' ? 'LF' : 'CRLF';
        tab.fileInfo.IsDirty = true;
        const dirtyEl = tab.element.querySelector('.tab-dirty');
        if (dirtyEl) dirtyEl.style.display = 'inline';
        this.updateStatusBar();
    }
    
    toggleWordWrap() {
        const tab = this.getActiveTab();
        if (!tab || !tab.textarea) return;
        
        const currentWrap = tab.textarea.wrap;
        tab.textarea.wrap = currentWrap === 'off' ? 'soft' : 'off';
        this.showNotification(`Word wrap ${tab.textarea.wrap === 'off' ? 'disabled' : 'enabled'}`);
    }
    
    toggleLineNumbers() {
        this.showNotification('Line numbers feature coming soon');
    }
    
    toggleMinimap() {
        this.showNotification('Minimap feature coming soon');
    }
    
    // ============================================
    // Dialogs
    // ============================================
    
    showFindReplaceDialog(showReplace = false) {
        this.elements.dialogOverlay.classList.remove('hidden');
        document.getElementById('dialog-find-replace').classList.remove('hidden');
        document.getElementById('find-replace-input').focus();
        
        const replaceSection = document.getElementById('replace-section');
        if (replaceSection) {
            replaceSection.style.display = showReplace ? 'block' : 'none';
        }
        
        this.findReplaceMode = showReplace ? 'replace' : 'find';
    }
    
    findNext() {
        const tab = this.getActiveTab();
        if (!tab || !tab.textarea) return;
        
        const searchTerm = document.getElementById('find-replace-input').value;
        if (!searchTerm) return;
        
        const text = tab.textarea.value;
        const startPos = tab.textarea.selectionEnd;
        
        const index = text.indexOf(searchTerm, startPos);
        if (index !== -1) {
            tab.textarea.focus();
            tab.textarea.setSelectionRange(index, index + searchTerm.length);
            this.updateCursorPosition(tab);
        } else {
            // Wrap around
            const wrapIndex = text.indexOf(searchTerm, 0);
            if (wrapIndex !== -1 && wrapIndex < startPos) {
                tab.textarea.focus();
                tab.textarea.setSelectionRange(wrapIndex, wrapIndex + searchTerm.length);
                this.updateCursorPosition(tab);
            }
        }
    }
    
    findPrevious() {
        const tab = this.getActiveTab();
        if (!tab || !tab.textarea) return;
        
        const searchTerm = document.getElementById('find-replace-input').value;
        if (!searchTerm) return;
        
        const text = tab.textarea.value;
        const startPos = tab.textarea.selectionStart - 1;
        
        const index = text.lastIndexOf(searchTerm, startPos);
        if (index !== -1) {
            tab.textarea.focus();
            tab.textarea.setSelectionRange(index, index + searchTerm.length);
            this.updateCursorPosition(tab);
        } else {
            // Wrap around to end
            const wrapIndex = text.lastIndexOf(searchTerm);
            if (wrapIndex !== -1 && wrapIndex > startPos) {
                tab.textarea.focus();
                tab.textarea.setSelectionRange(wrapIndex, wrapIndex + searchTerm.length);
                this.updateCursorPosition(tab);
            }
        }
    }
    
    replaceOne() {
        const tab = this.getActiveTab();
        if (!tab || !tab.textarea) return;
        
        const searchTerm = document.getElementById('find-replace-input').value;
        const replaceTerm = document.getElementById('replace-input').value;
        
        if (!searchTerm) return;
        
        const text = tab.textarea.value;
        const startPos = tab.textarea.selectionStart;
        
        // Check if current selection matches
        const currentSelection = text.substring(tab.textarea.selectionStart, tab.textarea.selectionEnd);
        if (currentSelection === searchTerm) {
            // Replace current selection
            const newText = text.substring(0, tab.textarea.selectionStart) + replaceTerm + text.substring(tab.textarea.selectionEnd);
            tab.textarea.value = newText;
            tab.textarea.setSelectionRange(startPos, startPos + replaceTerm.length);
            this.onEditorChange(tab);
        } else {
            // Find next and select
            this.findNext();
        }
    }
    
    replaceAll() {
        const tab = this.getActiveTab();
        if (!tab || !tab.textarea) return;
        
        const searchTerm = document.getElementById('find-replace-input').value;
        const replaceTerm = document.getElementById('replace-input').value;
        
        if (!searchTerm) return;
        
        const text = tab.textarea.value;
        const regex = new RegExp(this.escapeRegExp(searchTerm), 'g');
        const newText = text.replace(regex, replaceTerm);
        
        if (text !== newText) {
            tab.textarea.value = newText;
            this.onEditorChange(tab);
            this.showNotification(`Replaced ${(text.match(regex) || []).length} occurrence(s)`, 'success');
        }
    }
    
    escapeRegExp(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    }
    
    showGoToDialog() {
        this.elements.dialogOverlay.classList.remove('hidden');
        document.getElementById('dialog-goto').classList.remove('hidden');
        document.getElementById('goto-input').focus();
    }
    
    hideDialogs() {
        this.elements.dialogOverlay.classList.add('hidden');
        document.querySelectorAll('.dialog').forEach(d => d.classList.add('hidden'));
    }
    
    showNotification(message, type = 'info') {
        // Remove existing notification
        const existing = document.getElementById('notification-toast');
        if (existing) existing.remove();
        
        const notification = document.createElement('div');
        notification.id = 'notification-toast';
        notification.className = `notification ${type}`;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        // Animate in
        setTimeout(() => notification.classList.add('show'), 10);
        
        // Remove after delay
        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => notification.remove(), 300);
        }, 3000);
    }
    
    showConfirmDialog(title, message, onConfirm, onCancel = null) {
        this.elements.dialogOverlay.classList.remove('hidden');
        
        let dialog = document.getElementById('dialog-confirm');
        if (!dialog) {
            dialog = document.createElement('div');
            dialog.id = 'dialog-confirm';
            dialog.className = 'dialog dialog-confirm hidden';
            dialog.innerHTML = `
                <div class="dialog-header" id="confirm-title">Confirm</div>
                <div class="dialog-body" id="confirm-message">Are you sure?</div>
                <div class="dialog-footer">
                    <button id="confirm-yes">Yes</button>
                    <button id="confirm-no">No</button>
                </div>
            `;
            this.elements.dialogOverlay.appendChild(dialog);
        }
        
        document.getElementById('confirm-title').textContent = title;
        document.getElementById('confirm-message').textContent = message;
        
        dialog.classList.remove('hidden');
        
        const yesBtn = document.getElementById('confirm-yes');
        const noBtn = document.getElementById('confirm-no');
        
        // Remove old listeners
        const newYesBtn = yesBtn.cloneNode(true);
        const newNoBtn = noBtn.cloneNode(true);
        yesBtn.parentNode.replaceChild(newYesBtn, yesBtn);
        noBtn.parentNode.replaceChild(newNoBtn, noBtn);
        
        newYesBtn.addEventListener('click', () => {
            this.hideDialogs();
            if (onConfirm) onConfirm();
        });
        
        newNoBtn.addEventListener('click', () => {
            this.hideDialogs();
            if (onCancel) onCancel();
        });
    }
    
    showSaveConfirmDialog(title, message, onSave, onDontSave, onCancel = null) {
        this.elements.dialogOverlay.classList.remove('hidden');
        
        let dialog = document.getElementById('dialog-save-confirm');
        if (!dialog) {
            dialog = document.createElement('div');
            dialog.id = 'dialog-save-confirm';
            dialog.className = 'dialog dialog-confirm hidden';
            dialog.style.width = '400px';
            dialog.innerHTML = `
                <div class="dialog-header" id="save-confirm-title" style="display: flex; align-items: center; gap: 10px;">
                    <span style="font-size: 20px;">💾</span>
                    <span>Unsaved Changes</span>
                </div>
                <div class="dialog-body" id="save-confirm-message" style="padding: 20px; text-align: center;">
                    Save changes?
                </div>
                <div class="dialog-footer" style="justify-content: center; gap: 10px; padding: 15px 20px;">
                    <button id="save-confirm-save" class="btn-primary" style="padding: 8px 20px; background: var(--accent-color); color: white; border: none; border-radius: 4px; cursor: pointer;">Save</button>
                    <button id="save-confirm-dontsave" style="padding: 8px 20px; background: var(--bg-tertiary); color: var(--text-primary); border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer;">Don't Save</button>
                    <button id="save-confirm-cancel" style="padding: 8px 20px; background: transparent; color: var(--text-secondary); border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer;">Cancel</button>
                </div>
            `;
            this.elements.dialogOverlay.appendChild(dialog);
        }
        
        document.getElementById('save-confirm-title').innerHTML = `<span style="font-size: 20px;">💾</span><span>${title}</span>`;
        document.getElementById('save-confirm-message').textContent = message;
        
        dialog.classList.remove('hidden');
        
        const saveBtn = document.getElementById('save-confirm-save');
        const dontSaveBtn = document.getElementById('save-confirm-dontsave');
        const cancelBtn = document.getElementById('save-confirm-cancel');
        
        // Remove old listeners
        const newSaveBtn = saveBtn.cloneNode(true);
        const newDontSaveBtn = dontSaveBtn.cloneNode(true);
        const newCancelBtn = cancelBtn.cloneNode(true);
        saveBtn.parentNode.replaceChild(newSaveBtn, saveBtn);
        dontSaveBtn.parentNode.replaceChild(newDontSaveBtn, dontSaveBtn);
        cancelBtn.parentNode.replaceChild(newCancelBtn, cancelBtn);
        
        newSaveBtn.addEventListener('click', () => {
            this.hideDialogs();
            if (onSave) onSave();
        });
        
        newDontSaveBtn.addEventListener('click', () => {
            this.hideDialogs();
            if (onDontSave) onDontSave();
        });
        
        newCancelBtn.addEventListener('click', () => {
            this.hideDialogs();
            if (onCancel) onCancel();
        });
    }
    
    showStyledConfirmDialog(title, message, confirmText, cancelText, onConfirm, onCancel = null) {
        this.elements.dialogOverlay.classList.remove('hidden');
        
        const dialogId = 'dialog-styled-confirm';
        let dialog = document.getElementById(dialogId);
        if (!dialog) {
            dialog = document.createElement('div');
            dialog.id = dialogId;
            dialog.className = 'dialog dialog-confirm hidden';
            dialog.style.width = '380px';
            dialog.innerHTML = `
                <div class="dialog-header" id="styled-confirm-title" style="display: flex; align-items: center; gap: 10px; border-bottom: 1px solid var(--border-color);">
                    <span id="styled-confirm-icon" style="font-size: 20px;">⚠️</span>
                    <span id="styled-confirm-header-text">Confirm</span>
                </div>
                <div class="dialog-body" id="styled-confirm-message" style="padding: 25px 20px; text-align: center; font-size: 14px; line-height: 1.5;">
                    Are you sure?
                </div>
                <div class="dialog-footer" style="justify-content: center; gap: 12px; padding: 15px 20px;">
                    <button id="styled-confirm-yes" class="btn-danger" style="padding: 8px 20px; background: var(--error-color); color: white; border: none; border-radius: 4px; cursor: pointer; font-weight: 500;">Delete</button>
                    <button id="styled-confirm-no" style="padding: 8px 20px; background: var(--bg-tertiary); color: var(--text-primary); border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer;">Cancel</button>
                </div>
            `;
            this.elements.dialogOverlay.appendChild(dialog);
        }
        
        // Update content
        const isDelete = confirmText.toLowerCase().includes('delete');
        const icon = isDelete ? '🗑️' : '⚠️';
        const btnClass = isDelete ? 'var(--error-color)' : 'var(--accent-color)';
        
        document.getElementById('styled-confirm-icon').textContent = icon;
        document.getElementById('styled-confirm-header-text').textContent = title;
        document.getElementById('styled-confirm-message').textContent = message;
        
        const yesBtn = document.getElementById('styled-confirm-yes');
        const noBtn = document.getElementById('styled-confirm-no');
        
        yesBtn.textContent = confirmText;
        yesBtn.style.background = btnClass;
        noBtn.textContent = cancelText;
        
        dialog.classList.remove('hidden');
        
        // Remove old listeners
        const newYesBtn = yesBtn.cloneNode(true);
        const newNoBtn = noBtn.cloneNode(true);
        yesBtn.parentNode.replaceChild(newYesBtn, yesBtn);
        noBtn.parentNode.replaceChild(newNoBtn, noBtn);
        
        newYesBtn.addEventListener('click', () => {
            this.hideDialogs();
            if (onConfirm) onConfirm();
        });
        
        newNoBtn.addEventListener('click', () => {
            this.hideDialogs();
            if (onCancel) onCancel();
        });
    }
    
    goToLine() {
        const lineNum = parseInt(document.getElementById('goto-input').value);
        if (lineNum > 0) {
            const tab = this.tabs.find(t => t.id === this.activeTabId);
            if (tab && tab.textarea) {
                const lines = tab.textarea.value.split('\n');
                let pos = 0;
                for (let i = 0; i < Math.min(lineNum - 1, lines.length); i++) {
                    pos += lines[i].length + 1;
                }
                tab.textarea.focus();
                tab.textarea.setSelectionRange(pos, pos);
            }
        }
        this.hideDialogs();
    }
    
    showContextMenu(event) {
        event.preventDefault();
        const menu = this.elements.contextMenu;
        menu.style.left = `${event.pageX}px`;
        menu.style.top = `${event.pageY}px`;
        menu.classList.remove('hidden');
    }
    
    // ============================================
    // Event Listeners
    // ============================================
    
    setupEventListeners() {
        if (this.elements.newTabBtn) {
            this.elements.newTabBtn.addEventListener('click', () => this.newFile());
        }
        
        const closeAiBtn = document.getElementById('close-ai-sidebar');
        if (closeAiBtn) {
            closeAiBtn.addEventListener('click', () => {
                this.elements.aiSidebar.classList.add('hidden');
            });
        }
        
        if (this.elements.statusLineEnding) {
            this.elements.statusLineEnding.addEventListener('click', () => this.toggleLineEnding());
        }
        
        if (this.elements.statusZoom) {
            this.elements.statusZoom.addEventListener('click', () => this.resetZoom());
        }
        
        // Find/Replace dialog buttons
        const findClose = document.getElementById('find-close');
        if (findClose) {
            findClose.addEventListener('click', () => this.hideDialogs());
        }
        
        const findNextBtn = document.getElementById('find-next-btn');
        if (findNextBtn) {
            findNextBtn.addEventListener('click', () => this.findNext());
        }
        
        const findPrevBtn = document.getElementById('find-prev-btn');
        if (findPrevBtn) {
            findPrevBtn.addEventListener('click', () => this.findPrevious());
        }
        
        const replaceOneBtn = document.getElementById('replace-one-btn');
        if (replaceOneBtn) {
            replaceOneBtn.addEventListener('click', () => this.replaceOne());
        }
        
        const replaceAllBtn = document.getElementById('replace-all-btn');
        if (replaceAllBtn) {
            replaceAllBtn.addEventListener('click', () => this.replaceAll());
        }
        
        const gotoCancel = document.getElementById('goto-cancel');
        if (gotoCancel) {
            gotoCancel.addEventListener('click', () => this.hideDialogs());
        }
        
        const gotoOk = document.getElementById('goto-ok');
        if (gotoOk) {
            gotoOk.addEventListener('click', () => this.goToLine());
        }
    }
    
    setupKeyboardShortcuts() {
        // Define default shortcuts
        this.shortcuts = {
            'new': { key: 'n', ctrl: true, shift: false, action: () => this.newFile() },
            'open': { key: 'o', ctrl: true, shift: false, action: () => this.openFile() },
            'save': { key: 's', ctrl: true, shift: false, action: () => this.saveTab() },
            'save-as': { key: 's', ctrl: true, shift: true, action: () => this.saveAs() },
            'find': { key: 'f', ctrl: true, shift: false, action: () => this.showFindReplaceDialog(false) },
            'replace': { key: 'h', ctrl: true, shift: false, action: () => this.showFindReplaceDialog(true) },
            'go-to': { key: 'g', ctrl: true, shift: false, action: () => this.showGoToDialog() },
            'zoom-in': { key: '+', ctrl: true, shift: true, action: () => this.zoomIn() },
            'zoom-out': { key: '-', ctrl: true, shift: false, action: () => this.zoomOut() },
            'reset-zoom': { key: '0', ctrl: true, shift: false, action: () => this.resetZoom() },
            'word-wrap': { key: 'z', ctrl: true, shift: false, action: () => this.toggleWordWrap() },
            'line-numbers': { key: 'l', ctrl: true, shift: true, action: () => this.toggleLineNumbers() },
            'minimap': { key: 'm', ctrl: true, shift: true, action: () => this.toggleMinimap() },
            'fullscreen': { key: 'F11', ctrl: false, shift: false, action: () => this.toggleFullscreen() },
            'ai-sidebar': { key: 'a', ctrl: true, shift: true, action: () => this.toggleAISidebar() }
        };
        
        document.addEventListener('keydown', (e) => {
            // Handle F11 separately
            if (e.key === 'F11') {
                e.preventDefault();
                this.toggleFullscreen();
                return;
            }
            
            // Handle zoom shortcuts specially (Ctrl++, Ctrl+-, Ctrl+0)
            if ((e.ctrlKey || e.metaKey) && !e.altKey) {
                if (e.key === '+' || e.key === '=') {
                    e.preventDefault();
                    this.zoomIn();
                    return;
                }
                if (e.key === '-') {
                    e.preventDefault();
                    this.zoomOut();
                    return;
                }
                if (e.key === '0') {
                    e.preventDefault();
                    this.resetZoom();
                    return;
                }
            }
            
            // Check for matching shortcut
            for (const [name, shortcut] of Object.entries(this.shortcuts)) {
                const keyMatch = e.key.toLowerCase() === shortcut.key.toLowerCase();
                const ctrlMatch = e.ctrlKey === shortcut.ctrl || e.metaKey === shortcut.ctrl;
                const shiftMatch = e.shiftKey === shortcut.shift;
                
                if (keyMatch && ctrlMatch && shiftMatch) {
                    // Don't trigger shortcuts in inputs except for file operations
                    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
                        if (name !== 'save' && name !== 'save-as' && name !== 'open' && name !== 'new') {
                            continue;
                        }
                    }
                    
                    e.preventDefault();
                    shortcut.action();
                    return;
                }
            }
        });
    }
    
    // ============================================
    // Status Bar
    // ============================================
    
    updateStatusBar() {
        const tab = this.tabs.find(t => t.id === this.activeTabId);
        if (!tab) return;
        
        if (this.elements.statusFile) {
            this.elements.statusFile.textContent = tab.fileInfo.Name;
        }
        if (this.elements.statusEncoding) {
            this.elements.statusEncoding.textContent = tab.fileInfo.Encoding;
        }
        if (this.elements.statusLineEnding) {
            this.elements.statusLineEnding.textContent = tab.fileInfo.LineEnding;
        }
        if (this.elements.statusZoom) {
            this.elements.statusZoom.textContent = `${this.zoomLevel}%`;
        }
        
        this.updateCursorPosition(tab);
    }
    
    getActiveTab() {
        return this.tabs.find(t => t.id === this.activeTabId);
    }
    
    // ============================================
    // App Control
    // ============================================
    
    showAboutDialog() {
        this.elements.dialogOverlay.classList.remove('hidden');
        
        let dialog = document.getElementById('dialog-about');
        if (!dialog) {
            dialog = document.createElement('div');
            dialog.id = 'dialog-about';
            dialog.className = 'dialog hidden';
            dialog.style.width = '450px';
            dialog.innerHTML = `
                <div class="dialog-header">About Akashic</div>
                <div class="dialog-body" style="text-align: center; padding: 30px;">
                    <img src="./logo.png" alt="Akashic Logo" style="width: 80px; height: 80px; margin-bottom: 20px; border-radius: 8px;">
                    <h2 style="margin-bottom: 10px; color: var(--accent-color);">Akashic Editor</h2>
                    <p style="font-size: 14px; color: var(--text-secondary); margin-bottom: 20px;">v0.1.0</p>
                    <p style="font-size: 13px; line-height: 1.6; margin-bottom: 20px;">
                        Akashic is an AI-enhanced text editor built with Wails and modern web technologies.
                        Designed for developers who want a lightweight yet powerful editing experience.
                    </p>
                </div>
                <div class="dialog-footer" style="justify-content: center;">
                    <button id="about-close" style="padding: 8px 24px;">Close</button>
                </div>
            `;
            this.elements.dialogOverlay.appendChild(dialog);
            
            document.getElementById('about-close').addEventListener('click', () => {
                this.hideDialogs();
            });
        }
        
        dialog.classList.remove('hidden');
    }
    
    showKeyboardShortcutsDialog() {
        this.elements.dialogOverlay.classList.remove('hidden');
        
        let dialog = document.getElementById('dialog-shortcuts');
        if (!dialog) {
            dialog = document.createElement('div');
            dialog.id = 'dialog-shortcuts';
            dialog.className = 'dialog hidden';
            dialog.style.width = '500px';
            dialog.style.maxHeight = '80vh';
            dialog.innerHTML = `
                <div class="dialog-header">Keyboard Shortcuts</div>
                <div class="dialog-body" style="padding: 0; max-height: 400px; overflow-y: auto;">
                    <table style="width: 100%; border-collapse: collapse; font-size: 13px;">
                        <thead style="position: sticky; top: 0; background: var(--bg-secondary);">
                            <tr>
                                <th style="text-align: left; padding: 10px 15px; border-bottom: 1px solid var(--border-color);">Action</th>
                                <th style="text-align: left; padding: 10px 15px; border-bottom: 1px solid var(--border-color);">Shortcut</th>
                            </tr>
                        </thead>
                        <tbody id="shortcuts-list">
                        </tbody>
                    </table>
                </div>
                <div class="dialog-footer">
                    <button id="shortcuts-close">Close</button>
                </div>
            `;
            this.elements.dialogOverlay.appendChild(dialog);
            
            document.getElementById('shortcuts-close').addEventListener('click', () => {
                this.hideDialogs();
            });
        }
        
        this.renderShortcutsList();
        dialog.classList.remove('hidden');
    }
    
    renderShortcutsList() {
        const tbody = document.getElementById('shortcuts-list');
        if (!tbody) return;
        
        tbody.innerHTML = '';
        
        const shortcutNames = {
            'new': 'New File',
            'open': 'Open File',
            'save': 'Save',
            'save-as': 'Save As',
            'find': 'Find',
            'replace': 'Replace',
            'go-to': 'Go To Line',
            'zoom-in': 'Zoom In',
            'zoom-out': 'Zoom Out',
            'reset-zoom': 'Reset Zoom',
            'word-wrap': 'Toggle Word Wrap',
            'line-numbers': 'Toggle Line Numbers',
            'minimap': 'Toggle Minimap',
            'fullscreen': 'Fullscreen',
            'ai-sidebar': 'Toggle AI Sidebar'
        };
        
        for (const [name, shortcut] of Object.entries(this.shortcuts)) {
            const row = document.createElement('tr');
            row.style.borderBottom = '1px solid var(--border-color)';
            
            const displayShortcut = [];
            if (shortcut.ctrl) displayShortcut.push('Ctrl');
            if (shortcut.shift) displayShortcut.push('Shift');
            displayShortcut.push(shortcut.key.toUpperCase());
            
            row.innerHTML = `
                <td style="padding: 8px 15px; color: var(--text-primary);">${shortcutNames[name] || name}</td>
                <td style="padding: 8px 15px;">
                    <kbd style="background: var(--bg-tertiary); padding: 2px 6px; border-radius: 3px; font-family: monospace; font-size: 12px;">
                        ${displayShortcut.join('+')}
                    </kbd>
                </td>
            `;
            tbody.appendChild(row);
        }
    }
    
    setupWindowCloseHandler() {
        // Handle window close event
        window.addEventListener('beforeunload', (e) => {
            const dirtyTabs = this.tabs.filter(t => t.fileInfo.IsDirty);
            if (dirtyTabs.length > 0) {
                // This will show the dialog but browser may not wait for async
                e.preventDefault();
                e.returnValue = '';
                
                // Show our custom dialog
                this.showSaveConfirmDialog(
                    'Unsaved Changes',
                    `You have ${dirtyTabs.length} unsaved file(s). Save before closing?`,
                    async () => {
                        await Promise.all(dirtyTabs.map(tab => this.saveTab(tab)));
                        window.removeEventListener('beforeunload', this);
                        window.close();
                    },
                    () => {
                        window.removeEventListener('beforeunload', this);
                        window.close();
                    },
                    () => {
                        // Cancel - do nothing, stay on page
                    }
                );
            }
        });
    }
    
    exitApp() {
        const dirtyTabs = this.tabs.filter(t => t.fileInfo.IsDirty);
        if (dirtyTabs.length > 0) {
            this.showSaveConfirmDialog(
                'Unsaved Changes',
                `You have ${dirtyTabs.length} unsaved file(s). Save before exiting?`,
                () => {
                    // Save all and exit
                    Promise.all(dirtyTabs.map(tab => this.saveTab(tab))).then(() => {
                        this.quitApplication();
                    });
                },
                () => {
                    // Don't save, just exit
                    this.quitApplication();
                },
                () => {
                    // Cancel - do nothing
                }
            );
        } else {
            this.quitApplication();
        }
    }
    
    quitApplication() {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.Quit) {
            window.go.main.App.Quit();
        } else {
            window.close();
        }
    }

    // ============================================
    // Ollama AI Integration
    // ============================================
    
    async checkOllamaInstallation() {
        try {
            const status = await CheckOllamaInstalled();
            this.ollamaStatus = status;

            const statusIndicator = document.getElementById('ai-status-indicator');
            const statusText = document.getElementById('ai-status-text');
            const setupPanel = document.getElementById('ai-setup-panel');

            if (status.installed) {
                statusIndicator.className = 'status-dot online';
                statusText.textContent = `Ollama ${status.version}`;
                setupPanel.classList.add('hidden');

                // Load models and check server
                await this.loadInstalledModels();
                await this.checkServerStatus();
            } else {
                statusIndicator.className = 'status-dot offline';
                statusText.textContent = 'Ollama not installed';
                setupPanel.classList.remove('hidden');
            }
        } catch (err) {
            console.error('Failed to check Ollama:', err);
            this.showNotification('Failed to check Ollama installation', 'error');
        }
    }
    
    async loadInstalledModels() {
        try {
            const models = await GetInstalledModels();
            this.installedModels = models;
            
            const select = document.getElementById('ai-model');
            
            select.innerHTML = '';
            
            if (models.length === 0) {
                select.innerHTML = '<option value="">No models installed</option>';
            } else {
                models.forEach(model => {
                    const option = document.createElement('option');
                    option.value = model.name;
                    option.textContent = `${model.name} (${model.size})`;
                    select.appendChild(option);
                });
                
                // Select first model by default
                this.selectedModel = models[0].name;
                select.value = this.selectedModel;
            }
            
            select.addEventListener('change', (e) => {
                this.selectedModel = e.target.value;
            });
        } catch (err) {
            console.error('Failed to load models:', err);
        }
    }
    
    async checkServerStatus() {
        try {
            // Try to get models - if it works, server is running
            await GetInstalledModels();
            this.serverRunning = true;
            this.updateServerStatusUI(true);
        } catch (err) {
            this.serverRunning = false;
            this.updateServerStatusUI(false);
        }
    }
    
    updateServerStatusUI(running) {
        const statusIndicator = document.getElementById('ai-status-indicator');
        const statusText = document.getElementById('ai-status-text');
        const startBtn = document.getElementById('start-ollama-server');
        
        if (!statusIndicator || !statusText) return;
        
        if (running) {
            statusIndicator.className = 'status-dot online';
            statusText.textContent = 'Server running';
            statusText.style.color = 'var(--success-color)';
            if (startBtn) {
                startBtn.classList.add('hidden');
            }
        } else {
            statusIndicator.className = 'status-dot offline';
            statusText.textContent = 'Server not running';
            statusText.style.color = 'var(--error-color)';
            if (startBtn) {
                startBtn.classList.remove('hidden');
            }
        }
    }
    
    async startOllamaServer() {
        try {
            this.showNotification('Starting Ollama server...', 'info');
            await StartOllamaServer();
            this.showNotification('Ollama server started!', 'success');
            this.serverRunning = true;
            this.updateServerStatusUI(true);
            document.getElementById('ai-chat-container').classList.remove('hidden');
        } catch (err) {
            console.error('Failed to start server:', err);
            this.showNotification(err.message || 'Failed to start server', 'error');
        }
    }
    
    async generateWithAI() {
        if (!this.selectedModel) {
            this.showNotification('Please select a model first', 'warning');
            return;
        }
        
        // Create new chat if none exists
        if (!this.currentChatId) {
            await this.createNewChat();
        }
        
        const promptInput = document.getElementById('ai-prompt');
        const prompt = promptInput.value;
        if (!prompt.trim()) {
            this.showNotification('Please enter a prompt', 'warning');
            return;
        }
        
        const messagesDiv = document.getElementById('ai-messages');
        
        // Remove welcome message if present
        const welcome = messagesDiv.querySelector('.ai-welcome');
        if (welcome) welcome.remove();
        
        // Add user message
        const userMsgDiv = document.createElement('div');
        userMsgDiv.className = 'ai-message user';
        userMsgDiv.innerHTML = `<div class="ai-message-content">${this.escapeHtml(prompt)}</div>`;
        messagesDiv.appendChild(userMsgDiv);
        
        // Add AI response placeholder
        const aiMsgDiv = document.createElement('div');
        aiMsgDiv.className = 'ai-message assistant';
        aiMsgDiv.innerHTML = '<div class="ai-message-content" style="color: var(--text-secondary); font-style: italic;">Generating...</div>';
        messagesDiv.appendChild(aiMsgDiv);
        
        // Clear input
        promptInput.value = '';
        
        // Scroll to bottom
        messagesDiv.scrollTop = messagesDiv.scrollHeight;
        
        try {
            // Save user message to database
            await AddMessage(this.currentChatId, 'user', prompt);
            
            // Get context from previous messages
            const context = await GetChatContext(this.currentChatId, 10);
            
            // Generate with context
            const fullPrompt = context + '\n\nUser: ' + prompt + '\n\nAssistant:';
            const response = await GenerateWithOllama(this.selectedModel, fullPrompt);
            
            // Update AI message
            aiMsgDiv.innerHTML = `<div class="ai-message-content" style="white-space: pre-wrap;">${this.escapeHtml(response)}</div>`;
            this.lastAIResponse = response;
            
            // Save AI response to database
            await AddMessage(this.currentChatId, 'assistant', response);
            
            // Update chat title if it's the first message
            const chat = this.chats.find(c => c.id === this.currentChatId);
            if (chat && chat.title === 'New Chat') {
                await RenameChatFromFirstMessage(this.currentChatId);
                // Reload chat list to show updated title
                await this.loadChatHistory();
                this.updateChatInfo();
            }
        } catch (err) {
            console.error('Generation failed:', err);
            aiMsgDiv.innerHTML = `<div class="ai-message-content" style="color: var(--error-color);">Error: ${err.message}</div>`;
        }
        
        // Scroll to bottom again
        messagesDiv.scrollTop = messagesDiv.scrollHeight;
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
    
    insertAIResponse() {
        if (!this.lastAIResponse) {
            this.showNotification('No AI response to insert', 'warning');
            return;
        }
        
        const tab = this.getActiveTab();
        if (!tab || !tab.textarea) return;
        
        const start = tab.textarea.selectionStart;
        const end = tab.textarea.selectionEnd;
        const text = tab.textarea.value;
        
        tab.textarea.value = text.substring(0, start) + this.lastAIResponse + text.substring(end);
        tab.textarea.setSelectionRange(start + this.lastAIResponse.length, start + this.lastAIResponse.length);
        this.onEditorChange(tab);
        this.showNotification('AI response inserted', 'success');
    }
    
    replaceWithAIResponse() {
        if (!this.lastAIResponse) {
            this.showNotification('No AI response to insert', 'warning');
            return;
        }
        
        const tab = this.getActiveTab();
        if (!tab || !tab.textarea) return;
        
        const start = tab.textarea.selectionStart;
        const end = tab.textarea.selectionEnd;
        const text = tab.textarea.value;
        
        // Replace selected text or insert at cursor
        tab.textarea.value = text.substring(0, start) + this.lastAIResponse + text.substring(end);
        tab.textarea.setSelectionRange(start, start + this.lastAIResponse.length);
        this.onEditorChange(tab);
        this.showNotification('Text replaced with AI response', 'success');
    }
    
    setupAIEventListeners() {
        // Check Ollama button
        const checkAgainBtn = document.getElementById('check-ollama-again');
        if (checkAgainBtn) {
            checkAgainBtn.addEventListener('click', () => this.checkOllamaInstallation());
        }
        
        // Start server button
        const startServerBtn = document.getElementById('start-ollama-server');
        if (startServerBtn) {
            startServerBtn.addEventListener('click', () => this.startOllamaServer());
        }
        
        // AI action buttons
        const generateBtn = document.getElementById('ai-send');
        if (generateBtn) {
            generateBtn.addEventListener('click', () => this.generateWithAI());
        }
        
        const insertBtn = document.getElementById('ai-insert');
        if (insertBtn) {
            insertBtn.addEventListener('click', () => this.insertAIResponse());
        }
        
        const replaceBtn = document.getElementById('ai-replace');
        if (replaceBtn) {
            replaceBtn.addEventListener('click', () => this.replaceWithAIResponse());
        }
        
        // AI suggestions
        document.querySelectorAll('.ai-suggestion').forEach(btn => {
            btn.addEventListener('click', () => {
                const prompt = btn.dataset.prompt;
                document.getElementById('ai-prompt').value = prompt;
                this.updateSendButtonState();
                this.generateWithAI();
            });
        });
        
        // Enter key in prompt and input validation
        const promptInput = document.getElementById('ai-prompt');
        if (promptInput) {
            promptInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    this.generateWithAI();
                }
            });
            
            // Enable/disable send button based on input
            promptInput.addEventListener('input', () => {
                this.updateSendButtonState();
            });
        }
        
        // Model select change
        const modelSelect = document.getElementById('ai-model');
        if (modelSelect) {
            modelSelect.addEventListener('change', () => {
                this.updateSendButtonState();
            });
        }
        
        // Initial state
        this.updateSendButtonState();
    }
    
    updateSendButtonState() {
        const sendBtn = document.getElementById('ai-send');
        const promptInput = document.getElementById('ai-prompt');
        const modelSelect = document.getElementById('ai-model');
        
        if (!sendBtn || !promptInput || !modelSelect) return;
        
        const hasText = promptInput.value.trim().length > 0;
        const hasModel = modelSelect.value && modelSelect.value !== '';
        
        sendBtn.disabled = !(hasText && hasModel);
    }
}

// Initialize
window.akashic = new AkashicEditor();
