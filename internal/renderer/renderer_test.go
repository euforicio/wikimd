package renderer_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/euforicio/wikimd/internal/renderer"
)

func TestRenderWithMetadataAndMermaid(t *testing.T) {
	t.Parallel()
	svc := renderer.NewService(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))

	content := []byte("---\n" +
		"title: Example Doc\n" +
		"description: Sample description\n" +
		"tags:\n" +
		"  - go\n" +
		"  - wiki\n" +
		"---\n\n" +
		"# Hello\n\n" +
		"Some inline text.\n\n" +
		"```mermaid\n" +
		"graph TD;\n" +
		"A-->B;\n" +
		"```\n\n" +
		"```go\n" +
		"package main\n\n" +
		"import \"fmt\"\n\n" +
		"func main() {\n" +
		"  fmt.Println(\"hello\")\n" +
		"}\n" +
		"```\n")

	modTime := time.Unix(1_000, 0)
	doc, err := svc.Render(context.Background(), "docs/example.md", modTime, content)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if doc.Metadata.Title != "Example Doc" {
		t.Fatalf("expected title 'Example Doc', got %q", doc.Metadata.Title)
	}
	if doc.Metadata.Description != "Sample description" {
		t.Fatalf("unexpected description: %q", doc.Metadata.Description)
	}
	if len(doc.Metadata.Tags) != 2 || doc.Metadata.Tags[0] != "go" || doc.Metadata.Tags[1] != "wiki" {
		t.Fatalf("unexpected tags: %#v", doc.Metadata.Tags)
	}

	html := doc.HTML
	if !strings.Contains(html, `<div class="mermaid">`) {
		t.Fatalf("expected mermaid div in HTML, got %s", html)
	}
	if strings.Contains(html, "language-mermaid") {
		t.Fatalf("expected mermaid fence to be wrapped, saw raw language class: %s", html)
	}
	if !strings.Contains(html, "graph TD;") {
		t.Fatalf("expected mermaid content in HTML")
	}
	if !strings.Contains(html, `class="chroma"`) {
		t.Fatalf("expected chroma highlighter output, got %s", html)
	}
	if !strings.Contains(html, `<span class="kn">package</span>`) {
		t.Fatalf("expected go syntax tokens in HTML, got %s", html)
	}
	if !doc.Modified.Equal(modTime) {
		t.Fatalf("expected modified timestamp to match, got %v", doc.Modified)
	}
}

func TestRenderCaching(t *testing.T) {
	t.Parallel()
	svc := renderer.NewService(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))

	ctx := context.Background()
	path := "docs/cache.md"
	modTime := time.Unix(2_000, 0)

	doc1, err := svc.Render(ctx, path, modTime, []byte("# First"))
	if err != nil {
		t.Fatalf("first render: %v", err)
	}

	doc2, err := svc.Render(ctx, path, modTime, []byte("# Second"))
	if err != nil {
		t.Fatalf("second render: %v", err)
	}
	if doc2.HTML != doc1.HTML {
		t.Fatalf("expected cached HTML, got different output")
	}

	doc3, err := svc.Render(ctx, path, modTime.Add(time.Second), []byte("# Second"))
	if err != nil {
		t.Fatalf("third render: %v", err)
	}
	if doc3.HTML == doc1.HTML {
		t.Fatalf("expected updated render after mod time change")
	}
	if !strings.Contains(doc3.HTML, "Second") {
		t.Fatalf("expected new HTML to include updated content, got %s", doc3.HTML)
	}
}
