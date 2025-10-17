package exporter

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportPage(t *testing.T) {
	t.Parallel()
	// Create test data
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	testContent := []byte("# Test Page\n\nThis is a test page with **bold** text.")
	if err := os.WriteFile(testFile, testContent, 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	exp, err := New(logger)
	if err != nil {
		t.Fatalf("failed to create exporter: %v", err)
	}

	ctx := context.Background()

	t.Run("HTML export", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "test.md",
			Format:  FormatHTML,
			Writer:  &buf,
		})
		if err != nil {
			t.Fatalf("HTML export failed: %v", err)
		}

		html := buf.String()
		if !strings.Contains(html, "<!DOCTYPE html>") {
			t.Error("HTML export missing DOCTYPE")
		}
		if !strings.Contains(html, "Test Page") {
			t.Error("HTML export missing content")
		}
	})

	t.Run("Markdown export", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "test.md",
			Format:  FormatMarkdown,
			Writer:  &buf,
		})
		if err != nil {
			t.Fatalf("Markdown export failed: %v", err)
		}

		md := buf.String()
		if md != string(testContent) {
			t.Errorf("Markdown export mismatch:\ngot:  %q\nwant: %q", md, string(testContent))
		}
	})

	t.Run("Plain text export", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "test.md",
			Format:  FormatPlainText,
			Writer:  &buf,
		})
		if err != nil {
			t.Fatalf("Plain text export failed: %v", err)
		}

		text := buf.String()
		if !strings.Contains(text, "Test Page") {
			t.Error("Plain text export missing content")
		}
		if strings.Contains(text, "<") || strings.Contains(text, ">") {
			t.Error("Plain text export contains HTML tags")
		}
	})

	t.Run("PDF export", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "test.md",
			Format:  FormatPDF,
			Writer:  &buf,
		})
		if err != nil {
			t.Fatalf("PDF export failed: %v", err)
		}

		// Check that we got PDF content (PDF files start with %PDF-)
		pdfContent := buf.Bytes()
		if len(pdfContent) < 5 {
			t.Error("PDF export returned empty content")
		} else if !bytes.HasPrefix(pdfContent, []byte("%PDF-")) {
			t.Errorf("PDF export did not return valid PDF (got %q...)", string(pdfContent[:min(20, len(pdfContent))]))
		}
	})

	t.Run("PDF export with complex markdown", func(t *testing.T) {
		t.Parallel()
		// Create a file with various markdown features
		complexFile := filepath.Join(tmpDir, "complex.md")
		complexContent := []byte(`# Main Title

## Section 1

This is a paragraph with **bold** and *italic* text.

### Code Block

` + "```go\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n```" + `

### Lists

- Item 1
- Item 2
  - Nested item
- Item 3

### Table

| Column 1 | Column 2 |
|----------|----------|
| Data 1   | Data 2   |
| Data 3   | Data 4   |

### Task List

- [x] Completed task
- [ ] Pending task

> This is a blockquote.

## Section 2

Some more content with ~~strikethrough~~ text.
`)
		if err := os.WriteFile(complexFile, complexContent, 0o644); err != nil {
			t.Fatalf("failed to create complex test file: %v", err)
		}

		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "complex.md",
			Format:  FormatPDF,
			Writer:  &buf,
		})
		if err != nil {
			t.Fatalf("PDF export with complex markdown failed: %v", err)
		}

		// Check that we got valid PDF content
		pdfContent := buf.Bytes()
		if len(pdfContent) < 100 {
			t.Error("PDF export returned suspiciously small content")
		}
		if !bytes.HasPrefix(pdfContent, []byte("%PDF-")) {
			t.Error("PDF export did not return valid PDF header")
		}
		// PDF files should end with %%EOF
		if !bytes.Contains(pdfContent, []byte("%%EOF")) {
			t.Error("PDF export did not contain EOF marker")
		}
	})

	t.Run("Invalid format", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "test.md",
			Format:  "invalid",
			Writer:  &buf,
		})
		if err == nil {
			t.Error("Expected error for invalid format")
		}
	})

	t.Run("Missing file", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "missing.md",
			Format:  FormatHTML,
			Writer:  &buf,
		})
		if err == nil {
			t.Error("Expected error for missing file")
		}
	})

	t.Run("Path traversal with ..", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "../etc/passwd",
			Format:  FormatHTML,
			Writer:  &buf,
		})
		if err == nil {
			t.Error("Expected error for path traversal attempt with ..")
		}
		if err != nil && !strings.Contains(err.Error(), "directory traversal") {
			t.Errorf("Expected 'directory traversal' error, got: %v", err)
		}
	})

	t.Run("Path traversal with multiple ..", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "../../etc/passwd",
			Format:  FormatHTML,
			Writer:  &buf,
		})
		if err == nil {
			t.Error("Expected error for path traversal attempt with multiple ..")
		}
		if err != nil && !strings.Contains(err.Error(), "directory traversal") {
			t.Errorf("Expected 'directory traversal' error, got: %v", err)
		}
	})

	t.Run("Absolute path", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "/etc/passwd",
			Format:  FormatHTML,
			Writer:  &buf,
		})
		if err == nil {
			t.Error("Expected error for absolute path attempt")
		}
		// Note: On Windows, /etc/passwd is treated as relative path, so it may return "page not found"
		// rather than "directory traversal" error. Both are acceptable rejections.
		if err != nil && !strings.Contains(err.Error(), "directory traversal") && !strings.Contains(err.Error(), "page not found") {
			t.Errorf("Expected 'directory traversal' or 'page not found' error, got: %v", err)
		}
	})

	t.Run("Path with .. in middle", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "subdir/../../../etc/passwd",
			Format:  FormatHTML,
			Writer:  &buf,
		})
		if err == nil {
			t.Error("Expected error for path traversal attempt with .. in middle")
		}
		if err != nil && !strings.Contains(err.Error(), "directory traversal") {
			t.Errorf("Expected 'directory traversal' error, got: %v", err)
		}
	})

	t.Run("Valid path with subdirectory", func(t *testing.T) {
		t.Parallel()
		// Create a subdirectory with a test file
		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}
		subFile := filepath.Join(subDir, "nested.md")
		if err := os.WriteFile(subFile, []byte("# Nested"), 0o644); err != nil {
			t.Fatalf("failed to create nested file: %v", err)
		}

		var buf bytes.Buffer
		err := exp.ExportPage(ctx, ExportPageOptions{
			RootDir: tmpDir,
			Path:    "subdir/nested.md",
			Format:  FormatHTML,
			Writer:  &buf,
		})
		if err != nil {
			t.Errorf("Valid nested path should succeed, got error: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("Expected content in export buffer")
		}
	})
}

func TestIsValidFormat(t *testing.T) {
	t.Parallel()
	f := func(format string, expected bool) {
		t.Helper()
		result := IsValidFormat(format)
		if result != expected {
			t.Errorf("IsValidFormat(%q) = %v, want %v", format, result, expected)
		}
	}

	f("html", true)
	f("HTML", true)
	f("pdf", true)
	f("markdown", true)
	f("txt", true)
	f("invalid", false)
	f("", false)
	f("json", false)
}

func TestContentType(t *testing.T) {
	t.Parallel()
	f := func(format Format, expected string) {
		t.Helper()
		result := ContentType(format)
		if result != expected {
			t.Errorf("ContentType(%q) = %q, want %q", format, result, expected)
		}
	}

	f(FormatHTML, "text/html; charset=utf-8")
	f(FormatMarkdown, "text/markdown; charset=utf-8")
	f(FormatPlainText, "text/plain; charset=utf-8")
	f(FormatPDF, "application/pdf")
	f("invalid", "application/octet-stream")
}

func TestFileExtension(t *testing.T) {
	t.Parallel()
	f := func(format Format, expected string) {
		t.Helper()
		result := FileExtension(format)
		if result != expected {
			t.Errorf("FileExtension(%q) = %q, want %q", format, result, expected)
		}
	}

	f(FormatHTML, ".html")
	f(FormatMarkdown, ".md")
	f(FormatPlainText, ".txt")
	f(FormatPDF, ".pdf")
	f("invalid", "")
}

func TestStripHTML(t *testing.T) {
	t.Parallel()
	f := func(html, expected string) {
		t.Helper()
		result := stripHTML(html)
		result = strings.TrimSpace(result)
		expected = strings.TrimSpace(expected)
		if result != expected {
			t.Errorf("stripHTML(%q) =\n%q\nwant:\n%q", html, result, expected)
		}
	}

	f("<p>Hello world</p>", "Hello world")
	f("<h1>Title</h1><p>Content</p>", "TitleContent")
	f("<script>alert('test')</script><p>Text</p>", "Text")
	f("<style>.class{color:red}</style><p>Text</p>", "Text")
	f("Plain text", "Plain text")
	f("<a href='test'>Link</a>", "Link")
}
