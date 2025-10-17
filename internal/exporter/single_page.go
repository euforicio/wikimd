package exporter

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	pdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// Format represents an export format.
type Format string

const (
	// FormatHTML exports as HTML.
	FormatHTML Format = "html"
	// FormatMarkdown exports as markdown.
	FormatMarkdown Format = "markdown"
	// FormatPlainText exports as plain text.
	FormatPlainText Format = "txt"
	// FormatPDF exports as PDF.
	FormatPDF Format = "pdf"
)

// ValidFormats returns the list of supported export formats.
func ValidFormats() []Format {
	return []Format{FormatHTML, FormatMarkdown, FormatPlainText, FormatPDF}
}

// IsValidFormat checks if the given format is valid.
func IsValidFormat(format string) bool {
	f := Format(strings.ToLower(strings.TrimSpace(format)))
	for _, valid := range ValidFormats() {
		if f == valid {
			return true
		}
	}
	return false
}

// ExportPageOptions configures a single page export.
type ExportPageOptions struct {
	Writer  io.Writer
	Format  Format
	RootDir string
	Path    string
}

// ExportPage exports a single page in the specified format.
func (e *Exporter) ExportPage(ctx context.Context, opts ExportPageOptions) error {
	if err := validateExportPageOptions(opts); err != nil {
		return err
	}

	rootDir, err := filepath.Abs(opts.RootDir)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}

	absPath, err := resolveExportPath(rootDir, opts.Path)
	if err != nil {
		return err
	}

	info, raw, err := readExportSource(absPath, opts.Path)
	if err != nil {
		return err
	}

	switch opts.Format {
	case FormatHTML:
		return e.exportHTML(ctx, absPath, info.ModTime(), raw, opts.Writer)
	case FormatMarkdown:
		return e.exportMarkdown(raw, opts.Writer)
	case FormatPlainText:
		return e.exportPlainText(ctx, absPath, info.ModTime(), raw, opts.Writer)
	case FormatPDF:
		return e.exportPDF(ctx, absPath, info.ModTime(), raw, opts.Writer)
	default:
		return fmt.Errorf("unsupported format: %s", opts.Format)
	}
}

func validateExportPageOptions(opts ExportPageOptions) error {
	if strings.TrimSpace(opts.RootDir) == "" {
		return errors.New("root directory is required")
	}
	if strings.TrimSpace(opts.Path) == "" {
		return errors.New("page path is required")
	}
	if opts.Writer == nil {
		return errors.New("writer is required")
	}
	if !IsValidFormat(string(opts.Format)) {
		return fmt.Errorf("unsupported format: %s (allowed: html, pdf, markdown, txt)", opts.Format)
	}
	return nil
}

func resolveExportPath(rootDir, pagePath string) (string, error) {
	cleanPath := filepath.Clean(pagePath)
	if strings.Contains(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		return "", errors.New("invalid path: directory traversal not allowed")
	}

	absPath := filepath.Join(rootDir, filepath.FromSlash(cleanPath))
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	if !strings.HasPrefix(absPath, rootDir+string(filepath.Separator)) && absPath != rootDir {
		return "", errors.New("invalid path: must be within root directory")
	}

	return absPath, nil
}

func readExportSource(absPath, originalPath string) (os.FileInfo, []byte, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("page not found: %s", originalPath)
		}
		return nil, nil, fmt.Errorf("stat page: %w", err)
	}

	raw, err := os.ReadFile(absPath) //nolint:gosec // absPath constructed from validated root
	if err != nil {
		return nil, nil, fmt.Errorf("read page: %w", err)
	}

	return info, raw, nil
}

func (e *Exporter) exportHTML(ctx context.Context, path string, modTime time.Time, raw []byte, w io.Writer) error {
	doc, err := e.renderer.Render(ctx, path, modTime, raw)
	if err != nil {
		return fmt.Errorf("render html: %w", err)
	}

	// Create a standalone HTML document
	tmplStr := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }}</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      line-height: 1.6;
      max-width: 800px;
      margin: 0 auto;
      padding: 2rem;
      color: #333;
    }
    h1, h2, h3, h4, h5, h6 {
      margin-top: 1.5em;
      margin-bottom: 0.5em;
      line-height: 1.2;
    }
    h1 { font-size: 2em; border-bottom: 1px solid #eee; padding-bottom: 0.3em; }
    h2 { font-size: 1.5em; }
    h3 { font-size: 1.25em; }
    code {
      background: #f5f5f5;
      padding: 0.2em 0.4em;
      border-radius: 3px;
      font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
      font-size: 0.9em;
    }
    pre {
      background: #f5f5f5;
      padding: 1em;
      border-radius: 5px;
      overflow-x: auto;
    }
    pre code {
      background: none;
      padding: 0;
    }
    blockquote {
      border-left: 4px solid #ddd;
      padding-left: 1em;
      margin-left: 0;
      color: #666;
    }
    table {
      border-collapse: collapse;
      width: 100%;
      margin: 1em 0;
    }
    th, td {
      border: 1px solid #ddd;
      padding: 0.5em;
      text-align: left;
    }
    th {
      background: #f5f5f5;
      font-weight: bold;
    }
    img {
      max-width: 100%;
      height: auto;
    }
    a {
      color: #0366d6;
      text-decoration: none;
    }
    a:hover {
      text-decoration: underline;
    }
  </style>
</head>
<body>
{{ if .Title }}<h1>{{ .Title }}</h1>{{ end }}
{{ .HTML }}
</body>
</html>`

	tmpl, err := template.New("export").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	data := struct {
		Title string
		HTML  template.HTML
	}{
		Title: doc.Metadata.Title,
		HTML:  template.HTML(doc.HTML), //nolint:gosec // HTML from trusted renderer
	}

	return tmpl.Execute(w, data)
}

func (e *Exporter) exportMarkdown(raw []byte, w io.Writer) error {
	_, err := w.Write(raw)
	return err
}

func (e *Exporter) exportPlainText(ctx context.Context, path string, modTime time.Time, raw []byte, w io.Writer) error {
	doc, err := e.renderer.Render(ctx, path, modTime, raw)
	if err != nil {
		return fmt.Errorf("render text: %w", err)
	}

	// Strip HTML tags for plain text
	text := stripHTML(doc.HTML)
	_, err = w.Write([]byte(text))
	return err
}

func (e *Exporter) exportPDF(_ context.Context, _ string, _ time.Time, raw []byte, w io.Writer) error {
	// Create a goldmark parser with PDF renderer and extensions
	// Note: PDF renderer doesn't support hard wraps configuration like HTML renderer
	pdfRenderer := pdf.New()
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,           // GitHub Flavored Markdown
			extension.Table,         // Tables
			extension.Strikethrough, // Strikethrough text
			extension.TaskList,      // Task lists
			meta.Meta,               // Frontmatter metadata
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Auto-generate heading IDs
		),
		goldmark.WithRenderer(pdfRenderer),
	)

	// Parse and render markdown to PDF
	if err := md.Convert(raw, w); err != nil {
		return fmt.Errorf("convert markdown to PDF: %w", err)
	}

	return nil
}

// stripHTML removes HTML tags from a string.
func stripHTML(html string) string {
	// Simple HTML tag removal - not perfect but sufficient for basic text extraction
	// First, remove script and style tags with their content
	html = removeTagWithContent(html, "script")
	html = removeTagWithContent(html, "style")

	var result strings.Builder
	inTag := false

	for i := 0; i < len(html); i++ {
		char := html[i]

		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}

		if !inTag {
			result.WriteByte(char)
		}
	}

	// Clean up whitespace
	text := result.String()
	text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	text = strings.TrimSpace(text)

	return text
}

// removeTagWithContent removes all occurrences of a tag and its content.
func removeTagWithContent(html, tag string) string {
	for {
		openTag := "<" + tag
		closeTag := "</" + tag + ">"

		// Find opening tag
		start := strings.Index(strings.ToLower(html), strings.ToLower(openTag))
		if start == -1 {
			break
		}

		// Find the end of the opening tag
		tagEnd := strings.Index(html[start:], ">")
		if tagEnd == -1 {
			break
		}

		// Find closing tag
		end := strings.Index(strings.ToLower(html[start:]), strings.ToLower(closeTag))
		if end == -1 {
			break
		}

		// Remove the tag and its content
		html = html[:start] + html[start+end+len(closeTag):]
	}

	return html
}

// ContentType returns the MIME type for the given format.
func ContentType(format Format) string {
	switch format {
	case FormatHTML:
		return "text/html; charset=utf-8"
	case FormatMarkdown:
		return "text/markdown; charset=utf-8"
	case FormatPlainText:
		return "text/plain; charset=utf-8"
	case FormatPDF:
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

// FileExtension returns the file extension for the given format.
func FileExtension(format Format) string {
	switch format {
	case FormatHTML:
		return ".html"
	case FormatMarkdown:
		return ".md"
	case FormatPlainText:
		return ".txt"
	case FormatPDF:
		return ".pdf"
	default:
		return ""
	}
}
