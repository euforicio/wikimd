// Package content provides markdown file monitoring, caching, and change notification services.
package content

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/euforicio/wikimd/internal/content/tree"
	"github.com/euforicio/wikimd/internal/renderer"
)

const (
	eventTypeTreeUpdated = "treeUpdated"
	eventTypeDeleted     = "deleted"
	eventTypePageUpdated = "pageUpdated"
	eventTypeUnknown     = "unknown"
)

// Event describes change notifications emitted to subscribers.
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Path      string    `json:"path,omitempty"`
}

// Service coordinates content rendering, indexing, and change notifications.
type Service struct {
	ctx           context.Context
	logger        *slog.Logger
	watcher       *fsnotify.Watcher
	renderer      *renderer.Service
	cancel        context.CancelFunc
	tree          atomic.Pointer[tree.Node]
	subscribers   map[uint64]*subscriber
	root          string
	subCounter    atomic.Uint64
	subsMu        sync.RWMutex
	writeMu       sync.Mutex
	rebuildMu     sync.Mutex
	includeHidden bool
}

type subscriber struct {
	ctx context.Context
	ch  chan Event
}

// Options configures the content service.
type Options struct {
	IncludeHidden bool
}

// NewService initializes content monitoring rooted at path.
func NewService(parentCtx context.Context, root string, rendererSvc *renderer.Service, logger *slog.Logger, opts Options) (*Service, error) {
	if root == "" {
		return nil, errors.New("root directory must be provided")
	}
	if rendererSvc == nil {
		return nil, errors.New("renderer service must be provided")
	}
	if logger == nil {
		logger = slog.Default()
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	ctx, cancel := context.WithCancel(parentCtx)

	svc := &Service{
		root:          absRoot,
		renderer:      rendererSvc,
		includeHidden: opts.IncludeHidden,
		logger:        logger.With("component", "content_service"),
		ctx:           ctx,
		cancel:        cancel,
		subscribers:   make(map[uint64]*subscriber),
	}

	if err := svc.initTree(ctx); err != nil {
		cancel()
		return nil, err
	}

	if err := svc.startWatcher(); err != nil {
		cancel()
		return nil, err
	}

	return svc, nil
}

// Close releases resources associated with the service.
func (s *Service) Close() error {
	s.cancel()
	if s.watcher != nil {
		return s.watcher.Close()
	}
	return nil
}

// CurrentTree returns the cached tree snapshot.
func (s *Service) CurrentTree(ctx context.Context) (*tree.Node, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	n := s.tree.Load()
	if n == nil {
		return nil, errors.New("tree not initialized")
	}
	return n, nil
}

// Document loads and renders a markdown document by relative path.
func (s *Service) Document(ctx context.Context, relPath string) (renderer.Document, error) {
	if err := ctx.Err(); err != nil {
		return renderer.Document{}, err
	}

	rel, abs, err := s.resolveDocumentPath(relPath)
	if err != nil {
		return renderer.Document{}, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return renderer.Document{}, fmt.Errorf("stat document: %w", err)
	}
	if info.IsDir() {
		return renderer.Document{}, fmt.Errorf("path %s is a directory", rel)
	}

	content, err := os.ReadFile(abs) //nolint:gosec // abs is validated against root directory
	if err != nil {
		return renderer.Document{}, fmt.Errorf("read document: %w", err)
	}

	// Pass wiki-relative path (not absolute filesystem path) to renderer
	doc, err := s.renderer.Render(ctx, rel, info.ModTime(), content)
	if err != nil {
		return renderer.Document{}, err
	}
	return doc, nil
}

func (s *Service) resolveDocumentPath(relPath string) (string, string, error) {
	trimmed := strings.TrimSpace(relPath)
	if trimmed == "" {
		return "", "", fmt.Errorf("invalid path: %s", relPath)
	}
	clean := filepath.Clean(trimmed)
	if clean == "." || clean == "" {
		return "", "", fmt.Errorf("invalid path: %s", relPath)
	}
	if filepath.IsAbs(clean) {
		return "", "", fmt.Errorf("invalid path: %s", relPath)
	}
	if vol := filepath.VolumeName(clean); vol != "" {
		return "", "", fmt.Errorf("invalid path: %s", relPath)
	}

	clean = filepath.ToSlash(clean)
	if strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", "", fmt.Errorf("invalid path: %s", relPath)
	}

	// Add .md extension if not present
	if !strings.HasSuffix(clean, ".md") && !strings.HasSuffix(clean, ".markdown") {
		clean += ".md"
	}

	abs := filepath.Join(s.root, filepath.FromSlash(clean))
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", "", fmt.Errorf("resolve document path: %w", err)
	}
	relToRoot, err := filepath.Rel(s.root, abs)
	if err != nil {
		return "", "", fmt.Errorf("resolve document path: %w", err)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("resolved path escapes root: %s", relPath)
	}
	return clean, abs, nil
}

// Subscribe registers for change events. The returned channel will close when ctx is done.
func (s *Service) Subscribe(ctx context.Context) <-chan Event {
	ch := make(chan Event, 8)
	id := s.subCounter.Add(1)

	s.subsMu.Lock()
	s.subscribers[id] = &subscriber{ctx: ctx, ch: ch}
	s.subsMu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
		case <-s.ctx.Done():
		}

		s.subsMu.Lock()
		if sub, ok := s.subscribers[id]; ok {
			close(sub.ch)
			delete(s.subscribers, id)
		}
		s.subsMu.Unlock()
	}()

	return ch
}

func (s *Service) initTree(ctx context.Context) error {
	node, err := tree.Build(ctx, s.root, tree.Options{Renderer: s.renderer, IncludeHidden: s.includeHidden})
	if err != nil {
		return err
	}
	s.tree.Store(node)
	return nil
}

func (s *Service) startWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	s.watcher = watcher

	if err := s.watchRecursive(s.root); err != nil {
		return err
	}

	go s.runWatcher()
	return nil
}

func (s *Service) runWatcher() {
	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			s.handleEvent(event)
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			s.logger.Error("watcher error", slog.Any("err", err))
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) handleEvent(event fsnotify.Event) {
	if event.Name == "" {
		return
	}

	rel := s.relativePath(event.Name)
	op := event.Op

	s.logger.Debug("fsnotify event", slog.String("path", rel), slog.String("op", op.String()))

	isMarkdown := isMarkdownPath(event.Name)

	if isMarkdown && op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 {
		s.renderer.Invalidate(event.Name)
	}

	if op&fsnotify.Create != 0 {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			_ = s.watchRecursive(event.Name)
		}
	}

	eventType := classifyEvent(event.Name, op, isMarkdown)

	rebuildOK := s.rebuildTree()
	if !rebuildOK && (eventType == eventTypeTreeUpdated || eventType == eventTypeDeleted) {
		s.logger.Warn("skipping tree broadcast due to rebuild failure", slog.String("path", rel))
		return
	}

	s.broadcast(Event{Type: eventType, Path: rel, Timestamp: time.Now()})
}

func (s *Service) rebuildTree() bool {
	s.rebuildMu.Lock()
	defer s.rebuildMu.Unlock()

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	node, err := tree.Build(ctx, s.root, tree.Options{Renderer: s.renderer, IncludeHidden: s.includeHidden})
	if err != nil {
		s.logger.Error("rebuild tree failed", slog.Any("err", err))
		return false
	}
	s.tree.Store(node)
	return true
}

func (s *Service) broadcast(evt Event) {
	s.subsMu.RLock()
	var stale []uint64
	for id, sub := range s.subscribers {
		select {
		case <-sub.ctx.Done():
			stale = append(stale, id)
		case <-s.ctx.Done():
			stale = append(stale, id)
		case sub.ch <- evt:
		default:
			// drop event when subscriber lags
		}
	}
	s.subsMu.RUnlock()

	for _, id := range stale {
		s.removeSubscriber(id)
	}
}

func (s *Service) removeSubscriber(id uint64) {
	s.subsMu.Lock()
	if sub, ok := s.subscribers[id]; ok {
		close(sub.ch)
		delete(s.subscribers, id)
	}
	s.subsMu.Unlock()
}

func (s *Service) watchRecursive(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !s.includeHidden && strings.HasPrefix(d.Name(), ".") && path != s.root {
				return filepath.SkipDir
			}
			if err := s.watcher.Add(path); err != nil {
				s.logger.Warn("failed to watch directory", slog.String("path", path), slog.Any("err", err))
			}
		}
		return nil
	})
}

func (s *Service) relativePath(abs string) string {
	rel, err := filepath.Rel(s.root, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(rel)
}

// SaveDocument writes updated markdown contents to an existing document.
func (s *Service) SaveDocument(ctx context.Context, relPath string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rel, abs, err := s.resolveDocumentPath(relPath)
	if err != nil {
		return err
	}
	if !isMarkdownPath(rel) {
		return fmt.Errorf("updates allowed for markdown documents only: %s", rel)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if _, err := os.Stat(abs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("document not found: %s: %w", rel, os.ErrNotExist)
		}
		return fmt.Errorf("stat document: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil { //nolint:gosec // standard directory permissions
		return fmt.Errorf("ensure directory: %w", err)
	}

	if err := writeFileAtomic(abs, data); err != nil {
		return err
	}

	s.renderer.Invalidate(abs)
	return nil
}

// CreateDocument creates a new markdown document with the provided contents.
func (s *Service) CreateDocument(ctx context.Context, relPath string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rel, abs, err := s.resolveDocumentPath(relPath)
	if err != nil {
		return err
	}
	if !isMarkdownPath(rel) {
		return fmt.Errorf("only markdown documents are supported: %s", rel)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if _, err := os.Stat(abs); err == nil {
		return fmt.Errorf("document already exists: %s: %w", rel, os.ErrExist)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat document: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil { //nolint:gosec // standard directory permissions
		return fmt.Errorf("ensure directory: %w", err)
	}

	if err := writeFileAtomic(abs, data); err != nil {
		return err
	}

	s.renderer.Invalidate(abs)
	return nil
}

// RenameDocument renames an existing markdown document to a new path.
func (s *Service) RenameDocument(ctx context.Context, fromPath, toPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fromRel, fromAbs, err := s.resolveDocumentPath(fromPath)
	if err != nil {
		return err
	}
	toRel, toAbs, err := s.resolveDocumentPath(toPath)
	if err != nil {
		return err
	}
	if !isMarkdownPath(fromRel) || !isMarkdownPath(toRel) {
		return fmt.Errorf("rename supported for markdown documents only")
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if _, err := os.Stat(fromAbs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("source document not found: %s: %w", fromRel, os.ErrNotExist)
		}
		return fmt.Errorf("stat source document: %w", err)
	}

	if _, err := os.Stat(toAbs); err == nil {
		return fmt.Errorf("destination already exists: %s: %w", toRel, os.ErrExist)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat destination document: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(toAbs), 0o755); err != nil { //nolint:gosec // standard directory permissions
		return fmt.Errorf("ensure target directory: %w", err)
	}

	if err := os.Rename(fromAbs, toAbs); err != nil {
		return fmt.Errorf("rename document: %w", err)
	}

	s.renderer.Invalidate(fromAbs)
	s.renderer.Invalidate(toAbs)
	return nil
}

// DeleteDocument removes a markdown document from disk.
func (s *Service) DeleteDocument(ctx context.Context, relPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rel, abs, err := s.resolveDocumentPath(relPath)
	if err != nil {
		return err
	}
	if !isMarkdownPath(rel) {
		return fmt.Errorf("delete supported for markdown documents only: %s", rel)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if _, err := os.Stat(abs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("document not found: %s: %w", rel, os.ErrNotExist)
		}
		return fmt.Errorf("stat document: %w", err)
	}

	if err := os.Remove(abs); err != nil {
		return fmt.Errorf("delete document: %w", err)
	}

	s.renderer.Invalidate(abs)
	return nil
}

func writeFileAtomic(target string, data []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".wikimd-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("replace document: %w", err)
	}
	keep = true
	return nil
}

func classifyEvent(path string, op fsnotify.Op, isMarkdown bool) string {
	switch {
	case op&fsnotify.Remove != 0:
		if isMarkdown {
			if _, err := os.Stat(path); err == nil {
				return eventTypePageUpdated
			}
			return eventTypeDeleted
		}
		return eventTypeTreeUpdated
	case op&fsnotify.Rename != 0:
		return eventTypeTreeUpdated
	case op&(fsnotify.Write|fsnotify.Create) != 0:
		if isMarkdown {
			return eventTypePageUpdated
		}
		return eventTypeTreeUpdated
	default:
		return eventTypeUnknown
	}
}

// DebugStatus returns diagnostic information for testing.
func (s *Service) DebugStatus() map[string]any {
	res := map[string]any{
		"root":          s.root,
		"includeHidden": s.includeHidden,
	}
	if w := s.watcher; w != nil {
		res["watcher"] = map[string]any{
			"platform": runtime.GOOS,
		}
	}
	return res
}

func isMarkdownPath(path string) bool {
	name := strings.ToLower(path)
	return strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".markdown")
}
