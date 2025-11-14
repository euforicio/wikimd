package d2

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2layouts/d2elklayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	"oss.terrastruct.com/d2/d2themes/d2themescatalog"
	d2log "oss.terrastruct.com/d2/lib/log"
	"oss.terrastruct.com/d2/lib/textmeasure"
)

// Result captures the outcome of a render attempt.
type Result struct {
	SVG      string
	Duration time.Duration
}

var (
	// ErrEmptyDiagram is returned when the supplied diagram body is empty.
	ErrEmptyDiagram = errors.New("empty d2 diagram")
)

// Renderer performs server-side D2 compilation using the embedded D2 compiler.
// Layout choices are left entirely to the source diagram (via D2 config blocks)
// or environment variables such as D2_LAYOUT.
type Renderer struct {
	logger  *slog.Logger
	timeout time.Duration
}

// Options configure the renderer.
type Options struct {
	Timeout time.Duration
}

// New creates a renderer instance. The provided context is unused for now but
// kept for API parity with future enhancements.
func New(_ context.Context, logger *slog.Logger, opts *Options) (*Renderer, error) {
	if logger == nil {
		logger = slog.Default()
	}

	cfg := Options{
		Timeout: 12 * time.Second,
	}
	if opts != nil && opts.Timeout > 0 {
		cfg.Timeout = opts.Timeout
	}

	return &Renderer{
		logger:  logger,
		timeout: cfg.Timeout,
	}, nil
}

// Render compiles the given D2 script into SVG, respecting any layout directives
// defined inside the document itself.
func (r *Renderer) Render(ctx context.Context, source string) (Result, error) {
	if strings.TrimSpace(source) == "" {
		return Result{}, ErrEmptyDiagram
	}
	if ctx == nil {
		ctx = context.Background()
	}

	ctx = d2log.With(ctx, r.logger)
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return Result{}, fmt.Errorf("init ruler: %w", err)
	}

	themeID := d2themescatalog.DarkFlagshipTerrastruct.ID
	darkThemeID := d2themescatalog.DarkFlagshipTerrastruct.ID
	pad := int64(d2svg.DEFAULT_PADDING)
	renderOpts := &d2svg.RenderOpts{
		ThemeID:     &themeID,
		DarkThemeID: &darkThemeID,
		Pad:         &pad,
	}

	start := time.Now()
	compileOpts := &d2lib.CompileOptions{
		Ruler:          ruler,
		LayoutResolver: r.layoutResolver,
	}

	diagram, _, err := d2lib.Compile(ctx, source, compileOpts, renderOpts)
	if err != nil {
		return Result{}, err
	}
	if diagram == nil {
		return Result{}, errors.New("d2 compiler returned nil diagram")
	}

	svg, err := d2svg.Render(diagram, renderOpts)
	if err != nil {
		return Result{}, fmt.Errorf("render svg: %w", err)
	}

	return Result{
		SVG:      string(svg),
		Duration: time.Since(start),
	}, nil
}

func (r *Renderer) layoutResolver(engine string) (d2graph.LayoutGraph, error) {
	switch strings.ToLower(engine) {
	case "", "dagre":
		return func(ctx context.Context, g *d2graph.Graph) error {
			return d2dagrelayout.Layout(ctx, g, nil)
		}, nil
	case "elk":
		return func(ctx context.Context, g *d2graph.Graph) error {
			return d2elklayout.Layout(ctx, g, nil)
		}, nil
	default:
		return nil, fmt.Errorf("unsupported D2 layout %q (install plugin for advanced engines)", engine)
	}
}
