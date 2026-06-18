package pdfexport

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Renderer handles HTML to PDF conversion using headless Chrome
type Renderer struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRenderer creates a new PDF renderer
func NewRenderer() *Renderer {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	return &Renderer{
		ctx:    ctx,
		cancel: cancel,
	}
}

// IsChromeAvailable checks if Chrome is available for PDF generation
func (r *Renderer) IsChromeAvailable() bool {
	chromePaths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	}

	for _, path := range chromePaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	if _, err := exec.Command("which", "chrome").Output(); err == nil {
		return true
	}
	if _, err := exec.Command("which", "chromium").Output(); err == nil {
		return true
	}

	return false
}

// RenderHTMLToPDF converts HTML content to PDF file using Chrome
func (r *Renderer) RenderHTMLToPDF(htmlContent string, outputPath string) error {
	defer r.cancel()

	if !r.IsChromeAvailable() {
		return fmt.Errorf("Chrome is not available. Please install Google Chrome for best PDF quality")
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1200, 800),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(r.ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	tmpDir := os.TempDir()
	htmlFile := filepath.Join(tmpDir, "akashic_export_"+time.Now().Format("20060102_150405")+".html")

	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to create temp HTML file: %w", err)
	}
	defer os.Remove(htmlFile)

	err := chromedp.Run(taskCtx,
		chromedp.Navigate("file:///"+filepath.ToSlash(htmlFile)),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
	)

	if err != nil {
		return fmt.Errorf("failed to load page: %w", err)
	}

	var pdfData []byte
	err = chromedp.Run(taskCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfData, _, err = page.PrintToPDF().Do(ctx)
			return err
		}),
	)

	if err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}

	if err := os.WriteFile(outputPath, pdfData, 0644); err != nil {
		return fmt.Errorf("failed to write PDF file: %w", err)
	}

	return nil
}

// PDFOptions contains options for Chrome PDF generation
type PDFOptions struct {
	Landscape         bool
	Scale             float64
	PaperWidth        float64
	PaperHeight       float64
	MarginTop         float64
	MarginBottom      float64
	MarginLeft        float64
	MarginRight       float64
	PrintBackground   bool
	PreferCSSPageSize bool
}

// DefaultPDFOptions returns default options for A4 paper
func DefaultPDFOptions() PDFOptions {
	return PDFOptions{
		Landscape:         false,
		Scale:             1.0,
		PaperWidth:        8.5,
		PaperHeight:       11.0,
		MarginTop:         1.0,
		MarginBottom:      1.0,
		MarginLeft:        1.0,
		MarginRight:       1.0,
		PrintBackground:   true,
		PreferCSSPageSize: false,
	}
}

// ConvertHTMLToPDF converts HTML string to PDF using fallback
func ConvertHTMLToPDF(htmlContent string, outputPath string) error {
	fullHTML := "<!DOCTYPE html><html><head><meta charset=\"UTF-8\"><style>body{font-family:Arial,sans-serif;margin:40px;line-height:1.6}h1{font-size:24px;color:#333}h2{font-size:20px;color:#333}h3{font-size:16px;color:#333}p{margin:10px 0}code{background:#f4f4f4;padding:2px 6px;font-family:monospace}pre{background:#f4f4f4;padding:10px;overflow-x:auto}blockquote{border-left:4px solid #ddd;margin:10px 0;padding-left:10px;color:#666}table{border-collapse:collapse;width:100%;margin:10px 0}th,td{border:1px solid #ddd;padding:8px;text-align:left}th{background:#333;color:#fff}a{color:#06c}</style></head><body>" + htmlContent + "</body></html>"

	htmlPath := outputPath + ".html"
	if err := os.WriteFile(htmlPath, []byte(fullHTML), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	return fmt.Errorf("PDF generation requires Chrome. HTML saved to: %s", htmlPath)
}

// HTMLToPDFConverter provides unified interface for PDF conversion
type HTMLToPDFConverter struct {
	useChrome bool
}

// NewHTMLToPDFConverter creates a new converter
func NewHTMLToPDFConverter() *HTMLToPDFConverter {
	r := NewRenderer()
	available := r.IsChromeAvailable()
	return &HTMLToPDFConverter{
		useChrome: available,
	}
}

// IsChromeAvailable returns whether Chrome is available
func (c *HTMLToPDFConverter) IsChromeAvailable() bool {
	return c.useChrome
}

// Convert converts HTML to PDF using the best available method
func (c *HTMLToPDFConverter) Convert(htmlContent, outputPath string) error {
	if c.useChrome {
		r := NewRenderer()
		return r.RenderHTMLToPDF(htmlContent, outputPath)
	}
	return ConvertHTMLToPDF(htmlContent, outputPath)
}

// GetBestConverter returns the best available PDF converter
func GetBestConverter() *HTMLToPDFConverter {
	return NewHTMLToPDFConverter()
}

// RenderMarkdownToPDF converts markdown-formatted text to PDF
func RenderMarkdownToPDF(markdownContent, outputPath string) error {
	htmlContent := ConvertMarkdownToHTML(markdownContent)
	converter := GetBestConverter()
	return converter.Convert(htmlContent, outputPath)
}

// ConvertMarkdownToHTML converts markdown to HTML
func ConvertMarkdownToHTML(markdown string) string {
	var buf bytes.Buffer
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	inCodeBlock := false
	inList := false

	// Code block markers - using rune concatenation to avoid backtick issues
	codeBlockStart := string(rune(96)) + string(rune(96)) + string(rune(96))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if len(trimmed) >= 3 && strings.HasPrefix(trimmed, codeBlockStart) {
			if !inCodeBlock {
				inCodeBlock = true
				buf.WriteString("<pre><code>")
			} else {
				inCodeBlock = false
				buf.WriteString("</code></pre>")
			}
			continue
		}

		if inCodeBlock {
			buf.WriteString(escapeHTML(line))
			buf.WriteString("\n")
			continue
		}

		// Headers
		if len(trimmed) >= 2 && trimmed[0] == '#' {
			level := 0
			for _, c := range trimmed {
				if c == '#' {
					level++
				} else {
					break
				}
			}
			if level > 0 && level <= 6 && len(trimmed) > level && trimmed[level] == ' ' {
				content := escapeHTML(strings.TrimSpace(trimmed[level+1:]))
				buf.WriteString(fmt.Sprintf("<h%d>%s</h%d>", level, content, level))
				continue
			}
		}

		// Horizontal rule
		if len(trimmed) >= 3 && (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '_') {
			allSame := true
			for _, c := range trimmed {
				if c != rune(trimmed[0]) {
					allSame = false
					break
				}
			}
			if allSame {
				buf.WriteString("<hr>")
				continue
			}
		}

		// Bullet lists
		if len(trimmed) >= 2 && (trimmed[0] == '-' || trimmed[0] == '*') && trimmed[1] == ' ' {
			if !inList {
				inList = true
				buf.WriteString("<ul>")
			}
			content := escapeHTML(strings.TrimSpace(trimmed[2:]))
			buf.WriteString(fmt.Sprintf("<li>%s</li>", content))
			continue
		}

		// Numbered lists
		if len(trimmed) >= 2 {
			dotIdx := -1
			for i, c := range trimmed {
				if c >= '0' && c <= '9' {
					continue
				}
				if c == '.' || c == ')' {
					dotIdx = i
					break
				}
				break
			}
			if dotIdx > 0 && dotIdx < len(trimmed)-1 && trimmed[dotIdx+1] == ' ' {
				if !inList {
					inList = true
					buf.WriteString("<ol>")
				}
				content := escapeHTML(strings.TrimSpace(trimmed[dotIdx+2:]))
				buf.WriteString(fmt.Sprintf("<li>%s</li>", content))
				continue
			}
		}

		// Close list
		if inList && trimmed != "" {
			inList = false
			buf.WriteString("</ul>")
		}

		// Links
		if len(trimmed) >= 2 {
			linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)
			if match := linkRe.FindStringSubmatch(trimmed); match != nil {
				buf.WriteString(fmt.Sprintf("<p><a href=\"%s\">%s</a></p>", escapeHTML(match[2]), escapeHTML(match[1])))
				continue
			}
		}

		// Images
		if len(trimmed) >= 2 && trimmed[0] == '!' {
			imgRe := regexp.MustCompile(`!\[([^\]]*)\]\(([^\)]+)\)`)
			if match := imgRe.FindStringSubmatch(trimmed); match != nil {
				buf.WriteString(fmt.Sprintf("<p><img src=\"%s\" alt=\"%s\"></p>", escapeHTML(match[2]), escapeHTML(match[1])))
				continue
			}
		}

		// Regular paragraph
		if trimmed != "" {
			buf.WriteString(fmt.Sprintf("<p>%s</p>", processInline(trimmed)))
		}
	}

	if inList {
		buf.WriteString("</ul>")
	}

	htmlTemplate := "<!DOCTYPE html><html><head><meta charset=\"UTF-8\"><style>" +
		"body{font-family:Arial,Helvetica,sans-serif;margin:40px;line-height:1.6;color:#333}" +
		"h1{font-size:24px;color:#1a1a1a;margin-top:20px}" +
		"h2{font-size:20px;color:#333;margin-top:18px}" +
		"h3{font-size:16px;color:#444;margin-top:15px}" +
		"p{margin:10px 0}" +
		"code{background:#f4f4f4;padding:2px 6px;font-family:monospace;font-size:14px}" +
		"pre{background:#f4f4f4;padding:15px;overflow-x:auto;border-radius:4px}" +
		"pre code{padding:0;background:0}" +
		"blockquote{border-left:4px solid #ddd;margin:15px 0;padding:10px 15px;color:#666;background:#f9f9f9}" +
		"ul,ol{margin:10px 0;padding-left:30px}" +
		"li{margin:5px 0}" +
		"table{border-collapse:collapse;width:100%;margin:15px 0}" +
		"th,td{border:1px solid #ddd;padding:10px;text-align:left}" +
		"th{background:#333;color:#fff}" +
		"a{color:#06c;text-decoration:none}" +
		"a:hover{text-decoration:underline}" +
		"img{max-width:100%;height:auto}" +
		"hr{border:0;border-top:1px solid #ddd;margin:20px 0}" +
		"</style></head><body>" + buf.String() + "</body></html>"

	return htmlTemplate
}

func processInline(text string) string {
	text = escapeHTML(text)

	// Bold: **text** or __text__
	boldRe := regexp.MustCompile(`\*\*(.+?)\*\*`)
	text = boldRe.ReplaceAllString(text, "<strong>$1</strong>")
	boldRe2 := regexp.MustCompile(`__(.+?)__`)
	text = boldRe2.ReplaceAllString(text, "<strong>$1</strong>")

	// Italic: *text* or _text_
	italicRe := regexp.MustCompile(`\*(.+?)\*`)
	text = italicRe.ReplaceAllString(text, "<em>$1</em>")
	italicRe2 := regexp.MustCompile(`_(.+?)_`)
	text = italicRe2.ReplaceAllString(text, "<em>$1</em>")

	// Inline code using backtick - using rune to avoid issues
	backtick := string(rune(96))
	codeRe := regexp.MustCompile(backtick + `(.+?)` + backtick)
	text = codeRe.ReplaceAllString(text, "<code>$1</code>")

	return text
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// Close cleans up resources
func (r *Renderer) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}

// IsEdgeAvailable checks if Microsoft Edge is available
func (r *Renderer) IsEdgeAvailable() bool {
	edgePaths := []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"/usr/bin/microsoft-edge",
	}

	for _, path := range edgePaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// IsBraveAvailable checks if Brave Browser is available
func (r *Renderer) IsBraveAvailable() bool {
	bravePaths := []string{
		`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		"/usr/bin/brave-browser",
	}

	for _, path := range bravePaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// IsFirefoxAvailable checks if Firefox is available
func (r *Renderer) IsFirefoxAvailable() bool {
	firefoxPaths := []string{
		`C:\Program Files\Mozilla Firefox\firefox.exe`,
		"/Applications/Firefox.app/Contents/MacOS/firefox",
		"/usr/bin/firefox",
	}

	for _, path := range firefoxPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// GetBrowserPath returns the first available browser path
func (r *Renderer) GetBrowserPath() string {
	// Priority: Chrome -> Edge -> Brave -> Firefox
	searchPaths := []string{
		// Chrome
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		"/usr/bin/google-chrome",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		// Edge
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"/usr/bin/microsoft-edge",
		// Brave
		`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		"/usr/bin/brave-browser",
		// Firefox
		`C:\Program Files\Mozilla Firefox\firefox.exe`,
		"/Applications/Firefox.app/Contents/MacOS/firefox",
		"/usr/bin/firefox",
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
