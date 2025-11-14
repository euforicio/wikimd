package transform

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	d2renderer "github.com/euforicio/wikimd/internal/renderer/d2"
)

const d2Language = "d2"

// D2Transformer finds fenced ```d2 blocks and replaces them with rendered nodes.
type D2Transformer struct {
	renderer *d2renderer.Renderer
	logger   *slog.Logger
}

// NewD2Transformer constructs an AST transformer. If renderer is nil the transformer
// becomes a no-op.
func NewD2Transformer(renderer *d2renderer.Renderer, logger *slog.Logger) parser.ASTTransformer {
	return &D2Transformer{
		renderer: renderer,
		logger:   logger,
	}
}

// Transform implements parser.ASTTransformer.
func (t *D2Transformer) Transform(node *ast.Document, reader text.Reader, _ parser.Context) {
	if t.renderer == nil || node == nil {
		return
	}
	t.walk(node, reader)
}

func (t *D2Transformer) walk(parent ast.Node, reader text.Reader) {
	for child := parent.FirstChild(); child != nil; {
		next := child.NextSibling()

		if block, ok := child.(*ast.FencedCodeBlock); ok && isD2Block(block, reader.Source()) {
			replacement := t.renderBlock(block, reader)
			replacement.SetBlankPreviousLines(block.HasBlankPreviousLines())
			copyAttributes(block, replacement)
			parent.ReplaceChild(parent, block, replacement)
			child = next
			continue
		}

		if child.HasChildren() {
			t.walk(child, reader)
		}
		child = next
	}
}

func (t *D2Transformer) renderBlock(block *ast.FencedCodeBlock, reader text.Reader) *D2Block {
	source := blockSource(block, reader)
	result, err := t.renderer.Render(context.Background(), source)
	if err != nil {
		if t.logger != nil {
			t.logger.Warn("d2: render failed", "err", err)
		}
		return &D2Block{
			Source: source,
			Error:  err.Error(),
		}
	}
	return &D2Block{
		Source:  source,
		SVG:     result.SVG,
		Runtime: result.Duration,
	}
}

func isD2Block(block *ast.FencedCodeBlock, source []byte) bool {
	lang := strings.TrimSpace(string(block.Language(source)))
	return strings.EqualFold(lang, d2Language)
}

func blockSource(block *ast.FencedCodeBlock, reader text.Reader) string {
	var buf bytes.Buffer
	for i := 0; i < block.Lines().Len(); i++ {
		segment := block.Lines().At(i)
		buf.Write(segment.Value(reader.Source()))
	}
	return buf.String()
}

func copyAttributes(src ast.Node, dst ast.Node) {
	if src == nil || dst == nil {
		return
	}
	if src.Attributes() == nil {
		return
	}
	for _, attr := range src.Attributes() {
		dst.SetAttribute(attr.Name, attr.Value)
	}
}

// D2Block is a rendered diagram placeholder included directly in the AST.
type D2Block struct {
	ast.BaseBlock
	Source  string
	SVG     string
	Error   string
	Runtime time.Duration
}

// KindD2Block represents a rendered D2 node kind.
var KindD2Block = ast.NewNodeKind("D2Block")

// Kind implements ast.Node.
func (b *D2Block) Kind() ast.NodeKind {
	return KindD2Block
}

// IsRaw marks the node as raw HTML.
func (b *D2Block) IsRaw() bool {
	return true
}

// Dump aids debugging.
func (b *D2Block) Dump(source []byte, level int) {
	info := map[string]string{
		"Source": fmt.Sprintf("%d bytes", len(b.Source)),
	}
	if b.Error != "" {
		info["Error"] = fmt.Sprintf("%q", b.Error)
	}
	if b.Runtime > 0 {
		info["Runtime"] = b.Runtime.String()
	}
	ast.DumpHelper(b, source, level, info, nil)
}

// D2BlockRenderer writes rendered nodes into HTML output.
type D2BlockRenderer struct{}

// NewD2BlockRenderer returns a renderer for D2 nodes.
func NewD2BlockRenderer() renderer.NodeRenderer {
	return &D2BlockRenderer{}
}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *D2BlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindD2Block, r.renderD2Block)
}

func (r *D2BlockRenderer) renderD2Block(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	block := node.(*D2Block)

	var attrs strings.Builder
	if block.Runtime > 0 {
		fmt.Fprintf(&attrs, ` data-runtime-ms="%d"`, block.Runtime.Milliseconds())
	}
	if block.Source != "" {
		fmt.Fprintf(&attrs, ` data-source-b64="%s"`, encodeSource(block.Source))
	}

	if _, err := w.WriteString(`<div class="d2-block"` + attrs.String() + `>`); err != nil {
		return ast.WalkStop, err
	}

	var err error
	if block.Error != "" {
		_, err = w.WriteString(`<div class="d2-error">` + html.EscapeString(block.Error) + `</div>`)
	} else {
		_, err = w.WriteString(block.SVG)
	}
	if err != nil {
		return ast.WalkStop, err
	}

	if _, err := w.WriteString(`</div>`); err != nil {
		return ast.WalkStop, err
	}
	return ast.WalkSkipChildren, nil
}

func encodeSource(src string) string {
	return base64.StdEncoding.EncodeToString([]byte(src))
}
