package search_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/euforicio/wikimd/internal/search"
)

func TestSearchFindsMatches(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep (rg) not installed")
	}

	root := filepath.Join("..", "..", "testdata", "wiki")
	svc, err := search.NewService(root, nil)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	results, err := svc.Search(context.Background(), "Welcome", search.Options{Context: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected results, got zero")
	}

	found := false
	for _, r := range results {
		if strings.TrimPrefix(r.Path, "./") == "index.md" && strings.Contains(r.LineText, "# Welcome") {
			found = true
			if len(r.Before) == 0 {
				t.Fatalf("expected context lines before match, got %#v", r.Before)
			}
			break
		}
	}

	if !found {
		t.Fatalf("did not find target match in results: %#v", results)
	}
}

func TestSearchHonorsGlobExclusion(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep (rg) not installed")
	}

	root := filepath.Join("..", "..", "testdata", "wiki")
	svc, err := search.NewService(root, nil)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	results, err := svc.Search(context.Background(), "Guides", search.Options{ExcludeGlobs: []string{"guides/*"}})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	for _, r := range results {
		if strings.HasPrefix(r.Path, "guides/") {
			t.Fatalf("expected guides directory to be excluded, got result %#v", r)
		}
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "testdata", "wiki")
	svc, err := search.NewService(root, nil)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	if _, err := svc.Search(context.Background(), "  ", search.Options{}); err == nil {
		t.Fatalf("expected error for empty query")
	}
}
