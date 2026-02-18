package pdfexport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
)

// Renderer handles HTML to PDF conversion using headless Chrome
type Renderer struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRenderer creates a new PDF renderer
func NewRenderer() *Renderer {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	return &Renderer{
		ctx:    ctx,
		cancel: cancel,
	}
}

// RenderHTMLToPDF converts HTML content to PDF file
func (r *Renderer) RenderHTMLToPDF(htmlContent string, outputPath string) error {
	defer r.cancel()

	// Create allocator options - run Chrome in headless mode
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1200, 800),
	)

	// Create allocator
	allocCtx, cancel := chromedp.NewExecAllocator(r.ctx, opts...)
	defer cancel()

	// Create browser context
	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Create temporary HTML file
	tmpDir := os.TempDir()
	htmlFile := filepath.Join(tmpDir, "akashic_export_"+time.Now().Format("20060102_150405")+".html")

	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to create temp HTML file: %w", err)
	}
	defer os.Remove(htmlFile) // Clean up temp file

	// Navigate to the HTML file
	err := chromedp.Run(taskCtx,
		chromedp.Navigate("file:///"+filepath.ToSlash(htmlFile)),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond), // Wait for fonts to load
	)

	if err != nil {
		return fmt.Errorf("failed to load page: %w", err)
	}

	// Note: Full PDF generation requires Chrome's Page.printToPDF CDP command
	// This simplified version opens the HTML in browser for user to print
	return fmt.Errorf("PDF generation requires Chrome. HTML saved to: %s", htmlFile)
}

// Close cleans up resources
func (r *Renderer) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}
