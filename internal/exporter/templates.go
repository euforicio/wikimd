package exporter

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/euforicio/wikimd/internal/renderer"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

type templateRenderer struct {
	tmpl *template.Template
}

func newTemplateRenderer() (*templateRenderer, error) {
	funcs := template.FuncMap{
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires even number of args")
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				m[key] = values[i+1]
			}
			return m, nil
		},
		"isActive": func(active, candidate string) bool {
			return strings.EqualFold(strings.TrimSpace(active), strings.TrimSpace(candidate))
		},
		"formatTime": func(t time.Time) string {
			if t.IsZero() {
				return "—"
			}
			return t.Local().Format("Jan 2, 2006 3:04 PM")
		},
		"hasMetadata": func(meta renderer.Metadata) bool {
			return !meta.IsZero()
		},
		"pageURL": toHTMLRel,
	}

	base, err := template.New("layout").Funcs(funcs).ParseFS(templateFS, "templates/*.gohtml")
	if err != nil {
		return nil, err
	}
	return &templateRenderer{tmpl: base}, nil
}

func (r *templateRenderer) render(w io.Writer, name string, data any) error {
	return r.tmpl.ExecuteTemplate(w, name, data)
}
