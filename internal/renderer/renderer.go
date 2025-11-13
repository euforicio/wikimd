// Package renderer converts markdown to HTML with caching and syntax highlighting.
package renderer

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	goldmarkmeta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	htmlrenderer "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/anchor"

	"github.com/euforicio/wikimd/internal/renderer/transform"
)

// Metadata captures optional frontmatter data rendered alongside a document.
type Metadata struct {
	Raw         map[string]any
	Title       string
	Description string
	Tags        []string
}

// IsZero reports whether the metadata carries any meaningful values.
func (m Metadata) IsZero() bool {
	if m.Title != "" || m.Description != "" || len(m.Tags) > 0 {
		return false
	}
	return len(m.Raw) == 0
}

// Document represents a rendered markdown file.
//
//nolint:govet // field order optimized for readability, not memory
type Document struct {
	HTML     string
	Metadata Metadata
	Modified time.Time
	Raw      string
}

type cacheEntry struct {
	modTime time.Time
	doc     Document
}

type cacheKey string

// Service renders markdown into HTML with caching.
// It uses Goldmark for markdown parsing with GitHub-flavored markdown extensions,
// syntax highlighting, and automatic link transformation for wiki-style navigation.
// Rendered documents are cached by path and modification time for improved performance.
type Service struct {
	md     goldmark.Markdown
	logger *slog.Logger
	cache  sync.Map // map[cacheKey]cacheEntry
}

// contextKey for storing document path
var docPathKey = parser.NewContextKey()

// linkTransformer rewrites .md links to /page/ routes and image paths to /media/ routes
type linkTransformer struct{}

func (t *linkTransformer) Transform(node *ast.Document, _ text.Reader, pc parser.Context) {
	// Get current document path from context (wiki-relative path)
	currentPath := ""
	if v := pc.Get(docPathKey); v != nil {
		if str, ok := v.(string); ok {
			currentPath = str
		}
	}

	// Get directory of current document (wiki-relative)
	currentDir := path.Dir(currentPath)

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch typed := n.(type) {
		case *ast.Link:
			t.transformLink(typed, currentDir)
		case *ast.Image:
			t.transformImage(typed, currentDir)
		}

		return ast.WalkContinue, nil
	})
}

func (t *linkTransformer) transformLink(link *ast.Link, currentDir string) {
	dest := string(link.Destination)
	if dest == "" || t.isExternalLink(dest) || strings.HasPrefix(dest, "#") || strings.HasPrefix(dest, "/page/") {
		return
	}

	if !strings.HasSuffix(dest, ".md") {
		return
	}

	link.Destination = []byte("/page/" + normalizeWikiPath(dest, currentDir))
}

func (t *linkTransformer) transformImage(img *ast.Image, currentDir string) {
	dest := string(img.Destination)
	if dest == "" || t.isExternalLink(dest) || strings.HasPrefix(dest, "/media/") || strings.HasPrefix(dest, "/static/") {
		return
	}

	img.Destination = []byte("/media/" + normalizeWikiPath(dest, currentDir))
}

func (t *linkTransformer) isExternalLink(dest string) bool {
	return strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://")
}

func normalizeWikiPath(dest, currentDir string) string {
	if !strings.HasPrefix(dest, "/") {
		if currentDir != "" && currentDir != "." {
			dest = path.Join(currentDir, dest)
		}
		dest = path.Clean(dest)
	}

	return strings.TrimPrefix(dest, "/")
}

// NewService constructs a markdown renderer with GitHub-flavored markdown support.
// The renderer includes:
//   - GitHub-flavored markdown extensions (tables, strikethrough, task lists, autolinks, etc.)
//   - Syntax highlighting with the github-dark theme
//   - YAML frontmatter parsing for document metadata
//   - Automatic link transformation for .md files to /page/ routes
//   - Raw HTML rendering enabled (safe for local-only wikis)
//   - Soft line breaks (newlines become spaces, matching GitHub's default behavior)
//   - Hard line breaks can be created with two trailing spaces or <br> tags
//
// If logger is nil, the default slog logger is used.
func NewService(logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	highlight := highlighting.NewHighlighting(
		highlighting.WithStyle("github-dark"),
		highlighting.WithFormatOptions(
			html.WithLineNumbers(false),
			html.WithClasses(true),
		),
		highlighting.WithWrapperRenderer(transform.MermaidWrapper()),
	)

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			goldmarkmeta.Meta,
			highlight,
			&anchor.Extender{
				Position: anchor.After, // Place anchor link after heading text
			},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(), // Enable attribute syntax for blocks and inlines
			parser.WithASTTransformers(
				util.Prioritized(&linkTransformer{}, 100),
			),
		),
		goldmark.WithRendererOptions(
			// Enable unsafe HTML rendering to allow raw HTML like GitHub does.
			// This is safe for local-only wikis where all content is trusted.
			htmlrenderer.WithUnsafe(),
			htmlrenderer.WithXHTML(),
		),
	)

	return &Service{
		md:     md,
		logger: logger.With("component", "renderer"),
	}
}

// Render converts markdown content to HTML, caching results by path and modification time.
// If a cached entry exists with a matching modification time, it is returned immediately.
// Otherwise, the markdown is parsed and rendered, then cached for future requests.
// The path parameter is used for cache key generation and relative link resolution.
func (s *Service) Render(_ context.Context, path string, modTime time.Time, content []byte) (Document, error) {
	key := cacheKey(path)

	if entry, ok := s.cache.Load(key); ok {
		if cached, ok := entry.(cacheEntry); ok {
			if !cached.modTime.IsZero() && modTime.Equal(cached.modTime) {
				return cached.doc, nil
			}
		}
	}

	parserCtx := parser.NewContext()
	parserCtx.Set(docPathKey, path)
	buf := bytes.NewBuffer(nil)

	if err := s.md.Convert(content, buf, parser.WithContext(parserCtx)); err != nil {
		return Document{}, fmt.Errorf("render markdown: %w", err)
	}

	metadata := extractMetadata(parserCtx)
	doc := Document{
		HTML:     buf.String(),
		Metadata: metadata,
		Modified: modTime,
		Raw:      string(content),
	}

	s.cache.Store(key, cacheEntry{modTime: modTime, doc: doc})
	return doc, nil
}

// Invalidate removes the cached entry for the given path.
// This should be called when a document is updated or deleted to ensure
// the next Render call processes the latest content.
func (s *Service) Invalidate(path string) {
	s.cache.Delete(cacheKey(path))
}

func extractMetadata(ctx parser.Context) Metadata {
	raw := goldmarkmeta.Get(ctx)
	var meta Metadata
	if raw == nil {
		return meta
	}

	meta.Raw = make(map[string]any)
	for k, v := range raw {
		meta.Raw[k] = v
		switch k {
		case "title":
			if str, ok := toString(v); ok {
				meta.Title = str
			}
		case "description", "summary":
			if str, ok := toString(v); ok {
				meta.Description = str
			}
		case "tags", "keywords":
			meta.Tags = toStringSlice(v)
		}
	}

	if len(meta.Raw) == 0 {
		meta.Raw = nil
	}

	return meta
}

func toString(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case fmt.Stringer:
		return val.String(), true
	default:
		return "", false
	}
}

func toStringSlice(v any) []string {
	switch vv := v.(type) {
	case []any:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			if str, ok := toString(item); ok {
				out = append(out, str)
			}
		}
		return out
	case []string:
		return append([]string(nil), vv...)
	default:
		if str, ok := toString(v); ok {
			return []string{str}
		}
		return nil
	}
}
