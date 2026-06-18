// Package pdfexport provides professional PDF export capabilities
// Supports both pure Go generation and Chrome-based rendering
package pdfexport

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// =============================================
// Content Types and Structures
// =============================================

// ElementType represents the type of PDF element
type ElementType int

const (
	ElementText ElementType = iota
	ElementHeading
	ElementImage
	ElementTable
	ElementLink
	ElementLine
	ElementPageBreak
	ElementBullet
	ElementNumbered
)

// TextAlignment for text formatting
type TextAlignment int

const (
	AlignLeft TextAlignment = iota
	AlignCenter
	AlignRight
	AlignJustify
)

// Color represents RGB color
type Color struct {
	R, G, B float64 // 0-1 range
}

// Common colors
var (
	ColorBlack     = Color{0, 0, 0}
	ColorWhite     = Color{1, 1, 1}
	ColorGray      = Color{0.5, 0.5, 0.5}
	ColorDarkGray  = Color{0.2, 0.2, 0.2}
	ColorLightGray = Color{0.9, 0.9, 0.9}
	ColorBlue      = Color{0.2, 0.4, 0.8}
	ColorRed       = Color{0.8, 0.2, 0.2}
	ColorGreen     = Color{0.2, 0.6, 0.2}
	ColorYellow    = Color{1, 1, 0}
	ColorLink      = Color{0.2, 0.4, 0.8}
)

// FontFamily available fonts
type FontFamily int

const (
	FontHelvetica FontFamily = iota
	FontHelveticaBold
	FontHelveticaItalic
	FontHelveticaBoldItalic
	FontTimes
	FontTimesBold
	FontTimesItalic
	FontTimesBoldItalic
	FontCourier
	FontCourierBold
	FontCourierItalic
	FontCourierBoldItalic
)

// FontInfo contains font metadata
type FontInfo struct {
	Family      FontFamily
	Name        string
	BaseName    string
	IsBold      bool
	IsItalic    bool
	IsMonospace bool
}

// Built-in fonts
var builtInFonts = map[FontFamily]FontInfo{
	FontHelvetica:           {FontHelvetica, "Helvetica", "Helvetica", false, false, false},
	FontHelveticaBold:       {FontHelveticaBold, "Helvetica-Bold", "Helvetica-Bold", true, false, false},
	FontHelveticaItalic:     {FontHelveticaItalic, "Helvetica-Oblique", "Helvetica-Oblique", false, true, false},
	FontHelveticaBoldItalic: {FontHelveticaBoldItalic, "Helvetica-BoldOblique", "Helvetica-BoldOblique", true, true, false},
	FontTimes:               {FontTimes, "Times-Roman", "Times-Roman", false, false, false},
	FontTimesBold:           {FontTimesBold, "Times-Bold", "Times-Bold", true, false, false},
	FontTimesItalic:         {FontTimesItalic, "Times-Italic", "Times-Italic", false, true, false},
	FontTimesBoldItalic:     {FontTimesBoldItalic, "Times-BoldItalic", "Times-BoldItalic", true, true, false},
	FontCourier:             {FontCourier, "Courier", "Courier", false, false, true},
	FontCourierBold:         {FontCourierBold, "Courier-Bold", "Courier-Bold", true, false, true},
	FontCourierItalic:       {FontCourierItalic, "Courier-Oblique", "Courier-Oblique", false, true, true},
	FontCourierBoldItalic:   {FontCourierBoldItalic, "Courier-BoldOblique", "Courier-BoldOblique", true, true, true},
}

// FontStyle defines text styling
type FontStyle struct {
	Family FontFamily
	Size   float64
	Color  Color
	Bold   bool
	Italic bool
	Align  TextAlignment
}

// Default styles
var (
	StyleTitle      = FontStyle{FontHelveticaBold, 24, ColorBlack, true, false, AlignCenter}
	StyleH1         = FontStyle{FontHelveticaBold, 20, ColorBlack, true, false, AlignLeft}
	StyleH2         = FontStyle{FontHelveticaBold, 16, ColorBlack, true, false, AlignLeft}
	StyleH3         = FontStyle{FontHelveticaBold, 14, ColorBlack, true, false, AlignLeft}
	StyleBody       = FontStyle{FontHelvetica, 11, ColorBlack, false, false, AlignLeft}
	StyleBodyBold   = FontStyle{FontHelveticaBold, 11, ColorBlack, true, false, AlignLeft}
	StyleBodyItalic = FontStyle{FontHelveticaItalic, 11, ColorBlack, false, true, AlignLeft}
	StyleBullet     = FontStyle{FontHelvetica, 11, ColorBlack, false, false, AlignLeft}
	StyleCode       = FontStyle{FontCourier, 10, ColorDarkGray, false, false, AlignLeft}
	StyleLink       = FontStyle{FontHelvetica, 10, ColorLink, false, false, AlignLeft}
	StylePageNumber = FontStyle{FontHelvetica, 9, ColorGray, false, false, AlignCenter}
	StyleHeader     = FontStyle{FontHelvetica, 10, ColorDarkGray, false, false, AlignLeft}
	StyleFooter     = FontStyle{FontHelvetica, 9, ColorGray, false, false, AlignCenter}
)

// PDFElement is a generic PDF content element
type PDFElement interface {
	GetType() ElementType
}

// TextElement represents a text block
type TextElement struct {
	Content     string
	Style       FontStyle
	Link        string // URL for hyperlinks
	Indent      float64
	SpaceBefore float64
	SpaceAfter  float64
}

func (t *TextElement) GetType() ElementType { return ElementText }

// HeadingElement represents a heading
type HeadingElement struct {
	Level   int // 1-6
	Content string
	Style   FontStyle
}

func (h *HeadingElement) GetType() ElementType { return ElementHeading }

// ImageElement represents an image
type ImageElement struct {
	Path     string
	WidthMM  float64
	HeightMM float64
	Align    TextAlignment
	Alt      string
}

func (i *ImageElement) GetType() ElementType { return ElementImage }

// TableElement represents a table
type TableElement struct {
	Headers   []string
	Rows      [][]string
	ColWidths []float64 // percentages
	Style     TableStyle
}

// TableStyle defines table appearance
type TableStyle struct {
	HeaderBg       Color
	HeaderText     Color
	BorderColor    Color
	BorderWidth    float64
	CellPadding    float64
	RowAlternating bool
	AltRowColor    Color
}

var DefaultTableStyle = TableStyle{
	HeaderBg:       ColorDarkGray,
	HeaderText:     ColorWhite,
	BorderColor:    ColorGray,
	BorderWidth:    0.5,
	CellPadding:    2,
	RowAlternating: true,
	AltRowColor:    ColorLightGray,
}

func (t *TableElement) GetType() ElementType { return ElementTable }

// LinkElement represents a hyperlink
type LinkElement struct {
	URL     string
	Content string
	Style   FontStyle
}

func (l *LinkElement) GetType() ElementType { return ElementLink }

// LineElement represents a horizontal rule
type LineElement struct {
	WidthPercent float64 // 0-100
	Color        Color
	Thickness    float64
}

func (l *LineElement) GetType() ElementType { return ElementLine }

// BulletItem represents a bullet point
type BulletItem struct {
	Content     string
	Style       FontStyle
	IndentLevel int // 0 = top level, 1 = sub-item, etc.
	SpaceBefore float64
	SpaceAfter  float64
}

func (b *BulletItem) GetType() ElementType { return ElementBullet }

// NumberedItem represents a numbered list item
type NumberedItem struct {
	Number      int
	Content     string
	Style       FontStyle
	IndentLevel int
	SpaceBefore float64
	SpaceAfter  float64
}

func (n *NumberedItem) GetType() ElementType { return ElementNumbered }

// =============================================
// Document Configuration
// =============================================

// PageSize defines page dimensions
type PageSize struct {
	WidthMM  float64
	HeightMM float64
	Name     string
}

// Standard page sizes
var (
	PageSizeA4     = PageSize{210, 297, "A4"}
	PageSizeLetter = PageSize{215.9, 279.4, "Letter"}
	PageSizeLegal  = PageSize{215.9, 355.6, "Legal"}
	PageSizeA5     = PageSize{148, 210, "A5"}
)

// PageConfig defines page layout
type PageConfig struct {
	Size         PageSize
	MarginTop    float64
	MarginBottom float64
	MarginLeft   float64
	MarginRight  float64
	Orientation  string // "portrait" or "landscape"
	Header       string
	Footer       string
	ShowPageNum  bool
}

// DefaultPageConfig returns A4 default settings
func DefaultPageConfig() PageConfig {
	return PageConfig{
		Size:         PageSizeA4,
		MarginTop:    25,
		MarginBottom: 25,
		MarginLeft:   25,
		MarginRight:  20,
		Orientation:  "portrait",
		ShowPageNum:  true,
	}
}

// Document represents a PDF document
type Document struct {
	elements  []PDFElement
	config    PageConfig
	fonts     map[FontFamily]bool
	images    map[string][]byte
	pageCount int
}

// NewDocument creates a new PDF document
func NewDocument() *Document {
	return &Document{
		elements: make([]PDFElement, 0),
		config:   DefaultPageConfig(),
		fonts:    make(map[FontFamily]bool),
		images:   make(map[string][]byte),
	}
}

// SetConfig sets the page configuration
func (d *Document) SetConfig(config PageConfig) {
	d.config = config
}

// AddText adds a text element
func (d *Document) AddText(content string, style FontStyle) {
	d.elements = append(d.elements, &TextElement{
		Content: content,
		Style:   style,
	})
	d.fonts[style.Family] = true
}

// AddHeading adds a heading element
func (d *Document) AddHeading(level int, content string) {
	var style FontStyle
	switch level {
	case 1:
		style = StyleH1
	case 2:
		style = StyleH2
	case 3:
		style = StyleH3
	default:
		style = StyleH3
	}
	d.elements = append(d.elements, &HeadingElement{
		Level:   level,
		Content: content,
		Style:   style,
	})
	d.fonts[style.Family] = true
}

// AddImage adds an image element
func (d *Document) AddImage(path string, width, height float64) {
	d.elements = append(d.elements, &ImageElement{
		Path:     path,
		WidthMM:  width,
		HeightMM: height,
		Align:    AlignCenter,
	})
}

// AddTable adds a table element
func (d *Document) AddTable(headers []string, rows [][]string, style TableStyle) {
	d.elements = append(d.elements, &TableElement{
		Headers:   headers,
		Rows:      rows,
		ColWidths: make([]float64, len(headers)),
		Style:     style,
	})
}

// AddLink adds a hyperlink
func (d *Document) AddLink(url, content string) {
	d.elements = append(d.elements, &LinkElement{
		URL:     url,
		Content: content,
		Style:   StyleLink,
	})
	d.fonts[StyleLink.Family] = true
}

// AddHorizontalLine adds a horizontal line
func (d *Document) AddHorizontalLine(color Color, thickness float64) {
	d.elements = append(d.elements, &LineElement{
		WidthPercent: 100,
		Color:        color,
		Thickness:    thickness,
	})
}

// AddPageBreak forces a new page
func (d *Document) AddPageBreak() {
	d.elements = append(d.elements, &TextElement{Content: "", Style: StyleBody})
}

// AddBullet adds a bullet point
func (d *Document) AddBullet(content string, indentLevel int) {
	style := StyleBullet
	if indentLevel > 0 {
		style.Size = 10
	}
	d.elements = append(d.elements, &BulletItem{
		Content:     content,
		Style:       style,
		IndentLevel: indentLevel,
	})
	d.fonts[style.Family] = true
}

// AddNumbered adds a numbered list item
func (d *Document) AddNumbered(number int, content string, indentLevel int) {
	style := StyleBullet
	if indentLevel > 0 {
		style.Size = 10
	}
	d.elements = append(d.elements, &NumberedItem{
		Number:      number,
		Content:     content,
		Style:       style,
		IndentLevel: indentLevel,
	})
	d.fonts[style.Family] = true
}

// =============================================
// Rich Content Parser
// =============================================

var (
	headingRe  = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	bulletRe   = regexp.MustCompile(`^[\-\*]\s+(.+)$`)
	numberedRe = regexp.MustCompile(`^(\d+)\.?\s+(.+)$`)
	linkRe     = regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)
	imageRe    = regexp.MustCompile(`!\[([^\]]*)\]\(([^\)]+)\)`)
	hrRe       = regexp.MustCompile(`^[-*_]{3,}$`)
)

// ParseMarkdown converts markdown-like text to PDF elements
func (d *Document) ParseMarkdown(content string) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Empty line
		if trimmed == "" {
			continue
		}

		// Horizontal rule
		if hrRe.MatchString(trimmed) {
			d.AddHorizontalLine(ColorGray, 0.5)
			continue
		}

		// Headings
		if match := headingRe.FindStringSubmatch(trimmed); match != nil {
			level := len(match[1])
			d.AddHeading(level, match[2])
			continue
		}

		// Links
		if match := linkRe.FindStringSubmatch(trimmed); match != nil {
			d.AddLink(match[2], match[1])
			continue
		}

		// Images
		if match := imageRe.FindStringSubmatch(trimmed); match != nil {
			d.AddImage(match[2], 100, 75) // Default size
			continue
		}

		// Numbered lists
		if match := numberedRe.FindStringSubmatch(trimmed); match != nil {
			num, _ := strconv.Atoi(match[1])
			d.AddNumbered(num, match[2], 0)
			continue
		}

		// Bullet lists
		if match := bulletRe.FindStringSubmatch(trimmed); match != nil {
			d.AddBullet(match[1], 0)
			continue
		}

		// Regular text - treat as paragraph
		d.AddText(trimmed, StyleBody)
	}
}

// Convert to io.Reader for output
func (d *Document) ToReader() (io.Reader, error) {
	var buf bytes.Buffer
	err := d.generatePDF(&buf)
	if err != nil {
		return nil, err
	}
	return &buf, nil
}

// GetPageCount returns the number of pages
func (d *Document) GetPageCount() int {
	return d.pageCount
}

// generatePDF generates the actual PDF content
func (d *Document) generatePDF(w io.Writer) error {
	// This is a placeholder - actual PDF generation is in pdfexport.go
	// This allows the document model to be used with the existing exporter
	fmt.Fprintf(w, "PDF Document with %d elements\n", len(d.elements))
	return nil
}
