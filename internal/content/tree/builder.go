// Package tree builds hierarchical navigation trees from markdown file directories.
package tree

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/euforicio/wikimd/internal/renderer"
)

// NodeType identifies what a tree node represents.
type NodeType string

// Node type constants for directory and file entries.
const (
	NodeTypeDirectory NodeType = "directory"
	NodeTypeFile      NodeType = "file"
)

// Node represents a navigation entry (directory or markdown file).
type Node struct {
	Modified     time.Time          `json:"modified"`
	Metadata     *renderer.Metadata `json:"metadata,omitempty"`
	Name         string             `json:"name"`
	RawName      string             `json:"rawName"`
	RelativePath string             `json:"relativePath"`
	Slug         string             `json:"slug"`
	Type         NodeType           `json:"type"`
	Title        string             `json:"title"`
	Children     []*Node            `json:"children,omitempty"`
	Size         int64              `json:"size"`
}

// Options control how the tree is constructed.
type Options struct {
	Renderer      *renderer.Service
	ExcludeDirs   []string
	IncludeHidden bool
}

// Build walks the root directory and returns a tree of markdown content.
func Build(ctx context.Context, root string, opts Options) (*Node, error) {
	if root == "" {
		return nil, errors.New("root directory must be provided")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root %s is not a directory", absRoot)
	}

	b := newBuilder(absRoot, opts)

	return b.buildDir(ctx, absRoot, "")
}

// builder carries state during tree construction.
type builder struct {
	exclude map[string]struct{}
	root    string
	opts    Options
}

var defaultExcludedDirs = []string{
	"node_modules",
	"vendor",
	"venv",
	".venv",
	"deps",
	"third_party",
	".git",
	".hg",
	".svn",
	".idea",
	".vscode",
	"__pycache__",
}

func newBuilder(absRoot string, opts Options) *builder {
	exclude := make(map[string]struct{})
	for _, name := range defaultExcludedDirs {
		if name = strings.TrimSpace(name); name != "" {
			exclude[strings.ToLower(name)] = struct{}{}
		}
	}
	for _, name := range opts.ExcludeDirs {
		if name = strings.TrimSpace(name); name != "" {
			exclude[strings.ToLower(name)] = struct{}{}
		}
	}
	return &builder{
		root:    absRoot,
		opts:    opts,
		exclude: exclude,
	}
}

func (b *builder) isExcluded(name string) bool {
	_, ok := b.exclude[strings.ToLower(name)]
	return ok
}

//nolint:gocognit,gocyclo // directory traversal naturally requires multiple decision points
func (b *builder) buildDir(ctx context.Context, absPath, relPath string) (*Node, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", absPath, err)
	}

	children := make([]*Node, 0, len(entries))
	for _, entry := range entries {
		if !b.opts.IncludeHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		childRel := filepath.Join(relPath, entry.Name())
		childAbs := filepath.Join(absPath, entry.Name())

		if entry.IsDir() {
			if b.isExcluded(entry.Name()) {
				continue
			}
			childNode, err := b.buildDir(ctx, childAbs, childRel)
			if err != nil {
				return nil, err
			}
			if childNode == nil {
				continue
			}
			children = append(children, childNode)
			continue
		}

		if !isMarkdown(entry) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat file %s: %w", childAbs, err)
		}

		node, err := b.buildFileNode(ctx, childAbs, childRel, info)
		if err != nil {
			return nil, err
		}
		children = append(children, node)
	}

	if len(children) == 0 && relPath != "" {
		return nil, nil
	}

	sort.SliceStable(children, func(i, j int) bool {
		if children[i].Type == children[j].Type {
			return strings.Compare(children[i].Title, children[j].Title) < 0
		}
		return children[i].Type == NodeTypeDirectory
	})

	dispName := directoryDisplayName(b.root, relPath)
	rel := normalizeRelative(relPath)
	slug := slugify(rel)

	dirInfo, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat directory %s: %w", absPath, err)
	}

	return &Node{
		Name:         dispName,
		RawName:      filepath.Base(absPath),
		RelativePath: rel,
		Slug:         slug,
		Type:         NodeTypeDirectory,
		Title:        dispName,
		Modified:     dirInfo.ModTime(),
		Children:     children,
	}, nil
}

func (b *builder) buildFileNode(ctx context.Context, absPath, relPath string, info fs.FileInfo) (*Node, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(absPath) //nolint:gosec // absPath is constructed from validated root
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", absPath, err)
	}

	rel := normalizeRelative(relPath)
	display := fileDisplayName(filepath.Base(relPath))
	slug := slugify(strings.TrimSuffix(rel, filepath.Ext(rel)))

	var meta *renderer.Metadata
	title := display
	if b.opts.Renderer != nil {
		// Pass wiki-relative path (not absolute filesystem path) to renderer
		doc, err := b.opts.Renderer.Render(ctx, rel, info.ModTime(), content)
		if err != nil {
			return nil, fmt.Errorf("render metadata for %s: %w", rel, err)
		}
		metadata := doc.Metadata
		if !metadata.IsZero() {
			metaCopy := metadata
			meta = &metaCopy
			if metadata.Title != "" {
				title = metadata.Title
			}
		}
	}

	return &Node{
		Name:         display,
		RawName:      filepath.Base(relPath),
		RelativePath: rel,
		Slug:         slug,
		Type:         NodeTypeFile,
		Title:        title,
		Metadata:     meta,
		Modified:     info.ModTime(),
		Size:         info.Size(),
	}, nil
}

func isMarkdown(entry fs.DirEntry) bool {
	name := strings.ToLower(entry.Name())
	return strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".markdown")
}

func normalizeRelative(rel string) string {
	clean := filepath.ToSlash(rel)
	clean = strings.TrimPrefix(clean, "./")
	return clean
}

func fileDisplayName(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.ReplaceAll(name, "_", " ")
	return strings.TrimSpace(name)
}

func directoryDisplayName(root string, rel string) string {
	if rel == "" {
		return filepath.Base(root)
	}
	return fileDisplayName(rel)
}

func slugify(path string) string {
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		part = strings.TrimSuffix(part, filepath.Ext(part))
		part = strings.ReplaceAll(part, "_", " ")
		part = strings.ToLower(strings.TrimSpace(part))
		part = strings.ReplaceAll(part, " ", "-")
		parts[i] = part
	}
	return strings.Join(parts, "/")
}
