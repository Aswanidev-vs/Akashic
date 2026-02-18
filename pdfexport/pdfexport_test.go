package pdfexport

import (
"os"
"testing"
)

func TestExport(t *testing.T) {
exporter := NewExporter()

content := `Role Definition

You are a senior software engineer, security auditor, and technical documentation specialist.
Your responsibility is to perform a rigorous, production-grade code review and generate structured documentation for every issue you discover.

Your objectives are:

Detect bugs, logic flaws, architectural weaknesses, performance problems, and code smells.

Identify security vulnerabilities and classify their severity.

Evaluate maintainability, scalability, and readability.

Provide precise improvement suggestions.

Generate structured README files documenting each issue and its resolution path.`

err := exporter.Export(content, "test_output.pdf")
if err != nil {
t.Fatalf("Export failed: %v", err)
}

// Check file was created
if _, err := os.Stat("test_output.pdf"); os.IsNotExist(err) {
t.Fatal("PDF file was not created")
}

// Clean up
os.Remove("test_output.pdf")
t.Log("PDF generated successfully")
}
