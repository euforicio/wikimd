package mermaid

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommittedJSFixtures_StrictParity(t *testing.T) {
	renderer := NewRenderer(nil)
	ctx := context.Background()
	fixturesDir := filepath.Join("..", "..", "testdata", "comparison")

	required := []string{
		"flowchart_subgraph",
		"flowchart_self_edge",
		"flowchart_multi_edge",
		"flowchart_lr_direction",
		"flowchart_edge_labels",
	}

	for _, name := range required {
		t.Run(name, func(t *testing.T) {
			mmdPath := filepath.Join(fixturesDir, name+".mmd")
			jsPath := filepath.Join(fixturesDir, name+"_js.svg")

			src, err := os.ReadFile(mmdPath)
			if err != nil {
				t.Fatalf("read mmd fixture: %v", err)
			}
			expectedJS, err := os.ReadFile(jsPath)
			if err != nil {
				t.Fatalf("read js fixture: %v", err)
			}

			out, err := renderer.Render(ctx, string(src))
			if err != nil {
				t.Fatalf("render fixture: %v", err)
			}

			// Keep this gate stricter than loose JS comparison (50%) while tolerating
			// known renderer differences until full dagre parity lands.
			diffs, err := compareSVGsWithTolerance(out.SVG, string(expectedJS), 0.30)
			if err != nil {
				t.Fatalf("strict comparison failed: %v", err)
			}
			if len(diffs) > 0 {
				max := min(8, len(diffs))
				t.Fatalf("fixture drift against committed JS reference (%d diffs):\n%s", len(diffs), strings.Join(diffs[:max], "\n"))
			}

		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestCommittedJSFixtures_Existence(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "comparison")
	required := []string{
		"flowchart_subgraph",
		"flowchart_self_edge",
		"flowchart_multi_edge",
		"flowchart_lr_direction",
		"flowchart_edge_labels",
	}

	for _, name := range required {
		for _, suffix := range []string{".mmd", "_js.svg", "_go.svg"} {
			path := filepath.Join(fixturesDir, fmt.Sprintf("%s%s", name, suffix))
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("missing required fixture %q: %v", path, err)
			}
		}
	}
}
