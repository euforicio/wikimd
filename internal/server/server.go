// Package server provides the HTTP server for the WikiMD web application.
package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/euforicio/wikimd/internal/config"
	"github.com/euforicio/wikimd/internal/content"
	"github.com/euforicio/wikimd/internal/content/tree"
	"github.com/euforicio/wikimd/internal/exporter"
	"github.com/euforicio/wikimd/internal/renderer"
	"github.com/euforicio/wikimd/internal/search"
	"github.com/euforicio/wikimd/static"
)

// Server wraps the HTTP server and configuration for serving markdown content over HTTP.
// It provides a RESTful API for reading, creating, updating, and deleting markdown documents,
// along with a web interface for browsing the wiki.
type Server struct { //nolint:govet // field order favors logical grouping over padding optimizations
	mux            *http.ServeMux
	httpServer     *http.Server
	logger         *slog.Logger
	content        *content.Service
	search         *search.Service
	exporter       *exporter.Exporter
	templates      *templateRenderer
	cfg            config.Config
	customCSSPaths []string // Resolved custom CSS file paths (global + per-repo)
}

var (
	errPathRequired        = errors.New("path is required")
	errInvalidPathEncoding = errors.New("invalid path encoding")
)

// New constructs a new Server with the provided configuration and services.
// It initializes the HTTP server, registers all routes and middleware,
// and prepares the server for starting via the Start method.
// Returns an error if template loading or exporter initialization fails.
func New(cfg config.Config, logger *slog.Logger, contentSvc *content.Service, searchSvc *search.Service) (*Server, error) {
	tmpl, err := newTemplateRenderer()
	if err != nil {
		return nil, fmt.Errorf("load templates: %w", err)
	}

	exp, err := exporter.New(logger)
	if err != nil {
		return nil, fmt.Errorf("init exporter: %w", err)
	}

	mux := http.NewServeMux()

	s := &Server{
		cfg:       cfg,
		mux:       mux,
		logger:    logger.With("component", "http"),
		content:   contentSvc,
		search:    searchSvc,
		exporter:  exp,
		templates: tmpl,
	}

	s.registerRoutes()
	s.discoverCustomCSS() // Discover custom theme CSS files

	return s, nil
}

func (s *Server) registerRoutes() {
	staticHandler := http.StripPrefix("/static/", http.FileServer(s.resolveStaticFS()))
	s.mux.Handle("GET /static/{path...}", staticHandler)
	s.mux.Handle("HEAD /static/{path...}", staticHandler)

	// Custom theme CSS endpoints
	s.mux.HandleFunc("GET /custom-theme/{index}", s.handleCustomCSS)

	// Media files (images, etc.) from wiki root
	s.mux.HandleFunc("GET /media/{path...}", s.handleMedia)

	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /page/{path...}", s.handlePageRoute)
	s.mux.HandleFunc("GET /", s.handleRoot)

	s.mux.HandleFunc("GET /api/tree", s.handleTree)
	s.mux.HandleFunc("POST /api/page", s.handleCreatePage)
	s.mux.HandleFunc("PUT /api/page/{path...}", s.handleSavePage)
	s.mux.HandleFunc("POST /api/page/rename", s.handleRenamePage)
	s.mux.HandleFunc("DELETE /api/page/{path...}", s.handleDeletePage)
	s.mux.HandleFunc("GET /api/page/{path...}", s.handlePage)
	s.mux.HandleFunc("GET /api/search", s.handleSearch)
	s.mux.HandleFunc("GET /api/export", s.handleExport)
	s.mux.HandleFunc("GET /events", s.handleEvents)
}

func (s *Server) resolveStaticFS() http.FileSystem {
	dir := strings.TrimSpace(s.cfg.AssetsDir)
	if dir != "" {
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() {
			s.logger.Debug("serving assets from filesystem", slog.String("dir", dir))
			return http.Dir(dir)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			s.logger.Warn("assets dir check failed", slog.String("dir", dir), slog.Any("err", err))
		}
	}
	s.logger.Debug("serving embedded assets")
	return static.HTTP()
}

// Start runs the HTTP server and optionally opens the browser.
// The server will listen on the configured port (or allocate a dynamic port if cfg.Port is 0).
// It supports graceful shutdown when the provided context is canceled.
// The method blocks until the server stops or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	// Build middleware chain
	handler := chain(s.mux,
		recoveryMiddleware,
		csrfMiddleware,
		gzipMiddleware,
		loggingMiddleware(s.logger, s.cfg.Verbose),
	)

	var errCh chan error

	var serverURL string

	if s.cfg.Port == 0 {
		// Dynamic port allocation
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("failed to allocate port: %w", err)
		}
		tcpAddr, ok := listener.Addr().(*net.TCPAddr)
		if !ok {
			return fmt.Errorf("unexpected listener address type")
		}
		serverURL = fmt.Sprintf("http://localhost:%d", tcpAddr.Port)

		s.httpServer = &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
		}

		errCh = make(chan error, 1)
		go func() {
			if _, err := fmt.Fprintf(os.Stdout, "WikiMD server listening on %s\n", serverURL); err != nil {
				s.logger.Warn("failed to announce server address", slog.String("url", serverURL), slog.Any("err", err))
			}
			errCh <- s.httpServer.Serve(listener)
		}()

		if s.cfg.AutoOpen {
			go s.openBrowserWhenReady(ctx, serverURL)
		}
	} else {
		// Fixed port mode (original behavior)
		addr := fmt.Sprintf(":%d", s.cfg.Port)
		serverURL = fmt.Sprintf("http://localhost:%d", s.cfg.Port)

		s.httpServer = &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
		}

		errCh = make(chan error, 1)
		go func() {
			if _, err := fmt.Fprintf(os.Stdout, "WikiMD server listening on %s\n", serverURL); err != nil {
				s.logger.Warn("failed to announce server address", slog.String("url", serverURL), slog.Any("err", err))
			}
			errCh <- s.httpServer.ListenAndServe()
		}()

		if s.cfg.AutoOpen {
			go s.openBrowserWhenReady(ctx, serverURL)
		}
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			s.logger.ErrorContext(ctx, "graceful shutdown failed", slog.Any("err", err))
			return err
		}
		return ctx.Err()
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// Shutdown gracefully stops the server with the provided context timeout.
// It waits for all active connections to close or the context to be canceled.
// Returns an error if the shutdown process fails or times out.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	root, err := s.content.CurrentTree(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "load content tree failed", slog.Any("err", err))
		http.Error(w, "failed to load content tree", http.StatusInternalServerError)
		return
	}

	// Check for legacy query parameter format and redirect
	if queryPage := strings.TrimSpace(r.URL.Query().Get("page")); queryPage != "" {
		http.Redirect(w, r, "/page/"+queryPage+r.URL.Fragment, http.StatusMovedPermanently)
		return
	}

	// Find the first document and redirect to it
	active := findFirstDocument(root)
	if active != "" {
		http.Redirect(w, r, "/page/"+active, http.StatusFound)
		return
	}

	// No documents found, render empty state
	data := homeViewData{
		Tree:            root,
		ActivePath:      "",
		Page:            pageViewData{},
		HasDocument:     false,
		CustomCSSURLs:   s.customCSSURLs(),
		SearchAvailable: s.search != nil,
	}

	s.renderTemplate(w, r, "layout", data)
}

func (s *Server) handlePageRoute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path, err := parseWildcardPath(r.PathValue("path"))
	if err != nil {
		s.respondPathError(w, err)
		return
	}

	root, err := s.content.CurrentTree(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "load content tree failed", slog.Any("err", err))
		http.Error(w, "failed to load content tree", http.StatusInternalServerError)
		return
	}

	var (
		page        pageViewData
		hasDocument bool
	)

	doc, err := s.content.Document(ctx, path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			s.logger.WarnContext(ctx, "page load failed", slog.Any("err", err), slog.String("path", path))
			http.Error(w, "failed to load page", http.StatusInternalServerError)
			return
		}
		// Document not found, but still render the layout with a not found message
		page = pageViewData{
			Path:    path,
			Title:   fmt.Sprintf("%s (missing)", titleFromPath(path)),
			Missing: true,
		}
	}
	if err == nil {
		page = s.pageViewFromDocument(ctx, root, path, doc)
		hasDocument = true
	}

	data := homeViewData{
		Tree:            root,
		ActivePath:      path,
		Page:            page,
		HasDocument:     hasDocument,
		CustomCSSURLs:   s.customCSSURLs(),
		SearchAvailable: s.search != nil,
	}

	s.renderTemplate(w, r, "layout", data)
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	node, err := s.content.CurrentTree(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "fetch tree failed", slog.Any("err", err))
		respondJSON(w, http.StatusInternalServerError, errorResponse("failed to load tree"))
		return
	}

	if isHTMXRequest(r) {
		active := r.URL.Query().Get("current")
		if active == "" {
			active = r.URL.Query().Get("page")
		}
		setHXTrigger(w, map[string]any{
			"treeUpdated": map[string]any{
				"active": active,
			},
		})
		s.renderTemplate(w, r, "tree", treeViewData{
			Root:   node,
			Active: active,
		})
		return
	}

	resp := struct {
		GeneratedAt time.Time  `json:"generatedAt"`
		Root        *tree.Node `json:"root"`
	}{
		GeneratedAt: time.Now(),
		Root:        node,
	}
	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path, err := parseWildcardPath(r.PathValue("path"))
	if err != nil {
		s.respondPathError(w, err)
		return
	}

	doc, err := s.content.Document(ctx, path)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		s.logger.WarnContext(ctx, "load page failed", slog.Any("err", err), slog.String("path", path))

		if isHTMXRequest(r) && errors.Is(err, os.ErrNotExist) {
			setHXTrigger(w, map[string]any{
				"pageLoaded": map[string]any{
					"path":    path,
					"missing": true,
				},
			})
			w.Header().Set("X-Wikimd-Path", path)
			missing := fmt.Sprintf("<div class=\"rounded-2xl border border-dashed border-red-500/50 bg-red-500/10 p-6 text-sm text-red-200\">Document <code>%s</code> was not found.</div>", template.HTMLEscapeString(path))
			s.renderTemplate(w, r, "page", pageViewData{
				Path:     path,
				Title:    fmt.Sprintf("%s (missing)", titleFromPath(path)),
				HTML:     template.HTML(missing), //nolint:gosec // HTML is safely escaped
				Metadata: renderer.Metadata{},
				Missing:  true,
			})
			return
		}
		respondJSON(w, status, errorResponse(err.Error()))
		return
	}

	if isHTMXRequest(r) {
		var root *tree.Node
		treeRoot, err := s.content.CurrentTree(ctx)
		if err != nil {
			s.logger.WarnContext(ctx, "refresh tree for breadcrumbs failed", slog.Any("err", err))
		}
		if err == nil {
			root = treeRoot
		}
		page := s.pageViewFromDocument(ctx, root, path, doc)
		setHXTrigger(w, map[string]any{
			"pageLoaded": map[string]any{
				"path":  path,
				"title": page.Title,
			},
		})
		w.Header().Set("X-Wikimd-Path", path)
		s.renderTemplate(w, r, "page", page)
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "raw" || format == "markdown" {
		//nolint:govet // inline struct field order optimized for readability
		resp := struct {
			Metadata renderer.Metadata `json:"metadata"`
			Modified time.Time         `json:"modified"`
			Path     string            `json:"path"`
			Raw      string            `json:"raw"`
		}{
			Path:     path,
			Raw:      doc.Raw,
			Metadata: doc.Metadata,
			Modified: doc.Modified,
		}
		respondJSON(w, http.StatusOK, resp)
		return
	}

	//nolint:govet // inline struct field order optimized for readability
	resp := struct {
		Metadata renderer.Metadata `json:"metadata"`
		Modified time.Time         `json:"modified"`
		Path     string            `json:"path"`
		HTML     string            `json:"html"`
	}{
		Path:     path,
		HTML:     doc.HTML,
		Metadata: doc.Metadata,
		Modified: doc.Modified,
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSavePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path, err := parseWildcardPath(r.PathValue("path"))
	if err != nil {
		s.respondPathError(w, err)
		return
	}

	var payload struct {
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		s.logger.WarnContext(ctx, "decode save payload failed", slog.Any("err", err), slog.String("path", path))
		respondJSON(w, http.StatusBadRequest, errorResponse("invalid JSON payload"))
		return
	}

	if err := s.content.SaveDocument(ctx, path, []byte(payload.Content)); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		s.logger.WarnContext(ctx, "save document failed", slog.Any("err", err), slog.String("path", path))
		respondJSON(w, status, errorResponse(err.Error()))
		return
	}

	resp := struct {
		Path    string `json:"path"`
		Message string `json:"message"`
	}{
		Path:    path,
		Message: "saved",
	}
	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCreatePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var payload struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		s.logger.WarnContext(ctx, "decode create payload failed", slog.Any("err", err))
		respondJSON(w, http.StatusBadRequest, errorResponse("invalid JSON payload"))
		return
	}

	path := strings.TrimSpace(payload.Path)
	if path == "" {
		respondJSON(w, http.StatusBadRequest, errorResponse("path is required"))
		return
	}

	if err := s.content.CreateDocument(ctx, path, []byte(payload.Content)); err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, os.ErrExist):
			status = http.StatusConflict
		case errors.Is(err, os.ErrNotExist):
			status = http.StatusBadRequest
		}
		s.logger.WarnContext(ctx, "create document failed", slog.Any("err", err), slog.String("path", path))
		respondJSON(w, status, errorResponse(err.Error()))
		return
	}

	resp := struct {
		Path    string `json:"path"`
		Message string `json:"message"`
	}{
		Path:    path,
		Message: "created",
	}
	respondJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleRenamePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var payload struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		s.logger.WarnContext(ctx, "decode rename payload failed", slog.Any("err", err))
		respondJSON(w, http.StatusBadRequest, errorResponse("invalid JSON payload"))
		return
	}

	from := strings.TrimSpace(payload.From)
	to := strings.TrimSpace(payload.To)
	if from == "" || to == "" {
		respondJSON(w, http.StatusBadRequest, errorResponse("from and to paths are required"))
		return
	}
	if from == to {
		respondJSON(w, http.StatusBadRequest, errorResponse("destination path must differ from source"))
		return
	}

	if err := s.content.RenameDocument(ctx, from, to); err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, os.ErrNotExist):
			status = http.StatusNotFound
		case errors.Is(err, os.ErrExist):
			status = http.StatusConflict
		}
		s.logger.WarnContext(ctx, "rename document failed", slog.Any("err", err), slog.String("from", from), slog.String("to", to))
		respondJSON(w, status, errorResponse(err.Error()))
		return
	}

	resp := struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Message string `json:"message"`
	}{
		From:    from,
		To:      to,
		Message: "renamed",
	}
	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDeletePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path, err := parseWildcardPath(r.PathValue("path"))
	if err != nil {
		s.respondPathError(w, err)
		return
	}

	if err := s.content.DeleteDocument(ctx, path); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		s.logger.WarnContext(ctx, "delete document failed", slog.Any("err", err), slog.String("path", path))
		respondJSON(w, status, errorResponse(err.Error()))
		return
	}

	resp := struct {
		Path    string `json:"path"`
		Message string `json:"message"`
	}{
		Path:    path,
		Message: "deleted",
	}
	respondJSON(w, http.StatusOK, resp)
}

func parseWildcardPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errPathRequired
	}
	decoded, err := url.PathUnescape(trimmed)
	if err != nil {
		return "", errInvalidPathEncoding
	}
	path := strings.TrimSpace(decoded)
	if path == "" {
		return "", errPathRequired
	}
	return path, nil
}

func (s *Server) respondPathError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errPathRequired):
		respondJSON(w, http.StatusBadRequest, errorResponse("path is required"))
	case errors.Is(err, errInvalidPathEncoding):
		respondJSON(w, http.StatusBadRequest, errorResponse("invalid path encoding"))
	default:
		respondJSON(w, http.StatusBadRequest, errorResponse(err.Error()))
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if s.search == nil {
		respondJSON(w, http.StatusServiceUnavailable, errorResponse("search not configured"))
		return
	}

	query := r.URL.Query().Get("q")
	if strings.TrimSpace(query) == "" {
		if isHTMXRequest(r) {
			setHXTrigger(w, map[string]any{
				"searchResults": map[string]any{"query": "", "count": 0},
			})
			s.renderTemplate(w, r, "search", searchViewData{})
			return
		}
		respondJSON(w, http.StatusBadRequest, errorResponse("query parameter 'q' is required"))
		return
	}

	var opts search.Options
	if v := r.URL.Query().Get("caseSensitive"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, errorResponse("invalid caseSensitive value"))
			return
		}
		opts.CaseSensitive = b
	}
	if v := r.URL.Query().Get("context"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			respondJSON(w, http.StatusBadRequest, errorResponse("invalid context value"))
			return
		}
		opts.Context = n
	}
	if v := r.URL.Query().Get("hidden"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, errorResponse("invalid hidden value"))
			return
		}
		opts.SearchHidden = b
	}

	params := r.URL.Query()
	if globs, ok := params["glob"]; ok {
		opts.IncludeGlobs = append(opts.IncludeGlobs, globs...)
	}
	if ex, ok := params["exclude"]; ok {
		opts.ExcludeGlobs = append(opts.ExcludeGlobs, ex...)
	}

	// Always search only markdown files
	opts.IncludeGlobs = append(opts.IncludeGlobs, "*.md", "*.markdown")

	results, err := s.search.Search(ctx, query, opts)
	if err != nil {
		s.logger.WarnContext(ctx, "search failed", slog.Any("err", err))
		respondJSON(w, http.StatusBadRequest, errorResponse(err.Error()))
		return
	}

	if isHTMXRequest(r) {
		data := searchViewData{
			Query:   query,
			Count:   len(results),
			Results: results,
			Options: opts,
		}
		setHXTrigger(w, map[string]any{
			"searchResults": map[string]any{
				"query": query,
				"count": len(results),
			},
		})
		s.renderTemplate(w, r, "search", data)
		return
	}

	resp := struct {
		Query   string          `json:"query"`
		Results []search.Result `json:"results"`
		Context search.Options  `json:"options"`
		Count   int             `json:"count"`
	}{
		Query:   query,
		Count:   len(results),
		Results: results,
		Context: opts,
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) renderTemplate(w http.ResponseWriter, r *http.Request, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := s.templates.render(w, name, data); err != nil {
		s.logger.ErrorContext(r.Context(), "render template failed", slog.Any("err", err), slog.String("template", name))
		http.Error(w, "failed to render template", http.StatusInternalServerError)
	}
}

func (s *Server) pageViewFromDocument(ctx context.Context, root *tree.Node, path string, doc renderer.Document) pageViewData {
	title := doc.Metadata.Title
	if title == "" {
		title = titleFromPath(path)
	}

	var crumbs []breadcrumb
	if root != nil {
		crumbs = breadcrumbsFor(root, path)
	}
	if root == nil {
		treeRoot, err := s.content.CurrentTree(ctx)
		if err != nil {
			s.logger.WarnContext(ctx, "load tree for breadcrumbs failed", slog.Any("err", err))
		}
		if err == nil {
			crumbs = breadcrumbsFor(treeRoot, path)
		}
	}

	return pageViewData{
		Path:        path,
		Title:       title,
		HTML:        template.HTML(doc.HTML), //nolint:gosec // HTML from trusted renderer
		Metadata:    doc.Metadata,
		Modified:    doc.Modified,
		Breadcrumbs: crumbs,
		Missing:     false,
	}
}

func titleFromPath(p string) string {
	name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.TrimSpace(name)
	if name == "" {
		return "Untitled Document"
	}
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		lower := strings.ToLower(w)
		words[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(words, " ")
}

func findFirstDocument(node *tree.Node) string {
	if node == nil {
		return ""
	}
	if node.Type == tree.NodeTypeFile {
		return node.RelativePath
	}
	for _, child := range node.Children {
		if path := findFirstDocument(child); path != "" {
			return path
		}
	}
	return ""
}

func breadcrumbsFor(root *tree.Node, target string) []breadcrumb {
	nodes := findNodePath(root, target)
	if len(nodes) <= 1 {
		return nil
	}
	nodes = nodes[1:]
	out := make([]breadcrumb, 0, len(nodes))
	for i, node := range nodes {
		title := node.Title
		if title == "" {
			title = titleFromPath(node.RelativePath)
		}
		crumb := breadcrumb{
			Title: title,
			Path:  "",
		}
		if node.Type == tree.NodeTypeFile && i != len(nodes)-1 {
			crumb.Path = node.RelativePath
		}
		if i == len(nodes)-1 {
			crumb.Path = ""
		}
		out = append(out, crumb)
	}
	return out
}

func findNodePath(root *tree.Node, target string) []*tree.Node {
	if root == nil {
		return nil
	}
	if strings.EqualFold(root.RelativePath, target) {
		return []*tree.Node{root}
	}
	for _, child := range root.Children {
		if path := findNodePath(child, target); len(path) > 0 {
			return append([]*tree.Node{root}, path...)
		}
	}
	return nil
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := s.content.Subscribe(ctx)

	if _, err := w.Write([]byte(": ready\n\n")); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			payload, err := encodeJSON(evt)
			if err != nil {
				s.logger.WarnContext(ctx, "encode sse event failed", slog.Any("err", err))
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse and validate path parameter
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		respondJSON(w, http.StatusBadRequest, errorResponse("path parameter is required"))
		return
	}

	// Validate path to prevent directory traversal attacks
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		s.logger.WarnContext(ctx, "invalid export path attempted", slog.String("path", path))
		respondJSON(w, http.StatusBadRequest, errorResponse("invalid path"))
		return
	}

	// Verify the resolved path is within the wiki root directory
	absPath := filepath.Join(s.cfg.RootDir, filepath.FromSlash(cleanPath))
	absRoot, err := filepath.Abs(s.cfg.RootDir)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to resolve root directory", slog.Any("err", err))
		respondJSON(w, http.StatusInternalServerError, errorResponse("internal server error"))
		return
	}

	absPath, err = filepath.Abs(absPath)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to resolve absolute path", slog.Any("err", err), slog.String("path", path))
		respondJSON(w, http.StatusBadRequest, errorResponse("invalid path"))
		return
	}

	// Ensure the resolved path is within the root directory
	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) && absPath != absRoot {
		s.logger.WarnContext(ctx, "path outside root directory attempted", slog.String("path", path), slog.String("resolved", absPath))
		respondJSON(w, http.StatusBadRequest, errorResponse("invalid path"))
		return
	}

	// Validate and normalize format parameter
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "html"
	}

	if !exporter.IsValidFormat(format) {
		respondJSON(w, http.StatusBadRequest, errorResponse("invalid format. Supported formats: html, pdf, markdown, txt"))
		return
	}

	// Check if the document exists (using the cleaned path)
	_, err = s.content.Document(ctx, cleanPath)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		s.logger.WarnContext(ctx, "export document not found", slog.Any("err", err), slog.String("path", cleanPath))
		respondJSON(w, status, errorResponse("document not found"))
		return
	}

	// Generate filename using the cleaned path
	filename := sanitizeFilename(cleanPath) + exporter.FileExtension(exporter.Format(format))

	// Set response headers
	w.Header().Set("Content-Type", exporter.ContentType(exporter.Format(format)))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)

	// Export the page using the cleaned path
	opts := exporter.ExportPageOptions{
		RootDir: s.cfg.RootDir,
		Path:    cleanPath,
		Format:  exporter.Format(format),
		Writer:  w,
	}

	if err := s.exporter.ExportPage(ctx, opts); err != nil {
		s.logger.ErrorContext(ctx, "export failed", slog.Any("err", err), slog.String("path", cleanPath), slog.String("format", format))
		// At this point headers are already sent, so we can't return a proper error response
		// The error will be logged and the response will be incomplete
	}
}

func sanitizeFilename(path string) string {
	// Remove directory separators and get base name
	name := filepath.Base(path)
	// Remove extension if present
	name = strings.TrimSuffix(name, filepath.Ext(name))
	// Replace unsafe characters
	name = strings.Map(func(r rune) rune {
		if r == ' ' {
			return '-'
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
	// Ensure it's not empty
	if name == "" {
		name = "export"
	}
	return name
}

func errorResponse(message string) map[string]string {
	return map[string]string{"error": message}
}

// discoverCustomCSS searches for custom theme CSS files in global and per-repo locations
// and validates paths for security (symlink resolution, directory traversal prevention)
func (s *Server) discoverCustomCSS() {
	var cssPaths []string

	// 1. Global custom CSS: ~/.wikimd/custom.css
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalCSS := filepath.Join(homeDir, ".wikimd", "custom.css")
		if validCSS := s.validateCSSPath(globalCSS, filepath.Join(homeDir, ".wikimd")); validCSS != "" {
			s.logger.Debug("found global custom CSS", slog.String("path", validCSS))
			cssPaths = append(cssPaths, validCSS)
		}
	}

	// 2. Per-repo custom CSS: <wiki-root>/.wikimd/custom.css
	if s.cfg.RootDir != "" {
		repoCSS := filepath.Join(s.cfg.RootDir, ".wikimd", "custom.css")
		if validCSS := s.validateCSSPath(repoCSS, filepath.Join(s.cfg.RootDir, ".wikimd")); validCSS != "" {
			s.logger.Debug("found repo custom CSS", slog.String("path", validCSS))
			cssPaths = append(cssPaths, validCSS)
		}
	}

	s.customCSSPaths = cssPaths
	if len(cssPaths) > 0 {
		s.logger.Info("custom CSS theming enabled", slog.Int("count", len(cssPaths)))
	}
}

// validateCSSPath validates a CSS file path for security:
// - File must exist and be a regular file
// - Must have .css extension
// - Symlinks are resolved and validated
// - Resolved path must be within allowed directory
// Returns the validated absolute path or empty string if invalid
func (s *Server) validateCSSPath(cssPath, allowedDir string) string {
	// Check if file exists
	if !fileExists(cssPath) {
		return ""
	}

	// Validate extension
	if filepath.Ext(cssPath) != ".css" {
		s.logger.Warn("invalid CSS file extension", slog.String("path", cssPath))
		return ""
	}

	// Get absolute path
	absPath, err := filepath.Abs(cssPath)
	if err != nil {
		s.logger.Warn("failed to resolve CSS path", slog.String("path", cssPath), slog.Any("err", err))
		return ""
	}

	// Resolve symlinks to prevent symlink attacks
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		s.logger.Warn("failed to resolve symlinks", slog.String("path", absPath), slog.Any("err", err))
		return ""
	}

	// Get absolute allowed directory and resolve its symlinks too
	absAllowedDir, err := filepath.Abs(allowedDir)
	if err != nil {
		s.logger.Warn("failed to resolve allowed directory", slog.String("dir", allowedDir), slog.Any("err", err))
		return ""
	}

	// Resolve symlinks in allowed directory for consistent comparison
	realAllowedDir, err := filepath.EvalSymlinks(absAllowedDir)
	if err != nil {
		// If symlink resolution fails, fall back to absolute path
		realAllowedDir = absAllowedDir
	}

	// Validate that resolved path is within allowed directory
	if !strings.HasPrefix(realPath, realAllowedDir+string(filepath.Separator)) &&
		realPath != realAllowedDir {
		s.logger.Warn("CSS path outside allowed directory",
			slog.String("path", cssPath),
			slog.String("resolved", realPath),
			slog.String("allowed", realAllowedDir))
		return ""
	}

	return realPath
}

// handleCustomCSS serves custom theme CSS files with security validation,
// file size limits, and HTTP caching support
func (s *Server) handleCustomCSS(w http.ResponseWriter, r *http.Request) {
	// Extract index from URL path parameter
	indexStr := r.PathValue("index")
	index := 0
	if n, err := fmt.Sscanf(indexStr, "%d", &index); err != nil || n != 1 {
		http.Error(w, "Invalid CSS index", http.StatusBadRequest)
		return
	}

	// Validate index bounds
	if index < 0 || index >= len(s.customCSSPaths) {
		http.Error(w, "CSS file not found", http.StatusNotFound)
		return
	}

	cssPath := s.customCSSPaths[index]

	// Security: Re-validate file extension
	if filepath.Ext(cssPath) != ".css" {
		s.logger.Warn("invalid CSS file extension", slog.String("path", cssPath))
		http.Error(w, "Invalid file type", http.StatusForbidden)
		return
	}

	// Get file info for size check and caching headers
	info, err := os.Stat(cssPath)
	if err != nil {
		s.logger.Warn("failed to stat custom CSS", slog.Any("err", err), slog.String("path", cssPath))
		http.Error(w, "CSS file not found", http.StatusNotFound)
		return
	}

	// Security: Check file size to prevent DoS (1MB limit for CSS files)
	const maxCSSSize = 1 << 20 // 1MB
	if info.Size() > maxCSSSize {
		s.logger.Warn("CSS file too large", slog.String("path", cssPath), slog.Int64("size", info.Size()))
		http.Error(w, "CSS file too large", http.StatusRequestEntityTooLarge)
		return
	}

	// HTTP caching: Check If-Modified-Since header
	// Truncate to seconds for HTTP date comparison (HTTP dates don't have sub-second precision)
	modTime := info.ModTime().UTC().Truncate(time.Second)
	if t, err := http.ParseTime(r.Header.Get("If-Modified-Since")); err == nil && !modTime.After(t) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Read CSS file
	content, err := os.ReadFile(cssPath) // #nosec G304 -- cssPath comes from validated configuration, not user input
	if err != nil {
		s.logger.Warn("failed to read custom CSS", slog.Any("err", err), slog.String("path", cssPath))
		http.Error(w, "Error reading CSS file", http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Last-Modified", modTime.Format(http.TimeFormat))
	// Short cache with must-revalidate for live editing support
	w.Header().Set("Cache-Control", "public, max-age=60, must-revalidate")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(content); err != nil {
		s.logger.ErrorContext(r.Context(), "failed to write CSS response", slog.Any("err", err), slog.String("path", cssPath))
	}
}

// handleMedia serves media files (images, etc.) from the wiki root directory
func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract and validate the path parameter
	rawPath, err := parseWildcardPath(r.PathValue("path"))
	if err != nil {
		s.respondPathError(w, err)
		return
	}

	// Clean and validate the path to prevent directory traversal
	cleanPath := filepath.Clean(rawPath)
	if strings.Contains(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		s.logger.WarnContext(ctx, "invalid media path attempted", slog.String("path", rawPath))
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Build absolute path within wiki root
	absRoot, err := filepath.Abs(s.cfg.RootDir)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to resolve root directory", slog.Any("err", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	absPath := filepath.Join(absRoot, filepath.FromSlash(cleanPath))
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to resolve media path", slog.Any("err", err), slog.String("path", rawPath))
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Ensure the resolved path is within the root directory
	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) && absPath != absRoot {
		s.logger.WarnContext(ctx, "media path outside root directory attempted",
			slog.String("path", rawPath),
			slog.String("resolved", absPath))
		http.Error(w, "Invalid path", http.StatusForbidden)
		return
	}

	// Check if file exists and is not a directory
	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		s.logger.WarnContext(ctx, "failed to stat media file", slog.Any("err", err), slog.String("path", rawPath))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if info.IsDir() {
		http.Error(w, "Path is a directory", http.StatusBadRequest)
		return
	}

	// Serve the file with appropriate content type
	http.ServeFile(w, r, absPath)
}

// fileExists checks if a file exists and is not a directory
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// customCSSURLs generates URL paths for custom CSS files
func (s *Server) customCSSURLs() []string {
	urls := make([]string, len(s.customCSSPaths))
	for i := range s.customCSSPaths {
		urls[i] = fmt.Sprintf("/custom-theme/%d", i)
	}
	return urls
}

func (s *Server) openBrowserWhenReady(ctx context.Context, url string) {
	timer := time.NewTimer(300 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		if err := openBrowser(ctx, url); err != nil {
			s.logger.WarnContext(ctx, "auto-open failed", slog.String("url", url), slog.Any("err", err))
		}
	}
}

func openBrowser(ctx context.Context, url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	}
	return cmd.Start()
}
