package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/euforicio/wikimd/internal/config"
	"github.com/euforicio/wikimd/internal/content"
	"github.com/euforicio/wikimd/internal/content/tree"
	"github.com/euforicio/wikimd/internal/renderer"
	"github.com/euforicio/wikimd/internal/search"
)

func TestAPIHandlers(t *testing.T) {
	t.Parallel()
	srv, cleanup := newTestServer(t)
	t.Cleanup(cleanup)

	t.Run("tree returns root snapshot", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/tree", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var resp struct {
			GeneratedAt time.Time  `json:"generatedAt"`
			Root        *tree.Node `json:"root"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Root == nil {
			t.Fatalf("expected root node, got nil")
		}
		if len(resp.Root.Children) == 0 {
			t.Fatalf("expected root to have children")
		}
		found := false
		for _, child := range resp.Root.Children {
			if child.RelativePath == "index.md" {
				found = true
				if child.Metadata == nil || child.Metadata.Title != "Welcome Home" {
					t.Fatalf("expected metadata title 'Welcome Home', got %#v", child.Metadata)
				}
				break
			}
		}
		if !found {
			t.Fatalf("expected to find index.md in tree; got %+v", resp.Root.Children)
		}
	})

	t.Run("tree returns HTML for HTMX requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/tree?current=index.md", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("HX-Trigger") == "" {
			t.Fatalf("expected HX-Trigger header")
		}
		if !strings.Contains(rec.Body.String(), "data-tree-path") {
			t.Fatalf("expected tree fragment")
		}
	})

	t.Run("page endpoint renders markdown", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/page/index.md", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Path string `json:"path"`
			HTML string `json:"html"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Path != "index.md" {
			t.Fatalf("expected path index.md, got %s", resp.Path)
		}
		if !strings.Contains(resp.HTML, "<h1 id=\"welcome\">Welcome") {
			t.Fatalf("expected rendered HTML to contain heading, got %q", resp.HTML)
		}
	})

	t.Run("page endpoint returns raw markdown when requested", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/page/index.md?format=raw", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Path string `json:"path"`
			Raw  string `json:"raw"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Path != "index.md" {
			t.Fatalf("expected path index.md, got %s", resp.Path)
		}
		if !strings.Contains(resp.Raw, "# Welcome") {
			t.Fatalf("expected raw markdown content, got %q", resp.Raw)
		}
	})

	t.Run("page endpoint supports percent-encoded paths", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/page/guides%2Fgetting_started.md?format=raw", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d with body %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Path string `json:"path"`
			Raw  string `json:"raw"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Path != "guides/getting_started.md" {
			t.Fatalf("expected decoded path, got %s", resp.Path)
		}
		if !strings.Contains(resp.Raw, "Getting Started") {
			t.Fatalf("expected raw markdown to contain heading, got %q", resp.Raw)
		}
	})

	t.Run("page endpoint returns HTML for HTMX requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/page/index.md", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if trigger := rec.Header().Get("HX-Trigger"); trigger == "" {
			t.Fatalf("expected HX-Trigger header, got empty")
		}
		body := rec.Body.String()
		if !strings.Contains(body, "id=\"page-view\"") {
			t.Fatalf("expected page fragment, got %q", body)
		}
	})

	t.Run("page endpoint handles missing documents", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/page/missing.md", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for missing document, got %d", rec.Code)
		}
	})

	t.Run("save endpoint persists updated markdown", func(t *testing.T) {
		target := filepath.Join(srv.cfg.RootDir, "index.md")
		original, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read original document failed: %v", err)
		}
		t.Cleanup(func() {
			_ = os.WriteFile(target, original, 0o644)
		})

		payload := `{"content":"# Welcome Updated\n\nContent updated during test.\n"}`
		req := httptest.NewRequest(http.MethodPut, "/api/page/index.md", strings.NewReader(payload))
		req.Host = "localhost:8080" // Set Host for CSRF validation
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:8080") // CSRF protection requires Origin
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", rec.Code, rec.Body.String())
		}

		data, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read updated document failed: %v", err)
		}
		if !strings.Contains(string(data), "Welcome Updated") {
			t.Fatalf("expected updated content in document, got %q", string(data))
		}
	})

	t.Run("create endpoint writes new document", func(t *testing.T) {
		payload := `{"path":"notes/new-page.md","content":"# Draft Note\n"}`
		req := httptest.NewRequest(http.MethodPost, "/api/page", strings.NewReader(payload))
		req.Host = "localhost:8080" // Set Host for CSRF validation
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:8080") // CSRF protection requires Origin
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d with body %s", rec.Code, rec.Body.String())
		}

		target := filepath.Join(srv.cfg.RootDir, "notes", "new-page.md")
		t.Cleanup(func() {
			_ = os.Remove(target)
		})
		data, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read created document failed: %v", err)
		}
		if !strings.Contains(string(data), "# Draft Note") {
			t.Fatalf("expected created document to contain draft content, got %q", string(data))
		}
	})

	t.Run("rename endpoint moves document", func(t *testing.T) {
		payload := `{"from":"guides/getting_started.md","to":"guides/getting_started_v2.md"}`
		req := httptest.NewRequest(http.MethodPost, "/api/page/rename", strings.NewReader(payload))
		req.Host = "localhost:8080" // Set Host for CSRF validation
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:8080") // CSRF protection requires Origin
		rec := httptest.NewRecorder()

		original := filepath.Join(srv.cfg.RootDir, "guides", "getting_started.md")
		renamed := filepath.Join(srv.cfg.RootDir, "guides", "getting_started_v2.md")
		t.Cleanup(func() {
			if _, err := os.Stat(renamed); err == nil {
				_ = os.Rename(renamed, original)
			}
		})

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", rec.Code, rec.Body.String())
		}

		if _, err := os.Stat(original); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected original document to be removed, got err=%v", err)
		}

		data, err := os.ReadFile(renamed)
		if err != nil {
			t.Fatalf("read renamed document failed: %v", err)
		}
		if !strings.Contains(string(data), "Getting Started") {
			t.Fatalf("expected renamed document to retain content, got %q", string(data))
		}
	})

	t.Run("delete endpoint removes document", func(t *testing.T) {
		target := filepath.Join(srv.cfg.RootDir, "notes", "temp-delete.md")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("create directory failed: %v", err)
		}
		if err := os.WriteFile(target, []byte("# Temporary\n"), 0o644); err != nil {
			t.Fatalf("prepare document failed: %v", err)
		}
		req := httptest.NewRequest(http.MethodDelete, "/api/page/notes/temp-delete.md", nil)
		req.Host = "localhost:8080"                       // Set Host for CSRF validation
		req.Header.Set("Origin", "http://localhost:8080") // CSRF protection requires Origin
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", rec.Code, rec.Body.String())
		}

		if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected document to be deleted, got err=%v", err)
		}
	})

	t.Run("search endpoint returns ripgrep results", func(t *testing.T) {
		if _, err := exec.LookPath("rg"); err != nil {
			t.Skip("rg not installed")
		}

		req := httptest.NewRequest(http.MethodGet, "/api/search?q=Welcome", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Results []search.Result `json:"results"`
			Count   int             `json:"count"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Count == 0 {
			t.Fatalf("expected at least one search result")
		}
	})

	t.Run("search endpoint returns HTML for HTMX requests", func(t *testing.T) {
		if _, err := exec.LookPath("rg"); err != nil {
			t.Skip("rg not installed")
		}
		req := httptest.NewRequest(http.MethodGet, "/api/search?q=Welcome", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("HX-Trigger") == "" {
			t.Fatalf("expected HX-Trigger header")
		}
		if !strings.Contains(rec.Body.String(), "Results for") {
			t.Fatalf("expected rendered search fragment")
		}
	})
}

func TestRootHandlerRendersLayout(t *testing.T) {
	t.Parallel()
	srv, cleanup := newTestServer(t)
	t.Cleanup(cleanup)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	// Root handler redirects to /index.md
	if rec.Code != http.StatusFound && rec.Code != http.StatusMovedPermanently && rec.Code != http.StatusSeeOther {
		// If not a redirect, should return 200 with HTML
		if rec.Code != http.StatusOK {
			t.Fatalf("expected redirect or 200, got %d", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "<html") {
			t.Fatalf("expected HTML document, got: %s", body)
		}
		if !strings.Contains(body, "page-region") {
			t.Fatalf("expected rendered layout with page region")
		}
	}
}

func TestEventsHandlerSendsReadyComment(t *testing.T) {
	t.Parallel()
	srv, cleanup := newTestServer(t)
	t.Cleanup(cleanup)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	errCh := make(chan error, 1)
	go func() {
		srv.handleEvents(rec, req)
		errCh <- nil
	}()

	// Give the handler a moment to write the ready comment.
	time.Sleep(150 * time.Millisecond)
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("handleEvents returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, ": ready\n\n") {
		t.Fatalf("expected ready comment in body, got %q", body)
	}
}

func TestExportHandlerSecurity(t *testing.T) {
	t.Parallel()
	srv, cleanup := newTestServer(t)
	t.Cleanup(cleanup)

	t.Run("blocks path traversal with ..", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/export?path=../etc/passwd&format=html", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !strings.Contains(resp["error"], "invalid path") {
			t.Errorf("expected 'invalid path' error, got %q", resp["error"])
		}
	})

	t.Run("blocks path traversal with multiple ..", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/export?path=../../etc/passwd&format=html", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !strings.Contains(resp["error"], "invalid path") {
			t.Errorf("expected 'invalid path' error, got %q", resp["error"])
		}
	})

	t.Run("blocks absolute paths", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/export?path=/etc/passwd&format=html", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		// On Windows, /etc/passwd is treated as relative path, so it may return 404
		// rather than 400. Both are acceptable rejections.
		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
			t.Errorf("expected status 400 or 404, got %d", rec.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !strings.Contains(resp["error"], "invalid path") && !strings.Contains(resp["error"], "document not found") {
			t.Errorf("expected 'invalid path' or 'document not found' error, got %q", resp["error"])
		}
	})

	t.Run("blocks path with .. in middle", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/export?path=subdir/../../../etc/passwd&format=html", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !strings.Contains(resp["error"], "invalid path") {
			t.Errorf("expected 'invalid path' error, got %q", resp["error"])
		}
	})

	t.Run("rejects invalid format", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/export?path=index.md&format=invalid", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !strings.Contains(resp["error"], "invalid format") {
			t.Errorf("expected 'invalid format' error, got %q", resp["error"])
		}
	})

	t.Run("accepts valid path and format", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/export?path=index.md&format=html", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d with body: %s", rec.Code, rec.Body.String())
		}

		// Check content type
		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Errorf("expected content-type text/html, got %s", contentType)
		}

		// Check content disposition
		disposition := rec.Header().Get("Content-Disposition")
		if !strings.Contains(disposition, "attachment") {
			t.Errorf("expected attachment disposition, got %s", disposition)
		}
		if !strings.Contains(disposition, "index.html") {
			t.Errorf("expected filename index.html, got %s", disposition)
		}
	})

	t.Run("accepts valid nested path", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/export?path=guides/getting_started.md&format=markdown", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d with body: %s", rec.Code, rec.Body.String())
		}

		// Check content type
		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/markdown") {
			t.Errorf("expected content-type text/markdown, got %s", contentType)
		}
	})

	t.Run("requires path parameter", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/export?format=html", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !strings.Contains(resp["error"], "path parameter is required") {
			t.Errorf("expected 'path parameter is required' error, got %q", resp["error"])
		}
	})

	t.Run("defaults to html format", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/export?path=index.md", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		// Check content type for default HTML format
		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Errorf("expected default content-type text/html, got %s", contentType)
		}
	})
}

func newTestServer(t *testing.T) (*testServer, func()) {
	t.Helper()

	tempRoot := t.TempDir()
	copyDir(t, filepath.Join("..", "..", "testdata", "wiki"), tempRoot)

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	renderSvc := renderer.NewService(logger)

	contentSvc, err := content.NewService(context.Background(), tempRoot, renderSvc, logger, content.Options{})
	if err != nil {
		t.Fatalf("content service init failed: %v", err)
	}

	searchSvc, err := search.NewService(tempRoot, logger)
	if err != nil {
		contentSvc.Close()
		t.Fatalf("search service init failed: %v", err)
	}

	cfg := config.Default()
	cfg.RootDir = tempRoot
	cfg.AutoOpen = false
	cfg.AssetsDir = filepath.Join("..", "..", "static")

	srv, err := New(cfg, logger, contentSvc, searchSvc)
	if err != nil {
		contentSvc.Close()
		t.Fatalf("server init failed: %v", err)
	}

	// Build middleware chain for testing
	handler := chain(srv.mux,
		recoveryMiddleware,
		csrfMiddleware,
		gzipMiddleware,
		loggingMiddleware(srv.logger, cfg.Verbose),
	)

	cleanup := func() {
		_ = contentSvc.Close()
	}
	return &testServer{Server: srv, handler: handler}, cleanup
}

// testServer wraps Server with a handler for testing.
type testServer struct {
	*Server
	handler http.Handler
}

// ServeHTTP delegates to the handler with middleware.
func (ts *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts.handler.ServeHTTP(w, r)
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	if err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}
}
