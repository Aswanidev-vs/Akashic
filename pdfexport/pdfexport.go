// Package pdfexport provides a custom PDF export implementation from scratch
// This is a pure Go implementation that doesn't rely on external PDF libraries
package pdfexport

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// Exporter handles PDF export operations
type Exporter struct{}

// NewExporter creates a new PDF exporter instance
func NewExporter() *Exporter {
	return &Exporter{}
}

// Export exports content as a professionally formatted PDF
func (e *Exporter) Export(content string, filePath string) error {
	// Create PDF document
	doc := newPDFDocument()

	// Set up page dimensions (A4 in points: 210mm x 297mm)
	const (
		pageWidthMM   = 210.0
		pageHeightMM  = 297.0
		marginMM      = 20.0
		fontSize      = 11.0
		lineHeightMM  = 5.5
		paraSpacingMM = 3.0
	)

	margin := marginMM
	contentWidth := pageWidthMM - (margin * 2)
	pageHeight := pageHeightMM

	// Process content by paragraphs
	paragraphs := e.splitParagraphs(content)

	// Track current page and position
	var page *pdfPage
	y := 0.0

	// Helper to check/create new page
	ensurePage := func() {
		if page == nil || y > pageHeight-margin-lineHeightMM {
			page = doc.addPage()
			y = margin + 10 // Start with some top padding
		}
	}

	for _, para := range paragraphs {
		ensurePage()

		// Handle empty paragraphs
		if strings.TrimSpace(para) == "" {
			y += paraSpacingMM
			continue
		}

		// Calculate indentation from leading whitespace
		trimmedPara := strings.TrimLeftFunc(para, unicode.IsSpace)
		leadingSpaces := len(para) - len(trimmedPara)
		indent := float64(leadingSpaces) * 2.5 // 2.5mm per space

		// Render paragraph with word wrapping
		lines := e.wrapText(trimmedPara, contentWidth-indent, fontSize)

		for _, line := range lines {
			ensurePage()

			// Add text to page (convert mm to points: 1mm = 2.83465pt)
			xPt := (margin + indent) * 2.83465
			yPt := (pageHeight - y) * 2.83465 // Flip Y coordinate

			page.addText(line, xPt, yPt, fontSize)
			y += lineHeightMM
		}

		// Add paragraph spacing
		y += paraSpacingMM * 0.5
	}

	// Write PDF to file
	return doc.write(filePath)
}

// splitParagraphs splits content into paragraphs preserving empty lines
func (e *Exporter) splitParagraphs(content string) []string {
	lines := strings.Split(content, "\n")
	var paragraphs []string
	var currentPara strings.Builder

	for i, line := range lines {
		// Empty line indicates paragraph break
		if strings.TrimSpace(line) == "" {
			if currentPara.Len() > 0 {
				paragraphs = append(paragraphs, currentPara.String())
				currentPara.Reset()
			}
			paragraphs = append(paragraphs, "") // Empty paragraph marker
			continue
		}

		// Add line to current paragraph
		if currentPara.Len() > 0 {
			currentPara.WriteString(" ")
		}
		currentPara.WriteString(line)

		// Check for special line endings
		trimmed := strings.TrimSpace(line)
		isSpecialLine := strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "* ") ||
			strings.HasPrefix(trimmed, "• ") ||
			(len(trimmed) > 0 && trimmed[len(trimmed)-1] == ':')

		// If next line is empty or special, end paragraph
		if i < len(lines)-1 {
			nextLine := lines[i+1]
			if strings.TrimSpace(nextLine) == "" || isSpecialLine {
				if currentPara.Len() > 0 {
					paragraphs = append(paragraphs, currentPara.String())
					currentPara.Reset()
				}
			}
		}
	}

	// Don't forget last paragraph
	if currentPara.Len() > 0 {
		paragraphs = append(paragraphs, currentPara.String())
	}

	return paragraphs
}

// wrapText wraps text into lines that fit within maxWidth
func (e *Exporter) wrapText(text string, maxWidthMM float64, fontSize float64) []string {
	words := e.splitWords(text)
	if len(words) == 0 {
		return nil
	}

	// Convert max width from mm to approximate character count
	// At 11pt font, average char width is about 0.5mm
	maxChars := int(maxWidthMM / 0.5)

	var lines []string
	var currentLine strings.Builder
	currentChars := 0

	for _, word := range words {
		wordLen := len(word)

		// Check if word fits on current line
		needsSpace := currentLine.Len() > 0
		additionalChars := wordLen
		if needsSpace {
			additionalChars++ // Space before word
		}

		if currentChars+additionalChars > maxChars && currentLine.Len() > 0 {
			// Line is full, start new line
			lines = append(lines, strings.TrimSpace(currentLine.String()))
			currentLine.Reset()
			currentLine.WriteString(word)
			currentChars = wordLen
		} else {
			// Add word to current line
			if needsSpace {
				currentLine.WriteString(" ")
				currentChars++
			}
			currentLine.WriteString(word)
			currentChars += wordLen
		}
	}

	// Add last line
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

// PDF Document structures
type pdfDocument struct {
	pages []*pdfPage
}

type pdfPage struct {
	texts []pdfText
}

type pdfText struct {
	text     string
	x, y     float64 // in PDF points
	fontSize float64
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

func (p *pdfPage) addText(text string, x, y, fontSize float64) {
	p.texts = append(p.texts, pdfText{
		text:     text,
		x:        x,
		y:        y,
		fontSize: fontSize,
	})
}

func (d *pdfDocument) write(filePath string) error {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	// Track object offsets for xref table
	var offsets []int
	objectNum := 0

	// Helper to write object and track offset
	writeObject := func(content string) {
		offsets = append(offsets, buf.Len())
		objectNum++
		writer.WriteString(strconv.Itoa(objectNum) + " 0 obj\n")
		writer.WriteString(content)
		writer.WriteString("endobj\n")
	}

	// PDF Header
	writer.WriteString("%PDF-1.4\n")
	writer.WriteString("%âãÏÓ\n") // Binary marker

	// Object 1: Catalog
	writeObject("<<\n/Type /Catalog\n/Pages 2 0 R\n>>\n")

	// Object 2: Pages
	pagesKids := make([]string, len(d.pages))
	for i := range d.pages {
		pagesKids[i] = fmt.Sprintf("%d 0 R", 3+i*2)
	}
	writeObject(fmt.Sprintf("<<\n/Type /Pages\n/Kids [%s]\n/Count %d\n>>\n",
		strings.Join(pagesKids, " "), len(d.pages)))

	// Page objects and content objects
	for i, page := range d.pages {
		contentObjNum := 4 + i*2

		// Build content stream
		content := page.buildContentStream()
		compressed := compressStream(content)

		// Page object
		writeObject(fmt.Sprintf("<<\n/Type /Page\n/Parent 2 0 R\n"+
			"/MediaBox [0 0 595.28 841.89]\n"+
			"/Contents %d 0 R\n"+
			"/Resources <<\n/Font <<\n/F1 <<\n/Type /Font\n/Subtype /Type1\n/BaseFont /Helvetica\n>>\n>>\n>>\n>>\n",
			contentObjNum))

		// Content stream object
		writeObject(fmt.Sprintf("<<\n/Length %d\n/Filter /FlateDecode\n>>\nstream\n%s\n",
			len(compressed), string(compressed)))
	}

	writer.Flush()

	// Cross-reference table
	xrefOffset := buf.Len()
	writer.WriteString("xref\n")
	writer.WriteString(fmt.Sprintf("0 %d\n", objectNum+1))
	writer.WriteString("0000000000 65535 f \n")

	for _, offset := range offsets {
		writer.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}

	// Trailer
	writer.WriteString("trailer\n")
	writer.WriteString(fmt.Sprintf("<<\n/Size %d\n/Root 1 0 R\n>>\n", objectNum+1))
	writer.WriteString("startxref\n")
	writer.WriteString(strconv.Itoa(xrefOffset) + "\n")
	writer.WriteString("%%EOF\n")

	writer.Flush()

	// Write to file
	return os.WriteFile(filePath, buf.Bytes(), 0644)
}

func (p *pdfPage) buildContentStream() []byte {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	// Begin content stream
	writer.WriteString("BT\n")        // Begin text
	writer.WriteString("/F1 11 Tf\n") // Font F1, size 11

	for _, text := range p.texts {
		// Position and show text
		writer.WriteString(fmt.Sprintf("%.2f %.2f Td\n", text.x, text.y))
		writer.WriteString("(" + escapePDFString(text.text) + ") Tj\n")
	}

	writer.WriteString("ET\n") // End text
	writer.Flush()

	return buf.Bytes()
}

func escapePDFString(s string) string {
	// Escape special characters in PDF strings
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
			buf.WriteString("\\t")
		default:
			// Only include printable ASCII
			if r >= 32 && r <= 126 {
				buf.WriteRune(r)
			} else {
				// Skip non-ASCII characters for basic PDF
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
