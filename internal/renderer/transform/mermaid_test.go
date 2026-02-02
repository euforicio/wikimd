package transform_test

import (
	"bytes"
	"testing"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"

	"github.com/euforicio/wikimd/internal/renderer/transform"
)

func TestMermaidWrapper(t *testing.T) {
	f := func(name, input, wantContains, wantNotContains string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			md := goldmark.New(
				goldmark.WithExtensions(
					highlighting.NewHighlighting(
						highlighting.WithStyle("github-dark"),
						highlighting.WithFormatOptions(
							html.WithLineNumbers(false),
							html.WithClasses(true),
						),
						highlighting.WithWrapperRenderer(transform.MermaidWrapper()),
					),
				),
			)

			var buf bytes.Buffer
			if err := md.Convert([]byte(input), &buf); err != nil {
				t.Fatalf("convert failed: %v", err)
			}
			output := buf.String()

			if wantContains != "" && !bytes.Contains([]byte(output), []byte(wantContains)) {
				t.Fatalf("expected output to contain %q, got:\n%s", wantContains, output)
			}
			if wantNotContains != "" && bytes.Contains([]byte(output), []byte(wantNotContains)) {
				t.Fatalf("expected output NOT to contain %q, got:\n%s", wantNotContains, output)
			}
		})
	}

	// Mermaid blocks should NOT have wrapper or copy button
	f("mermaid no wrapper",
		"```mermaid\ngraph TD\nA-->B\n```\n",
		`<div class="mermaid">`,
		`code-block-wrapper`)

	// Go code blocks should have wrapper and copy button
	f("go code with wrapper",
		"```go\npackage main\n```\n",
		`<div class="code-block-wrapper">`,
		"")

	f("go code with copy button",
		"```go\npackage main\n```\n",
		`<button class="code-copy-button"`,
		"")

	// Non-highlighted code blocks should also have wrapper
	f("unknown lang with wrapper",
		"```unknown\nsome code\n```\n",
		`<div class="code-block-wrapper">`,
		"")

	f("unknown lang with copy button",
		"```unknown\nsome code\n```\n",
		`<button class="code-copy-button"`,
		"")

	// Plain code blocks (no language) should have wrapper
	f("plain code with wrapper",
		"```\nplain code\n```\n",
		`<div class="code-block-wrapper">`,
		"")
}
