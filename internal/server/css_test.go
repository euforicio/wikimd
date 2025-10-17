package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/euforicio/wikimd/internal/config"
)

func TestValidateCSSPath(t *testing.T) {
	t.Parallel()
	// Create temp directory for test
	tmpDir := t.TempDir()
	wikimdDir := filepath.Join(tmpDir, ".wikimd")
	if err := os.MkdirAll(wikimdDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create valid CSS file
	validCSS := filepath.Join(wikimdDir, "custom.css")
	if err := os.WriteFile(validCSS, []byte(":root { --color: #000; }"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create invalid file (wrong extension)
	invalidExt := filepath.Join(wikimdDir, "custom.txt")
	if err := os.WriteFile(invalidExt, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create subdirectory to test path escaping
	subDir := filepath.Join(tmpDir, "outside")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	outsideCSS := filepath.Join(subDir, "evil.css")
	if err := os.WriteFile(outsideCSS, []byte(":root {}"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		cssPath    string
		allowedDir string
		want       bool // true if path should be valid
	}{
		{
			name:       "valid CSS file",
			cssPath:    validCSS,
			allowedDir: wikimdDir,
			want:       true,
		},
		{
			name:       "invalid extension",
			cssPath:    invalidExt,
			allowedDir: wikimdDir,
			want:       false,
		},
		{
			name:       "non-existent file",
			cssPath:    filepath.Join(wikimdDir, "nonexistent.css"),
			allowedDir: wikimdDir,
			want:       false,
		},
		{
			name:       "file outside allowed directory",
			cssPath:    outsideCSS,
			allowedDir: wikimdDir,
			want:       false,
		},
	}

	s := &Server{
		logger: testLogger(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := s.validateCSSPath(tt.cssPath, tt.allowedDir)
			got := result != ""
			if got != tt.want {
				t.Errorf("validateCSSPath() = %v, want %v (result path: %q, allowed: %q)", got, tt.want, result, tt.allowedDir)
			}
		})
	}
}

func TestHandleCustomCSS_Security(t *testing.T) {
	t.Parallel()
	// Create temp directory and CSS file
	tmpDir := t.TempDir()
	cssDir := filepath.Join(tmpDir, ".wikimd")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		t.Fatal(err)
	}

	cssContent := `:root { --color-bg: #000; }`
	cssFile := filepath.Join(cssDir, "custom.css")
	if err := os.WriteFile(cssFile, []byte(cssContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		index      string
		wantStatus int
	}{
		{
			name:       "valid index",
			index:      "0",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid index (negative)",
			index:      "-1",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid index (out of bounds)",
			index:      "999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid index (non-numeric)",
			index:      "abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	s := &Server{
		logger:         testLogger(),
		cfg:            config.Config{RootDir: tmpDir},
		customCSSPaths: []string{cssFile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", "/custom-theme/"+tt.index, nil)
			req.SetPathValue("index", tt.index)
			w := httptest.NewRecorder()

			s.handleCustomCSS(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("handleCustomCSS() status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandleCustomCSS_FileSize(t *testing.T) {
	t.Parallel()
	// Create temp directory
	tmpDir := t.TempDir()
	cssDir := filepath.Join(tmpDir, ".wikimd")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a large CSS file (2MB, exceeds 1MB limit)
	largeCSSFile := filepath.Join(cssDir, "large.css")
	largeContent := make([]byte, 2<<20) // 2MB
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	if err := os.WriteFile(largeCSSFile, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		logger:         testLogger(),
		cfg:            config.Config{RootDir: tmpDir},
		customCSSPaths: []string{largeCSSFile},
	}

	req := httptest.NewRequest("GET", "/custom-theme/0", nil)
	req.SetPathValue("index", "0")
	w := httptest.NewRecorder()

	s.handleCustomCSS(w, req)

	// Should reject file that's too large
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("handleCustomCSS() status = %v, want %v for large file", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestHandleCustomCSS_Caching(t *testing.T) {
	t.Parallel()
	// Create temp directory and CSS file
	tmpDir := t.TempDir()
	cssDir := filepath.Join(tmpDir, ".wikimd")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		t.Fatal(err)
	}

	cssFile := filepath.Join(cssDir, "custom.css")
	if err := os.WriteFile(cssFile, []byte(":root {}"), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		logger:         testLogger(),
		cfg:            config.Config{RootDir: tmpDir},
		customCSSPaths: []string{cssFile},
	}

	// First request
	req1 := httptest.NewRequest("GET", "/custom-theme/0", nil)
	req1.SetPathValue("index", "0")
	w1 := httptest.NewRecorder()
	s.handleCustomCSS(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("First request failed: %v", w1.Code)
	}

	// Check caching headers are present
	lastModified := w1.Header().Get("Last-Modified")
	if lastModified == "" {
		t.Error("Last-Modified header not set")
	}
	if w1.Header().Get("Cache-Control") == "" {
		t.Error("Cache-Control header not set")
	}

	// Sleep briefly to ensure timestamp comparison works
	// (file modification time has second precision on some systems)
	// Note: In production, the Last-Modified header ensures proper caching

	// Second request with If-Modified-Since from a future time (should return 304)
	req2 := httptest.NewRequest("GET", "/custom-theme/0", nil)
	req2.SetPathValue("index", "0")
	// Use the exact Last-Modified value from first response
	req2.Header.Set("If-Modified-Since", lastModified)
	w2 := httptest.NewRecorder()
	s.handleCustomCSS(w2, req2)

	// Should return 304 since file hasn't been modified after the given time
	if w2.Code != http.StatusNotModified {
		t.Errorf("Expected 304 Not Modified for unchanged file, got %v (Last-Modified: %s)", w2.Code, lastModified)
	}
}

// testLogger creates a no-op logger for testing
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}
