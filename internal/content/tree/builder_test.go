package tree_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/euforicio/wikimd/internal/content/tree"
	"github.com/euforicio/wikimd/internal/renderer"
)

func TestBuildGeneratesTreeWithMetadata(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "testdata", "wiki")
	svc := renderer.NewService(nil)

	node, err := tree.Build(context.Background(), root, tree.Options{Renderer: svc})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if node == nil {
		t.Fatalf("expected root node")
	}
	if node.Type != tree.NodeTypeDirectory {
		t.Fatalf("expected root to be directory, got %s", node.Type)
	}
	if len(node.Children) < 2 {
		t.Fatalf("expected at least 2 children at root (guides dir + files), got %d", len(node.Children))
	}

	// Find the guides directory and index file
	var guides, index *tree.Node
	for _, child := range node.Children {
		if child.Type == tree.NodeTypeDirectory && child.Name == "guides" {
			guides = child
		}
		if child.Type == tree.NodeTypeFile && child.Name == "index" {
			index = child
		}
	}

	if guides == nil {
		t.Fatalf("expected guides directory in children")
	}
	if index == nil {
		t.Fatalf("expected index file in children")
	}

	// Check guides has correct children
	if len(guides.Children) != 2 {
		t.Fatalf("expected guides to have 2 children, got %d", len(guides.Children))
	}

	// Find getting-started file
	var gettingStarted *tree.Node
	for _, child := range guides.Children {
		if strings.Contains(child.RelativePath, "getting") {
			gettingStarted = child
			break
		}
	}
	if gettingStarted == nil {
		t.Fatalf("expected getting-started file in guides")
	}
	if gettingStarted.Title != "Getting Started" {
		t.Fatalf("expected metadata title 'Getting Started', got %q", gettingStarted.Title)
	}
	if gettingStarted.Metadata == nil || gettingStarted.Metadata.Title != "Getting Started" {
		t.Fatalf("expected metadata to be populated")
	}
	if gettingStarted.Slug != "guides/getting-started" {
		t.Fatalf("unexpected slug: %s", gettingStarted.Slug)
	}

	// Find advanced topics file
	var advanced *tree.Node
	for _, child := range guides.Children {
		if strings.Contains(child.RelativePath, "advanced") {
			advanced = child
			break
		}
	}
	if advanced == nil {
		t.Fatalf("expected advanced topics file in guides")
	}

	if advanced.Name != "advanced topics" {
		t.Fatalf("expected underscores replaced in name, got %q", advanced.Name)
	}
	if advanced.Title != "advanced topics" {
		t.Fatalf("expected default title from filename, got %q", advanced.Title)
	}
	if advanced.Metadata != nil {
		t.Fatalf("expected nil metadata when no frontmatter")
	}

	// Verify RawName is preserved
	if guides.RawName != "guides" {
		t.Fatalf("unexpected raw name for guides: %s", guides.RawName)
	}
	if index.RawName != "index.md" {
		t.Fatalf("unexpected raw name for index: %s", index.RawName)
	}
}

func TestHiddenFilesExcludedByDefault(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "testdata", "wiki")
	node, err := tree.Build(context.Background(), root, tree.Options{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if len(node.Children) < 2 {
		t.Fatalf("expected at least 2 children, got %d", len(node.Children))
	}

	// Recursively check that no hidden files are included
	var checkHidden func(*tree.Node)
	checkHidden = func(n *tree.Node) {
		if strings.HasPrefix(n.RawName, ".") {
			t.Fatalf("hidden file should be excluded: %s (path: %s)", n.RawName, n.RelativePath)
		}
		for _, child := range n.Children {
			checkHidden(child)
		}
	}

	for _, child := range node.Children {
		checkHidden(child)
	}
}

func TestDependencyDirectoriesExcluded(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	contentDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "overview.md"), []byte("# Overview"), 0o644); err != nil {
		t.Fatalf("write overview: %v", err)
	}
	depDir := filepath.Join(root, "node_modules", "lib")
	if err := os.MkdirAll(depDir, 0o755); err != nil {
		t.Fatalf("mkdir dep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depDir, "README.md"), []byte("# Should not show"), 0o644); err != nil {
		t.Fatalf("write dep readme: %v", err)
	}

	node, err := tree.Build(context.Background(), root, tree.Options{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if node == nil {
		t.Fatalf("expected root node")
	}

	for _, child := range node.Children {
		if strings.Contains(child.RelativePath, "node_modules") {
			t.Fatalf("node_modules should be excluded from tree: %+v", child)
		}
	}
}

func TestBuildExtractsMetadataWithoutRendererEvenWithBrokenMermaid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	content := `---
title: Broken Mermaid Page
description: Startup should not fail on Mermaid issues
tags:
  - docs
  - mermaid
---

# Broken Mermaid

` + "```mermaid\n" + `this is not valid mermaid
` + "```" + `
`
	if err := os.WriteFile(filepath.Join(root, "broken.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write broken.md: %v", err)
	}

	node, err := tree.Build(context.Background(), root, tree.Options{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if node == nil {
		t.Fatalf("expected root node")
	}
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(node.Children))
	}

	page := node.Children[0]
	if page.Type != tree.NodeTypeFile {
		t.Fatalf("expected file node, got %s", page.Type)
	}
	if page.Title != "Broken Mermaid Page" {
		t.Fatalf("expected title from frontmatter, got %q", page.Title)
	}
	if page.Metadata == nil {
		t.Fatalf("expected metadata to be populated")
	}
	if page.Metadata.Description != "Startup should not fail on Mermaid issues" {
		t.Fatalf("unexpected description: %q", page.Metadata.Description)
	}
	if len(page.Metadata.Tags) != 2 || page.Metadata.Tags[0] != "docs" || page.Metadata.Tags[1] != "mermaid" {
		t.Fatalf("unexpected tags: %#v", page.Metadata.Tags)
	}
}
