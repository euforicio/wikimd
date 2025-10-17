// Package exporter generates static HTML sites from markdown content trees.
package exporter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/euforicio/wikimd/internal/content/tree"
	"github.com/euforicio/wikimd/internal/renderer"
	wikistatic "github.com/euforicio/wikimd/static"
)

const indexHTML = "index.html"

// Options configure the static export behavior.
type Options struct {
	Root                string
	OutputDir           string
	AssetsDir           string
	SiteTitle           string
	AssetPrefix         string
	BaseURL             string
	IncludeHidden       bool
	DarkModeFirst       bool
	GenerateSearchIndex bool
	CleanOutput         bool
}

// Exporter renders markdown content into a static HTML bundle.
type Exporter struct {
	renderer  *renderer.Service
	templates *templateRenderer
	logger    *slog.Logger
}

// New constructs an exporter instance ready for use.
func New(logger *slog.Logger) (*Exporter, error) {
	if logger == nil {
		logger = slog.Default()
	}

	tmpl, err := newTemplateRenderer()
	if err != nil {
		return nil, fmt.Errorf("load templates: %w", err)
	}

	return &Exporter{
		renderer:  renderer.NewService(logger),
		templates: tmpl,
		logger:    logger.With("component", "exporter"),
	}, nil
}

// Export walks the markdown tree rooted at opts.Root and writes a static site to opts.OutputDir.
//
//nolint:gocognit,gocyclo // export orchestration requires sequential steps and validation
func (e *Exporter) Export(ctx context.Context, opts Options) error {
	if strings.TrimSpace(opts.Root) == "" {
		return errors.New("root directory is required")
	}
	if strings.TrimSpace(opts.OutputDir) == "" {
		return errors.New("output directory is required")
	}
	if strings.TrimSpace(opts.AssetPrefix) == "" {
		opts.AssetPrefix = "assets"
	}
	if strings.TrimSpace(opts.SiteTitle) == "" {
		opts.SiteTitle = "wikimd"
	}

	rootDir, err := filepath.Abs(opts.Root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}
	outputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return fmt.Errorf("resolve output: %w", err)
	}
	assetsDir := opts.AssetsDir
	if assetsDir != "" {
		if assetsDir, err = filepath.Abs(assetsDir); err != nil {
			return fmt.Errorf("resolve assets: %w", err)
		}
	}

	if err := e.prepareOutputDir(outputDir, opts.CleanOutput); err != nil {
		return err
	}

	generatedAt := time.Now().UTC()

	treeRoot, err := tree.Build(ctx, rootDir, tree.Options{
		IncludeHidden: opts.IncludeHidden,
		Renderer:      e.renderer,
	})
	if err != nil {
		return fmt.Errorf("build content tree: %w", err)
	}

	docs := collectDocuments(treeRoot)
	sort.Slice(docs, func(i, j int) bool {
		return strings.Compare(strings.ToLower(docs[i].RelativePath), strings.ToLower(docs[j].RelativePath)) < 0
	})

	site := siteViewData{
		Title:         opts.SiteTitle,
		GeneratedAt:   generatedAt,
		Tree:          treeRoot,
		DarkModeFirst: opts.DarkModeFirst,
		BaseURL:       strings.TrimRight(opts.BaseURL, "/"),
	}

	treePayload := struct {
		GeneratedAt time.Time  `json:"generatedAt"`
		Root        *tree.Node `json:"root"`
	}{
		GeneratedAt: generatedAt,
		Root:        treeRoot,
	}
	if rawTree, err := json.Marshal(treePayload); err == nil {
		site.TreeJSON = template.JS(rawTree) //nolint:gosec // JSON from trusted source
	} else {
		e.logger.Warn("encode tree json failed", slog.Any("err", err))
	}

	assets := buildAssetRefs(opts.AssetPrefix)

	assetDest := filepath.Join(outputDir, filepath.FromSlash(opts.AssetPrefix))
	if err := e.copyAssetBundle(assetDest, assetsDir); err != nil {
		return err
	}

	var (
		defaultDoc  *tree.Node
		defaultPage layoutViewData
		searchIndex []searchEntry
	)

	for _, node := range docs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		absPath := filepath.Join(rootDir, filepath.FromSlash(node.RelativePath))
		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("stat %s: %w", node.RelativePath, err)
		}
		raw, err := os.ReadFile(absPath) //nolint:gosec // absPath constructed from validated root
		if err != nil {
			return fmt.Errorf("read %s: %w", node.RelativePath, err)
		}

		doc, err := e.renderer.Render(ctx, absPath, info.ModTime(), raw)
		if err != nil {
			return fmt.Errorf("render %s: %w", node.RelativePath, err)
		}

		page := pageViewData{
			Path:        node.RelativePath,
			Output:      toHTMLRel(node.RelativePath),
			URL:         toHTMLRel(node.RelativePath),
			Title:       firstNonEmpty(doc.Metadata.Title, node.Title, titleFromPath(node.RelativePath)),
			HTML:        template.HTML(doc.HTML), //nolint:gosec // HTML from trusted renderer
			Metadata:    doc.Metadata,
			Modified:    doc.Modified,
			Breadcrumbs: breadcrumbsFor(treeRoot, node.RelativePath),
		}

		if site.BaseURL != "" {
			if page.URL == indexHTML {
				page.Canonical = site.BaseURL
			} else {
				page.Canonical = fmt.Sprintf("%s/%s", site.BaseURL, page.URL)
			}
		}

		layout := layoutViewData{
			Site:        site,
			Page:        page,
			Active:      node.RelativePath,
			HasDocument: true,
			Assets:      assets,
		}

		if err := e.writePage(outputDir, layout); err != nil {
			return fmt.Errorf("write page %s: %w", node.RelativePath, err)
		}

		if defaultDoc == nil {
			defaultDoc = node
			defaultPage = layout
		}

		if opts.GenerateSearchIndex {
			searchIndex = append(searchIndex, searchEntry{
				Path:     page.URL,
				Source:   node.RelativePath,
				Title:    page.Title,
				Summary:  doc.Metadata.Description,
				Modified: doc.Modified,
				Raw:      string(raw),
			})
		}
	}

	if defaultDoc == nil {
		welcome := layoutViewData{
			Site:   site,
			Assets: assets,
		}
		welcome.Page.Title = site.Title
		welcome.Page.Path = ""
		welcome.Page.URL = indexHTML
		welcome.Page.HTML = template.HTML(`<div class="rounded-2xl border border-dashed border-slate-700 bg-slate-900/60 p-8 text-sm text-slate-400">No markdown documents were found in the export root. Add <code>.md</code> files under the root directory and rerun <code>wikimd-export</code>.</div>`)
		welcome.HasDocument = false
		if err := e.writeCustomPage(outputDir, indexHTML, welcome); err != nil {
			return fmt.Errorf("write welcome page: %w", err)
		}
	} else if rel := toHTMLRel(defaultDoc.RelativePath); rel != indexHTML {
		if err := e.writeCustomPage(outputDir, indexHTML, defaultPage); err != nil {
			return fmt.Errorf("write landing page: %w", err)
		}
	}

	if err := writeTreeJSON(outputDir, treePayload); err != nil {
		return err
	}

	if opts.GenerateSearchIndex {
		if err := writeSearchIndex(outputDir, generatedAt, searchIndex); err != nil {
			return err
		}
	}

	e.logger.Info("export complete",
		slog.Int("documents", len(docs)),
		slog.String("output", outputDir),
		slog.Duration("duration", time.Since(generatedAt)))

	return nil
}

func (e *Exporter) prepareOutputDir(output string, clean bool) error {
	if clean {
		if err := os.RemoveAll(output); err != nil {
			return fmt.Errorf("clean output: %w", err)
		}
	}
	return os.MkdirAll(output, 0o755) //nolint:gosec // standard directory permissions
}

func (e *Exporter) writePage(root string, data layoutViewData) error {
	return e.writeCustomPage(root, data.Page.Output, data)
}

func (e *Exporter) writeCustomPage(root, rel string, data layoutViewData) error {
	dest := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil { //nolint:gosec // standard directory permissions
		return err
	}
	buf := bytes.Buffer{}
	if err := e.templates.render(&buf, "layout", data); err != nil {
		return err
	}
	return os.WriteFile(dest, buf.Bytes(), 0o644) //nolint:gosec // standard file permissions
}

func (e *Exporter) copyAssetBundle(dest, override string) error {
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("reset assets dir: %w", err)
	}
	override = strings.TrimSpace(override)
	if override != "" {
		if info, err := os.Stat(override); err == nil && info.IsDir() {
			if err := copyAssets(override, dest); err != nil {
				return fmt.Errorf("copy override assets: %w", err)
			}
			e.logger.Debug("exporter using override assets", slog.String("source", override))
			return nil
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat assets override: %w", err)
		}
	}

	if err := wikistatic.CopyAll(dest); err != nil {
		return fmt.Errorf("copy embedded assets: %w", err)
	}
	return nil
}

func collectDocuments(root *tree.Node) []*tree.Node {
	var docs []*tree.Node
	var walk func(*tree.Node)
	walk = func(n *tree.Node) {
		if n == nil {
			return
		}
		if n.Type == tree.NodeTypeFile {
			docs = append(docs, n)
			return
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	return docs
}

func toHTMLRel(rel string) string {
	clean := strings.TrimSpace(rel)
	if clean == "" {
		return indexHTML
	}
	ext := filepath.Ext(clean)
	if ext != "" {
		clean = strings.TrimSuffix(clean, ext)
	}
	clean = strings.TrimSuffix(clean, "/")
	if clean == "" {
		return indexHTML
	}
	return clean + ".html"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func titleFromPath(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.ReplaceAll(base, "_", " ")
	parts := strings.Split(base, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

type breadcrumb struct {
	Title string
	URL   string
}

func breadcrumbsFor(root *tree.Node, target string) []breadcrumb {
	nodes := findNodePath(root, target)
	if len(nodes) <= 1 {
		return nil
	}
	nodes = nodes[1:]
	out := make([]breadcrumb, 0, len(nodes))
	for i, node := range nodes {
		title := firstNonEmpty(node.Title, titleFromPath(node.RelativePath))
		crumb := breadcrumb{Title: title}
		if node.Type == tree.NodeTypeFile {
			if i != len(nodes)-1 {
				crumb.URL = toHTMLRel(node.RelativePath)
			}
		} else {
			crumb.URL = ""
		}
		if i == len(nodes)-1 {
			crumb.URL = ""
		}
		out = append(out, crumb)
	}
	return out
}

func findNodePath(root *tree.Node, target string) []*tree.Node {
	if root == nil {
		return nil
	}
	if root.RelativePath == target {
		return []*tree.Node{root}
	}
	for _, child := range root.Children {
		if path := findNodePath(child, target); len(path) > 0 {
			return append([]*tree.Node{root}, path...)
		}
	}
	return nil
}

func buildAssetRefs(prefix string) assetRefs {
	clean := strings.Trim(prefix, "/")
	if clean == "" {
		clean = "assets"
	}
	join := func(parts ...string) string {
		return path.Join(append([]string{clean}, parts...)...)
	}
	vendor := func(name string) string {
		return join("vendor", name)
	}
	return assetRefs{
		CSSApp:    join("css", "app.css"),
		CSSChroma: vendor("chroma-github-dark.min.css"),
		JSApp:     join("js", "static-site.js"),
		JSMermaid: vendor("mermaid.min.js"),
	}
}

func copyAssets(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("assets directory %s does not exist", src)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("assets path %s is not a directory", src)
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755) //nolint:gosec // standard directory permissions
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { //nolint:gosec // standard directory permissions
			return err
		}
		data, err := os.ReadFile(path) //nolint:gosec // path from validated source directory
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644) //nolint:gosec // standard file permissions
	})
}

func writeTreeJSON(output string, payload any) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode tree json: %w", err)
	}
	dest := filepath.Join(output, "tree.json")
	if err := os.WriteFile(dest, raw, 0o644); err != nil { //nolint:gosec // standard file permissions
		return fmt.Errorf("write tree.json: %w", err)
	}
	return nil
}

type searchEntry struct {
	Path     string    `json:"path"`
	Source   string    `json:"source"`
	Title    string    `json:"title"`
	Summary  string    `json:"summary,omitempty"`
	Modified time.Time `json:"modified"`
	Raw      string    `json:"raw"`
}

func writeSearchIndex(output string, generatedAt time.Time, entries []searchEntry) error {
	payload := struct {
		GeneratedAt time.Time     `json:"generatedAt"`
		Entries     []searchEntry `json:"entries"`
	}{
		GeneratedAt: generatedAt,
		Entries:     entries,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode search index: %w", err)
	}
	dest := filepath.Join(output, "search.json")
	if err := os.WriteFile(dest, raw, 0o644); err != nil { //nolint:gosec // standard file permissions
		return fmt.Errorf("write search.json: %w", err)
	}
	return nil
}

//nolint:govet // field order optimized for readability, not memory
type layoutViewData struct {
	Page        pageViewData
	Site        siteViewData
	Assets      assetRefs
	Active      string
	HasDocument bool
}

type siteViewData struct {
	GeneratedAt   time.Time
	Tree          *tree.Node
	Title         string
	TreeJSON      template.JS
	BaseURL       string
	DarkModeFirst bool
}

type pageViewData struct {
	Metadata    renderer.Metadata
	Modified    time.Time
	Path        string
	Output      string
	URL         string
	Title       string
	HTML        template.HTML
	Canonical   string
	Breadcrumbs []breadcrumb
}

type assetRefs struct {
	CSSApp    string
	CSSChroma string
	JSApp     string
	JSMermaid string
}
