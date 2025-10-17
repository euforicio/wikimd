// Package transform provides custom rendering transformations for markdown elements.
package transform

import (
	"bytes"
	"strings"

	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/util"
)

const mermaidLanguage = "mermaid"

// MermaidWrapper returns a wrapper renderer that converts ```mermaid fences into divs
// Mermaid.js can hydrate, while delegating to the default fallback for other code blocks.
func MermaidWrapper() highlighting.WrapperRenderer {
	return func(w util.BufWriter, ctx highlighting.CodeBlockContext, entering bool) {
		if ctx.Highlighted() {
			// Let the highlighter handle its own wrappers for highlighted blocks.
			return
		}

		lang, _ := ctx.Language()
		normalized := strings.TrimSpace(strings.ToLower(string(lang)))
		if normalized == mermaidLanguage {
			if entering {
				_, _ = w.WriteString(`<div class="mermaid">`)
			} else {
				_, _ = w.WriteString("</div>\n")
			}
			return
		}

		if entering {
			_, _ = w.WriteString("<pre><code")
			if len(bytes.TrimSpace(lang)) > 0 {
				_, _ = w.WriteString(` class="language-`)
				_, _ = w.Write(util.EscapeHTML(lang))
				_, _ = w.WriteString(`"`)
			}
			_, _ = w.WriteString(">")
			return
		}
		_, _ = w.WriteString("</code></pre>\n")
	}
}
