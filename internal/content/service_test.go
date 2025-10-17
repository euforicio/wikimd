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
