// Package pdfexport provides a custom PDF export implementation from scratch
// This is a pure Go implementation that doesn't rely on external PDF libraries
package pdfexport

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Font constants
const (
	fontRegular = "/F1" // Helvetica
	fontBold    = "/F2" // Helvetica-Bold
)

// textStyle represents the visual style of a text element
type textStyle struct {
	fontName string
	fontSize float64
	indent   float64 // left indent in mm
}

var (
	// Predefined styles
	styleTitle     = textStyle{fontBold, 18, 0}
	styleH1        = textStyle{fontBold, 16, 0}
	styleH2        = textStyle{fontBold, 14, 0}
	styleH3        = textStyle{fontBold, 12, 0}
	styleBody      = textStyle{fontRegular, 10.5, 0}
	styleBullet    = textStyle{fontRegular, 10.5, 8}
	styleSubBullet = textStyle{fontRegular, 10.5, 14}
	styleNumbered  = textStyle{fontRegular, 10.5, 8}

	// Regex for numbered list items like "1.", "2.", "1)", etc.
	numberedListRe = regexp.MustCompile(`^\d+[\.\)]\s+`)
)

// Exporter handles PDF export operations
type Exporter struct{}

// NewExporter creates a new PDF exporter instance
func NewExporter() *Exporter {
	return &Exporter{}
}

// Export exports content as a professionally formatted PDF
func (e *Exporter) Export(content string, filePath string) error {
	doc := newPDFDocument()

	// A4 dimensions in mm
	const (
		pageWidthMM  = 210.0
		pageHeightMM = 297.0
		marginLeft   = 25.0
		marginRight  = 20.0
		marginTop    = 25.0
		marginBottom = 25.0
	)

	contentWidth := pageWidthMM - marginLeft - marginRight

	// Parse content into styled blocks
	blocks := e.parseContent(content)

	// Render blocks across pages
	var page *pdfPage
	y := 0.0

	newPage := func() {
		page = doc.addPage()
		y = marginTop
	}

	for _, block := range blocks {
		style := block.style

		spaceBefore := block.spaceBefore
		availableWidth := contentWidth - style.indent

		// Word-wrap the text
		lines := e.wrapText(block.text, availableWidth, style.fontSize)
		if len(lines) == 0 {
			// Empty block just adds spacing
			if page != nil {
				y += spaceBefore
			}
			continue
		}

		lineHeight := style.fontSize * 0.45 // mm per line

		// Total height this block needs
		blockHeight := spaceBefore + float64(len(lines))*lineHeight

		// Check if we need a new page
		if page == nil || y+blockHeight > pageHeightMM-marginBottom {
			newPage()
		} else {
			y += spaceBefore
		}

		for _, line := range lines {
			// Check page overflow mid-block
			if y+lineHeight > pageHeightMM-marginBottom {
				newPage()
			}

			xPt := (marginLeft + style.indent) * 2.83465
			yPt := (pageHeightMM - y) * 2.83465

			page.addText(line, xPt, yPt, style.fontSize, style.fontName)
			y += lineHeight
		}
	}

	return doc.write(filePath)
}

// contentBlock represents a parsed piece of content with its style
type contentBlock struct {
	text        string
	style       textStyle
	spaceBefore float64 // mm of space before this block
}

// parseContent converts raw text into styled content blocks
func (e *Exporter) parseContent(content string) []contentBlock {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	var blocks []contentBlock
	isFirstContent := true

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Empty line - add paragraph spacing
		if trimmed == "" {
			if !isFirstContent {
				blocks = append(blocks, contentBlock{
					text:        "",
					style:       styleBody,
					spaceBefore: 2.0,
				})
			}
			continue
		}

		// Auto-detect title: ONLY the very first non-empty line,
		// and only if it's short + followed by a blank line
		if isFirstContent {
			isFirstContent = false
			if len(trimmed) < 60 && i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
				blocks = append(blocks, contentBlock{
					text:        trimmed,
					style:       styleTitle,
					spaceBefore: 0,
				})
				continue
			}
		}
		isFirstContent = false

		// Detect explicit markdown headings only
		if strings.HasPrefix(trimmed, "### ") {
			blocks = append(blocks, contentBlock{
				text:        strings.TrimPrefix(trimmed, "### "),
				style:       styleH3,
				spaceBefore: 4.0,
			})
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			blocks = append(blocks, contentBlock{
				text:        strings.TrimPrefix(trimmed, "## "),
				style:       styleH2,
				spaceBefore: 5.0,
			})
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			blocks = append(blocks, contentBlock{
				text:        strings.TrimPrefix(trimmed, "# "),
				style:       styleH1,
				spaceBefore: 6.0,
			})
			continue
		}

		// Detect leading whitespace for sub-items
		leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))
		isSubItem := leadingSpaces >= 2

		// Detect bullet points (- or *)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			bulletText := trimmed[2:]
			st := styleBullet
			if isSubItem {
				st = styleSubBullet
			}
			blocks = append(blocks, contentBlock{
				text:        "-  " + bulletText,
				style:       st,
				spaceBefore: 1.5,
			})
			continue
		}

		// Detect numbered lists
		if loc := numberedListRe.FindStringIndex(trimmed); loc != nil {
			number := trimmed[:loc[1]]
			rest := trimmed[loc[1]:]
			st := styleNumbered
			if isSubItem {
				st.indent = 14
			}
			blocks = append(blocks, contentBlock{
				text:        number + rest,
				style:       st,
				spaceBefore: 1.5,
			})
			continue
		}

		// Regular body text
		blocks = append(blocks, contentBlock{
			text:        trimmed,
			style:       styleBody,
			spaceBefore: 1.5,
		})
	}

	return blocks
}

// wrapText wraps text into lines that fit within maxWidth (mm)
func (e *Exporter) wrapText(text string, maxWidthMM float64, fontSize float64) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	words := e.splitWords(text)
	if len(words) == 0 {
		return nil
	}

	// Approximate character width based on font size
	// Helvetica average glyph width ≈ 500/1000 em. 1pt = 0.3528mm.
	// So charWidth ≈ fontSize * 0.5 * 0.3528 = fontSize * 0.18 mm
	charWidth := fontSize * 0.18
	maxChars := int(maxWidthMM / charWidth)
	if maxChars < 20 {
		maxChars = 20
	}

	var lines []string
	var currentLine strings.Builder
	currentChars := 0

	for _, word := range words {
		wordLen := len(word)

		needsSpace := currentLine.Len() > 0
		additionalChars := wordLen
		if needsSpace {
			additionalChars++
		}

		if currentChars+additionalChars > maxChars && currentLine.Len() > 0 {
			lines = append(lines, strings.TrimSpace(currentLine.String()))
			currentLine.Reset()
			currentLine.WriteString(word)
			currentChars = wordLen
		} else {
			if needsSpace {
				currentLine.WriteString(" ")
				currentChars++
			}
			currentLine.WriteString(word)
			currentChars += wordLen
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, strings.TrimSpace(currentLine.String()))
	}

	return lines
}

// splitWords splits text into words
func (e *Exporter) splitWords(text string) []string {
	var words []string
	var currentWord strings.Builder

	for _, r := range text {
		if unicode.IsSpace(r) {
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		} else {
			currentWord.WriteRune(r)
		}
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// =============================================
// PDF Document structures
// =============================================

type pdfDocument struct {
	pages []*pdfPage
}

type pdfPage struct {
	texts []pdfText
}

type pdfText struct {
	text     string
	x, y     float64
	fontSize float64
	fontName string
}

func newPDFDocument() *pdfDocument {
	return &pdfDocument{
		pages: make([]*pdfPage, 0),
	}
}

func (d *pdfDocument) addPage() *pdfPage {
	page := &pdfPage{
		texts: make([]pdfText, 0),
	}
	d.pages = append(d.pages, page)
	return page
}

func (p *pdfPage) addText(text string, x, y, fontSize float64, fontName string) {
	p.texts = append(p.texts, pdfText{
		text:     text,
		x:        x,
		y:        y,
		fontSize: fontSize,
		fontName: fontName,
	})
}

func (d *pdfDocument) write(filePath string) error {
	var buf bytes.Buffer

	var offsets []int
	objectNum := 0

	writeObject := func(content string) {
		offsets = append(offsets, buf.Len())
		objectNum++
		buf.WriteString(strconv.Itoa(objectNum) + " 0 obj\n")
		buf.WriteString(content)
		buf.WriteString("endobj\n")
	}

	writeStreamObject := func(header string, streamData []byte) {
		offsets = append(offsets, buf.Len())
		objectNum++
		buf.WriteString(strconv.Itoa(objectNum) + " 0 obj\n")
		buf.WriteString(header)
		buf.WriteString("stream\r\n")
		buf.Write(streamData)
		buf.WriteString("\r\nendstream\n")
		buf.WriteString("endobj\n")
	}

	// PDF Header
	buf.WriteString("%PDF-1.4\n")
	buf.Write([]byte{'%', 0xE2, 0xE3, 0xCF, 0xD3, '\n'})

	// Object 1: Catalog
	writeObject("<<\n/Type /Catalog\n/Pages 2 0 R\n>>\n")

	// Object 2: Pages
	pagesKids := make([]string, len(d.pages))
	for i := range d.pages {
		pagesKids[i] = fmt.Sprintf("%d 0 R", 3+i*2)
	}
	writeObject(fmt.Sprintf("<<\n/Type /Pages\n/Kids [%s]\n/Count %d\n>>\n",
		strings.Join(pagesKids, " "), len(d.pages)))

	// Page + content stream objects
	for i, page := range d.pages {
		contentObjNum := 4 + i*2

		content := page.buildContentStream()
		compressed := compressStream(content)

		// Page object with both regular and bold fonts
		writeObject(fmt.Sprintf("<<\n/Type /Page\n/Parent 2 0 R\n"+
			"/MediaBox [0 0 595.28 841.89]\n"+
			"/Contents %d 0 R\n"+
			"/Resources <<\n"+
			"/Font <<\n"+
			"/F1 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\n"+
			"/F2 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>\n"+
			">>\n>>\n>>\n",
			contentObjNum))

		streamHeader := fmt.Sprintf("<<\n/Length %d\n/Filter /FlateDecode\n>>\n",
			len(compressed))
		writeStreamObject(streamHeader, compressed)
	}

	// Cross-reference table
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("0 %d\n", objectNum+1))
	buf.WriteString("0000000000 65535 f \n")

	for _, offset := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}

	// Trailer
	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<<\n/Size %d\n/Root 1 0 R\n>>\n", objectNum+1))
	buf.WriteString("startxref\n")
	buf.WriteString(strconv.Itoa(xrefOffset) + "\n")
	buf.WriteString("%%EOF\n")

	return os.WriteFile(filePath, buf.Bytes(), 0644)
}

func (p *pdfPage) buildContentStream() []byte {
	var buf bytes.Buffer

	buf.WriteString("BT\n")

	currentFont := ""
	currentSize := 0.0

	for _, text := range p.texts {
		// Only emit font change when needed
		if text.fontName != currentFont || text.fontSize != currentSize {
			buf.WriteString(fmt.Sprintf("%s %.1f Tf\n", text.fontName, text.fontSize))
			currentFont = text.fontName
			currentSize = text.fontSize
		}

		// Absolute positioning via text matrix
		buf.WriteString(fmt.Sprintf("1 0 0 1 %.2f %.2f Tm\n", text.x, text.y))
		buf.WriteString("(" + escapePDFString(text.text) + ") Tj\n")
	}

	buf.WriteString("ET\n")

	return buf.Bytes()
}

func escapePDFString(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		switch r {
		case '\\':
			buf.WriteString("\\\\")
		case '(':
			buf.WriteString("\\(")
		case ')':
			buf.WriteString("\\)")
		case '\r':
			buf.WriteString("\\r")
		case '\n':
			buf.WriteString("\\n")
		case '\t':
			buf.WriteString("    ") // Convert tabs to spaces
		default:
			if r >= 32 && r <= 126 {
				buf.WriteRune(r)
			}
		}
	}
	return buf.String()
}

func compressStream(data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}
