/**
 * NeuralEditor - A project-native, high-performance code editor for Akashic.
 * Designed with the "Neural Midnight" aesthetic in mind.
 */
export class NeuralEditor {
    constructor(container, options = {}) {
        this.container = container;
        this.options = {
            fontSize: 14,
            lineHeight: 24, // Matches CSS
            tabSize: 4,
            language: 'javascript',
            ...options
        };

        this.lines = [''];
        this.cursor = { line: 0, ch: 0 };
        this.isFocused = false;
        
        // Syntax highlighting rules
        this.languages = {
            javascript: [
                { type: 'comment', regex: /\/\/.*/ },
                { type: 'comment', regex: /\/\*[\s\S]*?\*\// },
                { type: 'string', regex: /"(?:\\.|[^\\"])*"|'(?:\\.|[^\\'])*'|`(?:\\.|[^\\`])*`/ },
                { type: 'keyword', regex: /\b(break|case|catch|class|const|continue|debugger|default|delete|do|else|export|extends|finally|for|function|if|import|in|instanceof|new|return|super|switch|this|throw|try|typeof|var|void|while|with|yield|let|static|enum|await|async)\b/ },
                { type: 'number', regex: /\b\d+(\.\d+)?\b/ },
                { type: 'operator', regex: /[+\-*/%=&|^<>!~?:]+/ },
                { type: 'function', regex: /\b[a-zA-Z_]\w*(?=\s*\()/ }
            ],
            html: [
                { type: 'comment', regex: /<!--[\s\S]*?-->/ },
                { type: 'tag', regex: /<\/?[a-zA-Z][^>]*>/ },
                { type: 'attr', regex: /\b[a-zA-Z_-]+(?=\s*=)/ },
                { type: 'string', regex: /"(?:\\.|[^\\"])*"|'(?:\\.|[^\\'])*'/ }
            ],
            css: [
                { type: 'comment', regex: /\/\*[\s\S]*?\*\// },
                { type: 'keyword', regex: /@[a-zA-Z-]+/ },
                { type: 'type', regex: /\b(color|background|margin|padding|width|height|display|flex|position|top|left|right|bottom|font-family|font-size|font-weight|line-height|border|border-radius|box-shadow|backdrop-filter|animation|transition|transform)\b(?=\s*:)/ },
                { type: 'number', regex: /#-?[0-9a-fA-F.]+|[0-9.]+(px|em|rem|%|vh|vw|s|ms|deg)?/ },
                { type: 'string', regex: /"(?:\\.|[^\\"])*"|'(?:\\.|[^\\'])*'/ }
            ]
        };

        this.init();
    }

    init() {
        this.container.innerHTML = '';
        this.container.classList.add('neural-editor-container');
        
        this.wrapper = document.createElement('div');
        this.wrapper.className = 'neural-editor-wrapper';
        
        this.gutter = document.createElement('div');
        this.gutter.className = 'neural-gutter';
        
        this.contentArea = document.createElement('div');
        this.contentArea.className = 'neural-content';
        
        this.input = document.createElement('textarea');
        this.input.className = 'neural-hidden-input';
        this.input.autocapitalize = 'off';
        this.input.autocomplete = 'off';
        this.input.spellcheck = false;
        
        this.wrapper.appendChild(this.gutter);
        this.wrapper.appendChild(this.contentArea);
        this.container.appendChild(this.wrapper);
        this.container.appendChild(this.input);
        
        this.setupEvents();
        this.render();
    }

    setupEvents() {
        this.contentArea.addEventListener('click', (e) => {
            this.input.focus();
            this.updateCursorFromClick(e);
        });

        this.input.addEventListener('focus', () => {
            this.isFocused = true;
            this.wrapper.classList.add('focused');
            this.render();
        });

        this.input.addEventListener('blur', () => {
            this.isFocused = false;
            this.wrapper.classList.remove('focused');
            this.render();
        });

        this.input.addEventListener('input', (e) => {
            if (e.inputType === 'insertLineBreak' || (e.inputType === 'insertText' && e.data === null)) {
                // Handled by keydown Enter usually, but some browsers differ
                return;
            }
            if (e.data) {
                this.insertText(e.data);
                this.input.value = '';
            }
        });

        this.input.addEventListener('keydown', (e) => {
            this.handleKeydown(e);
        });

        this.contentArea.addEventListener('scroll', () => {
            this.gutter.scrollTop = this.contentArea.scrollTop;
        });
    }

    updateCursorFromClick(e) {
        // Simple line detection
        const rect = this.contentArea.getBoundingClientRect();
        const y = e.clientY - rect.top + this.contentArea.scrollTop;
        const lineIdx = Math.max(0, Math.min(this.lines.length - 1, Math.floor((y - 20) / this.options.lineHeight)));
        
        this.cursor.line = lineIdx;
        this.cursor.ch = this.lines[this.cursor.line].length; // Move to end for now
        this.render();
    }

    insertText(text) {
        const line = this.lines[this.cursor.line];
        this.lines[this.cursor.line] = line.slice(0, this.cursor.ch) + text + line.slice(this.cursor.ch);
        this.cursor.ch += text.length;
        this.render();
        this.triggerChange();
    }

    handleKeydown(e) {
        const { key, ctrlKey, shiftKey } = e;
        
        if (key === 'Enter') {
            e.preventDefault();
            const line = this.lines[this.cursor.line];
            const indent = line.match(/^\s*/)[0];
            
            const before = line.slice(0, this.cursor.ch);
            const after = line.slice(this.cursor.ch);
            
            this.lines[this.cursor.line] = before;
            this.lines.splice(this.cursor.line + 1, 0, indent + after);
            
            this.cursor.line++;
            this.cursor.ch = indent.length;
            this.render();
            this.triggerChange();
        } 
        else if (key === 'Backspace') {
            if (this.cursor.ch > 0) {
                const line = this.lines[this.cursor.line];
                this.lines[this.cursor.line] = line.slice(0, this.cursor.ch - 1) + line.slice(this.cursor.ch);
                this.cursor.ch--;
            } else if (this.cursor.line > 0) {
                const currentLine = this.lines[this.cursor.line];
                const prevLine = this.lines[this.cursor.line - 1];
                this.cursor.ch = prevLine.length;
                this.lines[this.cursor.line - 1] = prevLine + currentLine;
                this.lines.splice(this.cursor.line, 1);
                this.cursor.line--;
            }
            this.render();
            this.triggerChange();
        }
        else if (key === 'Delete') {
            const line = this.lines[this.cursor.line];
            if (this.cursor.ch < line.length) {
                this.lines[this.cursor.line] = line.slice(0, this.cursor.ch) + line.slice(this.cursor.ch + 1);
            } else if (this.cursor.line < this.lines.length - 1) {
                const nextLine = this.lines[this.cursor.line + 1];
                this.lines[this.cursor.line] += nextLine;
                this.lines.splice(this.cursor.line + 1, 1);
            }
            this.render();
            this.triggerChange();
        }
        else if (key === 'ArrowLeft') {
            if (this.cursor.ch > 0) this.cursor.ch--;
            else if (this.cursor.line > 0) {
                this.cursor.line--;
                this.cursor.ch = this.lines[this.cursor.line].length;
            }
            this.render();
        }
        else if (key['ArrowRight']) { // Typo fix in thought, keep correct in code
            if (this.cursor.ch < this.lines[this.cursor.line].length) this.cursor.ch++;
            else if (this.cursor.line < this.lines.length - 1) {
                this.cursor.line++;
                this.cursor.ch = 0;
            }
            this.render();
        }
        else if (key === 'ArrowRight') {
            if (this.cursor.ch < this.lines[this.cursor.line].length) this.cursor.ch++;
            else if (this.cursor.line < this.lines.length - 1) {
                this.cursor.line++;
                this.cursor.ch = 0;
            }
            this.render();
        }
        else if (key === 'ArrowUp') {
            if (this.cursor.line > 0) {
                this.cursor.line--;
                this.cursor.ch = Math.min(this.cursor.ch, this.lines[this.cursor.line].length);
            }
            this.render();
        }
        else if (key === 'ArrowDown') {
            if (this.cursor.line < this.lines.length - 1) {
                this.cursor.line++;
                this.cursor.ch = Math.min(this.cursor.ch, this.lines[this.cursor.line].length);
            }
            this.render();
        }
        else if (key === 'Tab') {
            e.preventDefault();
            this.insertText(' '.repeat(this.options.tabSize));
        }
        else if (ctrlKey && key === 'v') {
            // Paste is tricky with direct state, but browser handles it in textarea
            // We just need to catch the 'input' event 
        }
        else if (ctrlKey && key === 's') {
            e.preventDefault();
            if (this.onSave) this.onSave();
        }
    }

    setContent(content) {
        this.lines = content ? content.split(/\r?\n/) : [''];
        this.render();
    }

    getContent() {
        return this.lines.join('\n');
    }

    highlight(text) {
        const rules = this.languages[this.options.language] || [];
        let html = this.escapeHtml(text);
        
        if (rules.length === 0) return html;

        // Simple tag-based highlighting
        let tokens = [];
        rules.forEach(rule => {
            let match;
            const regex = new RegExp(rule.regex, 'g');
            while ((match = regex.exec(text)) !== null) {
                tokens.push({
                    start: match.index,
                    end: match.index + match[0].length,
                    type: rule.type,
                    text: match[0]
                });
            }
        });

        // Sort and filter tokens
        tokens.sort((a, b) => a.start - b.start || b.end - a.end);
        let lastEnd = 0;
        let result = '';
        
        tokens.forEach(token => {
            if (token.start >= lastEnd) {
                result += this.escapeHtml(text.slice(lastEnd, token.start));
                result += `<span class="token-${token.type}">${this.escapeHtml(token.text)}</span>`;
                lastEnd = token.end;
            }
        });
        result += this.escapeHtml(text.slice(lastEnd));

        return result || ' ';
    }

    render() {
        // Gutter
        this.gutter.innerHTML = '';
        this.lines.forEach((_, i) => {
            const num = document.createElement('div');
            num.className = `gutter-line ${i === this.cursor.line ? 'active' : ''}`;
            num.textContent = i + 1;
            this.gutter.appendChild(num);
        });

        // Content
        this.contentArea.innerHTML = '';
        this.lines.forEach((line, i) => {
            const lineDiv = document.createElement('div');
            lineDiv.className = `neural-line ${i === this.cursor.line ? 'active' : ''}`;
            
            const highlighted = this.highlight(line);
            
            if (i === this.cursor.line && this.isFocused) {
                // Render custom cursor
                const before = line.slice(0, this.cursor.ch);
                const char = line.slice(this.cursor.ch, this.cursor.ch + 1) || ' ';
                const after = line.slice(this.cursor.ch + 1);
                
                // We use a simplified highlighting for the active line to avoid complex tag nesting during cursor split
                lineDiv.innerHTML = `${this.highlight(before)}<span class="neural-cursor">${this.escapeHtml(char)}</span>${this.highlight(after)}`;
            } else {
                lineDiv.innerHTML = highlighted || ' ';
            }
            
            this.contentArea.appendChild(lineDiv);
        });

        // Scroll management
        const activeLine = this.contentArea.children[this.cursor.line];
        if (activeLine && this.isFocused) {
            const rect = activeLine.getBoundingClientRect();
            const parentRect = this.contentArea.getBoundingClientRect();
            if (rect.bottom > parentRect.bottom || rect.top < parentRect.top) {
                activeLine.scrollIntoView({ block: 'nearest' });
            }
        }
    }

    triggerChange() {
        if (this.onChange) this.onChange(this.getContent());
    }

    escapeHtml(text) {
        if (!text) return '';
        return text.replace(/[&<>"']/g, function(m) {
            return {
                '&': '&amp;',
                '<': '&lt;',
                '>': '&gt;',
                '"': '&quot;',
                "'": '&#039;'
            }[m];
        });
    }
}
