package content_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/euforicio/wikimd/internal/content"
	"github.com/euforicio/wikimd/internal/renderer"
)

func TestServiceEmitsEventsOnFileChange(t *testing.T) {
	t.Parallel()
	src := filepath.Join("..", "..", "testdata", "wiki")
	dst := t.TempDir()
	copyDir(t, src, dst)

	renderSvc := renderer.NewService(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn})))

	ctx, cancel := context.WithCancel(context.Background())
	svc, err := content.NewService(ctx, dst, renderSvc, slog.New(slog.NewTextHandler(io.Discard, nil)), content.Options{})
	if err != nil {
		cancel()
		t.Fatalf("NewService failed: %v", err)
	}
	t.Cleanup(func() {
		svc.Close()
		cancel()
	})

	if _, err := svc.CurrentTree(context.Background()); err != nil {
		t.Fatalf("CurrentTree error: %v", err)
	}

	subCtx, subCancel := context.WithCancel(context.Background())
	ch := svc.Subscribe(subCtx)
	t.Cleanup(subCancel)

	indexPath := filepath.Join(dst, "index.md")
	contentBytes := []byte("---\ntitle: Welcome Home\n---\n\n# Updated\n")

	// Give the watcher time to attach.
	time.Sleep(200 * time.Millisecond)

	if err := os.WriteFile(indexPath, contentBytes, 0o644); err != nil {
		t.Fatalf("failed to write test document: %v", err)
	}

	timeout := time.After(2 * time.Second)
	for {
		select {
		case evt := <-ch:
			if evt.Type == "pageUpdated" && evt.Path == "index.md" {
				return
			}
		case <-timeout:
			t.Fatalf("did not receive expected pageUpdated event")
		}
	}
}

func TestDocumentLoadsAndRendersMarkdown(t *testing.T) {
	t.Parallel()

	src := filepath.Join("..", "..", "testdata", "wiki")
	dst := t.TempDir()
	copyDir(t, src, dst)

	renderSvc := renderer.NewService(slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()

	svc, err := content.NewService(ctx, dst, renderSvc, slog.New(slog.NewTextHandler(io.Discard, nil)), content.Options{})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	t.Cleanup(func() { svc.Close() })

	tests := []struct {
		name, path, errSubstr string
		wantErr               bool
	}{
		{name: "valid document", path: "index.md", wantErr: false},
		{name: "valid document without extension", path: "index", wantErr: false},
		{name: "non-existent document", path: "nonexistent.md", wantErr: true, errSubstr: "stat document"},
		{name: "path traversal with ..", path: "../outside.md", wantErr: true, errSubstr: "invalid path"},
		{name: "path traversal in middle", path: "subdir/../../../etc/passwd", wantErr: true, errSubstr: "invalid path"},
		{name: "absolute path", path: "/etc/passwd", wantErr: true, errSubstr: "invalid path"},
		{name: "empty path", path: "", wantErr: true, errSubstr: "invalid path"},
		{name: "dot path", path: ".", wantErr: true, errSubstr: "invalid path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := svc.Document(ctx, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !containsSubstring(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if doc.HTML == "" {
				t.Error("expected non-empty HTML")
			}
		})
	}
}

func TestSaveDocument(t *testing.T) {
	t.Parallel()

	src := filepath.Join("..", "..", "testdata", "wiki")
	dst := t.TempDir()
	copyDir(t, src, dst)

	renderSvc := renderer.NewService(slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()

	svc, err := content.NewService(ctx, dst, renderSvc, slog.New(slog.NewTextHandler(io.Discard, nil)), content.Options{})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	t.Cleanup(func() { svc.Close() })

	tests := []struct {
		name, path, content, errSubstr string
		wantErr                        bool
	}{
		{name: "save existing document", path: "index.md", content: "# Updated\n", wantErr: false},
		{name: "save non-existent document", path: "nonexistent.md", content: "# New\n", wantErr: true, errSubstr: "not found"},
		{name: "path traversal", path: "../outside.md", content: "malicious", wantErr: true, errSubstr: "invalid path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.SaveDocument(ctx, tt.path, []byte(tt.content))
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !containsSubstring(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify content was saved
			data, err := os.ReadFile(filepath.Join(dst, tt.path))
			if err != nil {
				t.Fatalf("failed to read saved file: %v", err)
			}
			if string(data) != tt.content {
				t.Errorf("saved content mismatch: got %q, want %q", string(data), tt.content)
			}
		})
	}
}

func TestCreateDocument(t *testing.T) {
	t.Parallel()

	src := filepath.Join("..", "..", "testdata", "wiki")
	dst := t.TempDir()
	copyDir(t, src, dst)

	renderSvc := renderer.NewService(slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()

	svc, err := content.NewService(ctx, dst, renderSvc, slog.New(slog.NewTextHandler(io.Discard, nil)), content.Options{})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	t.Cleanup(func() { svc.Close() })

	tests := []struct {
		name, path, content, errSubstr string
		wantErr                        bool
	}{
		{name: "create new document", path: "newdoc.md", content: "# New Document\n", wantErr: false},
		{name: "create in subdirectory", path: "subdir/nested.md", content: "# Nested\n", wantErr: false},
		{name: "create existing document", path: "index.md", content: "# Conflict\n", wantErr: true, errSubstr: "already exists"},
		{name: "path traversal", path: "../outside.md", content: "malicious", wantErr: true, errSubstr: "invalid path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.CreateDocument(ctx, tt.path, []byte(tt.content))
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !containsSubstring(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify file was created
			absPath := filepath.Join(dst, tt.path)
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Errorf("file was not created at %s", absPath)
			}
		})
	}
}

func TestDeleteDocument(t *testing.T) {
	t.Parallel()

	src := filepath.Join("..", "..", "testdata", "wiki")
	dst := t.TempDir()
	copyDir(t, src, dst)

	renderSvc := renderer.NewService(slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()

	svc, err := content.NewService(ctx, dst, renderSvc, slog.New(slog.NewTextHandler(io.Discard, nil)), content.Options{})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	t.Cleanup(func() { svc.Close() })

	// Create a document to delete
	testFile := filepath.Join(dst, "todelete.md")
	if err := os.WriteFile(testFile, []byte("# Delete Me\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name, path, errSubstr string
		wantErr               bool
	}{
		{name: "delete existing document", path: "todelete.md", wantErr: false},
		{name: "delete non-existent document", path: "nonexistent.md", wantErr: true, errSubstr: "not found"},
		{name: "path traversal", path: "../outside.md", wantErr: true, errSubstr: "invalid path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.DeleteDocument(ctx, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !containsSubstring(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify file was deleted
			absPath := filepath.Join(dst, tt.path)
			if _, err := os.Stat(absPath); !os.IsNotExist(err) {
				t.Errorf("file was not deleted at %s", absPath)
			}
		})
	}
}

func TestRenameDocument(t *testing.T) {
	t.Parallel()

	src := filepath.Join("..", "..", "testdata", "wiki")
	dst := t.TempDir()
	copyDir(t, src, dst)

	renderSvc := renderer.NewService(slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()

	svc, err := content.NewService(ctx, dst, renderSvc, slog.New(slog.NewTextHandler(io.Discard, nil)), content.Options{})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	t.Cleanup(func() { svc.Close() })

	// Create a document to rename
	testFile := filepath.Join(dst, "torename.md")
	if err := os.WriteFile(testFile, []byte("# Rename Me\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name, from, to, errSubstr string
		wantErr                   bool
	}{
		{name: "rename document", from: "torename.md", to: "renamed.md", wantErr: false},
		{name: "rename to existing", from: "renamed.md", to: "index.md", wantErr: true, errSubstr: "already exists"},
		{name: "rename non-existent", from: "nonexistent.md", to: "new.md", wantErr: true, errSubstr: "not found"},
		{name: "path traversal in source", from: "../outside.md", to: "safe.md", wantErr: true, errSubstr: "invalid path"},
		{name: "path traversal in dest", from: "index.md", to: "../outside.md", wantErr: true, errSubstr: "invalid path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.RenameDocument(ctx, tt.from, tt.to)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !containsSubstring(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
