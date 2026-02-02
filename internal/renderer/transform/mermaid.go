// Package transform provides custom rendering transformations for markdown elements.
package transform

import (
	"bytes"
	"strings"

	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/util"
)

const mermaidLanguage = "mermaid"

// copyButtonHTML is the HTML for the code block copy button, rendered server-side.
const copyButtonHTML = `<button class="code-copy-button" type="button" aria-label="Copy code to clipboard">
<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
<rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
</svg>
<span>Copy</span>
</button>`

// MermaidWrapper returns a wrapper renderer that converts ```mermaid fences into divs
// Mermaid.js can hydrate, while wrapping other code blocks with a copy button.
// Non-mermaid code blocks are wrapped with .code-block-wrapper div and include
// the copy button HTML server-side to reduce client-side DOM manipulation.
func MermaidWrapper() highlighting.WrapperRenderer {
	return func(w util.BufWriter, ctx highlighting.CodeBlockContext, entering bool) {
		lang, _ := ctx.Language()
		normalized := strings.TrimSpace(strings.ToLower(string(lang)))

		// Mermaid blocks get their own special wrapper without copy button
		if normalized == mermaidLanguage {
			if entering {
				_, _ = w.WriteString(`<div class="mermaid">`)
			} else {
				_, _ = w.WriteString("</div>\n")
			}
			return
		}

		// For highlighted code blocks, we need to wrap the chroma output
		if ctx.Highlighted() {
			if entering {
				_, _ = w.WriteString(`<div class="code-block-wrapper">`)
			} else {
				_, _ = w.WriteString(copyButtonHTML)
				_, _ = w.WriteString("</div>\n")
			}
			return
		}

		// Non-highlighted code blocks: wrap with copy button
		if entering {
			_, _ = w.WriteString(`<div class="code-block-wrapper">`)
			_, _ = w.WriteString("<pre><code")
			if len(bytes.TrimSpace(lang)) > 0 {
				_, _ = w.WriteString(` class="language-`)
				_, _ = w.Write(util.EscapeHTML(lang))
				_, _ = w.WriteString(`"`)
			}
			_, _ = w.WriteString(">")
			return
		}
		_, _ = w.WriteString("</code></pre>")
		_, _ = w.WriteString(copyButtonHTML)
		_, _ = w.WriteString("</div>\n")
	}
}
