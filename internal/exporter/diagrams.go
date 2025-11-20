package exporter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"

	d2renderer "github.com/euforicio/wikimd/internal/renderer/d2"
)

// diagramEncoder turns fenced diagram blocks into data URI images so downstream
// renderers (PDF or text) don't need to understand custom nodes.
type diagramEncoder struct {
	d2 *d2renderer.Renderer
}

// encode rewrites ```d2 fences into Markdown image tags with embedded PNG data.
// If D2 rendering fails, the original fence is left intact so the caller can
// still surface the source.
func (e *diagramEncoder) encode(raw []byte) ([]byte, error) {
	var (
		out          bytes.Buffer
		scanner      = bufio.NewScanner(bytes.NewReader(raw))
		inFence      bool
		fenceMarker  string
		fenceLang    string
		diagramLines bytes.Buffer
	)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inFence {
			if marker, lang, ok := parseFenceStart(trimmed); ok {
				inFence = true
				fenceMarker = marker
				fenceLang = lang
				diagramLines.Reset()
				// Preserve non-diagram fences verbatim
				if !isDiagramFence(lang) {
					writeLine(&out, line)
				}
				continue
			}

			writeLine(&out, line)
			continue
		}

		// inside a fenced block
		if isFenceEnd(trimmed, fenceMarker) {
			switch strings.ToLower(fenceLang) {
			case "d2":
				if err := e.flushD2(&out, diagramLines.String()); err != nil {
					writeLine(&out, fenceMarker)
					out.Write(diagramLines.Bytes())
					writeLine(&out, fenceMarker)
				}
			case "mermaid":
				if err := e.flushMermaid(&out, diagramLines.String()); err != nil {
					writeLine(&out, fenceMarker)
					out.Write(diagramLines.Bytes())
					writeLine(&out, fenceMarker)
				}
			default:
				writeLine(&out, line)
			}
			inFence = false
			fenceMarker = ""
			fenceLang = ""
			continue
		}

		if strings.EqualFold(fenceLang, "d2") || strings.EqualFold(fenceLang, "mermaid") {
			writeLine(&diagramLines, line)
		} else {
			writeLine(&out, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Unclosed fence: emit buffered content as-is
	if inFence {
		writeLine(&out, fenceMarker)
		out.Write(diagramLines.Bytes())
	}

	return out.Bytes(), nil
}

func (e *diagramEncoder) flushD2(out *bytes.Buffer, source string) error {
	if strings.TrimSpace(source) == "" {
		return nil
	}

	if e == nil || e.d2 == nil {
		return fmt.Errorf("d2 renderer unavailable")
	}

	res, err := e.d2.Render(context.Background(), source)
	if err != nil {
		return fmt.Errorf("render d2: %w", err)
	}

	pngData, err := svgToPNG([]byte(res.SVG))
	if err != nil {
		return fmt.Errorf("rasterize d2 svg: %w", err)
	}

	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData)
	_, err = fmt.Fprintf(out, "![D2 diagram](%s)\n\n", dataURI)
	return err
}

func (e *diagramEncoder) flushMermaid(out *bytes.Buffer, source string) error {
	if strings.TrimSpace(source) == "" {
		return nil
	}

	pngData, err := renderMermaidWithCLI(source)
	if err != nil {
		return fmt.Errorf("render mermaid: %w", err)
	}

	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData)
	_, err = fmt.Fprintf(out, "![Mermaid diagram](%s)\n\n", dataURI)
	return err
}

func parseFenceStart(line string) (marker, lang string, ok bool) {
	if strings.HasPrefix(line, "```") {
		marker = line[:leadingCount(line, '`')]
		lang = strings.TrimSpace(strings.TrimPrefix(line, marker))
		ok = len(marker) >= 3
		return
	}

	if strings.HasPrefix(line, "~~~") {
		marker = line[:leadingCount(line, '~')]
		lang = strings.TrimSpace(strings.TrimPrefix(line, marker))
		ok = len(marker) >= 3
		return
	}

	return "", "", false
}

func isFenceEnd(line, marker string) bool {
	if marker == "" {
		return false
	}
	close := strings.Repeat(string(marker[0]), len(marker))
	return line == close
}

func isDiagramFence(lang string) bool {
	lang = strings.ToLower(strings.TrimSpace(lang))
	return lang == "d2" || lang == "mermaid"
}

func leadingCount(line string, char rune) int {
	count := 0
	for _, r := range line {
		if r == char {
			count++
			continue
		}
		break
	}
	return count
}

func writeLine(buf *bytes.Buffer, line string) {
	buf.WriteString(line)
	buf.WriteByte('\n')
}

// svgToPNG rasterizes an SVG into a PNG byte slice suitable for embedding as a data URI.
func svgToPNG(svg []byte) ([]byte, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svg))
	if err != nil {
		return nil, fmt.Errorf("parse svg: %w", err)
	}

	viewbox := icon.ViewBox
	width := int(math.Ceil(viewbox.W))
	height := int(math.Ceil(viewbox.H))
	if width <= 0 || height <= 0 {
		// Sensible default to avoid zero-sized canvases
		width, height = 800, 600
	}

	icon.SetTarget(0, 0, float64(width), float64(height))

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	scanner := rasterx.NewScannerGV(width, height, canvas, canvas.Bounds())
	raster := rasterx.NewDasher(width, height, scanner)
	icon.Draw(raster, 1.0)

	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}

func renderMermaidWithCLI(source string) ([]byte, error) {
	bin, err := exec.LookPath("mmdc")
	if err != nil {
		return nil, fmt.Errorf("mmdc not found: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "mermaid-cli-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	inPath := filepath.Join(tmpDir, "diagram.mmd")
	outPath := filepath.Join(tmpDir, "diagram.png")

	if err := os.WriteFile(inPath, []byte(source), 0o644); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin,
		"-i", inPath,
		"-o", outPath,
		"-b", "white",
		"-s", "2",
		"--quiet",
	)
	// mmdc writes temp files next to input; keep cwd in tmpdir
	cmd.Dir = tmpDir

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("mmdc failed: %w; output: %s", err, string(output))
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("mmdc produced empty png")
	}
	return data, nil
}
