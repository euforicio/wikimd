// Package search provides ripgrep-based full-text search across markdown files.
package search

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Options controls the behavior of the ripgrep search.
type Options struct {
	IncludeGlobs  []string
	ExcludeGlobs  []string
	Context       int
	CaseSensitive bool
	SearchHidden  bool
}

// Result represents a single match from ripgrep.
type Result struct {
	Path     string        `json:"path"`
	Match    string        `json:"match"`
	LineText string        `json:"lineText"`
	Before   []LineSnippet `json:"before,omitempty"`
	After    []LineSnippet `json:"after,omitempty"`
	Line     int           `json:"line"`
	Column   int           `json:"column"`
}

// LineSnippet captures contextual lines around a match.
type LineSnippet struct {
	Text string `json:"text"`
	Line int    `json:"line"`
}

// Service executes ripgrep searches rooted at the repository.
type Service struct {
	logger *slog.Logger
	root   string
}

// NewService constructs a ripgrep-backed search service.
func NewService(root string, logger *slog.Logger) (*Service, error) {
	if root == "" {
		return nil, errors.New("root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	if logger == nil {
		logger = slog.Default()
	}

	if _, err := exec.LookPath("rg"); err != nil {
		return nil, fmt.Errorf("ripgrep executable not found in PATH: %w", err)
	}

	return &Service{root: abs, logger: logger.With("component", "search")}, nil
}

// Search executes ripgrep with the provided query and options.
//
//nolint:gocognit,gocyclo // ripgrep argument building requires option handling
func (s *Service) Search(ctx context.Context, query string, opts Options) ([]Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query cannot be empty")
	}

	args := []string{"--json", "--line-number", "--color=never", "--no-heading"}
	if opts.CaseSensitive {
		args = append(args, "--case-sensitive")
	} else {
		args = append(args, "--smart-case")
	}
	if opts.Context > 0 {
		args = append(args, "-C", strconv.Itoa(opts.Context))
	}
	for _, glob := range opts.IncludeGlobs {
		if glob = strings.TrimSpace(glob); glob != "" {
			args = append(args, "--glob", glob)
		}
	}
	for _, glob := range opts.ExcludeGlobs {
		if glob = strings.TrimSpace(glob); glob != "" {
			if !strings.HasPrefix(glob, "!") {
				glob = "!" + glob
			}
			args = append(args, "--glob", glob)
		}
	}
	if opts.SearchHidden {
		args = append(args, "--hidden")
	}

	args = append(args, "--", query, "./")

	cmd := exec.CommandContext(ctx, "rg", args...)
	cmd.Dir = s.root

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderrBuf := &bytes.Buffer{}
	cmd.Stderr = stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start rg: %w", err)
	}

	results, err := parseRipgrepJSON(stdout, opts)
	if err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return results, nil
		}
		if exitErr != nil {
			return nil, fmt.Errorf("rg error (exit %d): %s", exitErr.ExitCode(), strings.TrimSpace(stderrBuf.String()))
		}
		return nil, err
	}

	return results, nil
}

type rgMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type rgMatch struct {
	Path struct {
		Text string `json:"text"`
	} `json:"path"`
	Lines struct {
		Text string `json:"text"`
	} `json:"lines"`
	Submatches []struct {
		Match struct {
			Text string `json:"text"`
		} `json:"match"`
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"submatches"`
	LineNumber int `json:"line_number"`
}

type rgContext struct {
	Path struct {
		Text string `json:"text"`
	} `json:"path"`
	Lines struct {
		Text string `json:"text"`
	} `json:"lines"`
	LineNumber int `json:"line_number"`
}

//nolint:gocognit,gocyclo // JSON parsing with context handling has inherent complexity
func parseRipgrepJSON(r io.Reader, opts Options) ([]Result, error) {
	dec := json.NewDecoder(bufio.NewReader(r))

	contextLines := make(map[string]map[int]string)
	var results []Result

	for {
		var msg rgMessage
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode ripgrep output: %w", err)
		}

		switch msg.Type {
		case "match":
			var m rgMatch
			if err := json.Unmarshal(msg.Data, &m); err != nil {
				return nil, fmt.Errorf("decode match: %w", err)
			}

			res := Result{
				Path:     m.Path.Text,
				Line:     m.LineNumber,
				LineText: strings.TrimRight(m.Lines.Text, "\n"),
			}

			if len(m.Submatches) > 0 {
				sub := m.Submatches[0]
				res.Match = sub.Match.Text
				res.Column = sub.Start + 1
			}

			if opts.Context > 0 {
				ctxLines := contextLines[m.Path.Text]
				if ctxLines != nil {
					for i := 1; i <= opts.Context; i++ {
						if line, ok := ctxLines[m.LineNumber-i]; ok {
							res.Before = append([]LineSnippet{{Line: m.LineNumber - i, Text: line}}, res.Before...)
						}
					}
					for i := 1; i <= opts.Context; i++ {
						if line, ok := ctxLines[m.LineNumber+i]; ok {
							res.After = append(res.After, LineSnippet{Line: m.LineNumber + i, Text: line})
						}
					}
				}
			}

			results = append(results, res)
		case "context":
			if opts.Context == 0 {
				continue
			}
			var ctxMsg rgContext
			if err := json.Unmarshal(msg.Data, &ctxMsg); err != nil {
				return nil, fmt.Errorf("decode context: %w", err)
			}
			pathKey := ctxMsg.Path.Text
			if _, ok := contextLines[pathKey]; !ok {
				contextLines[pathKey] = make(map[int]string)
			}
			contextLines[pathKey][ctxMsg.LineNumber] = strings.TrimRight(ctxMsg.Lines.Text, "\n")
		}
	}

	return results, nil
}
