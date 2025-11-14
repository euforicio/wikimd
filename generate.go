// Package wikimd provides a markdown-based wiki system with live preview.
//
// Build web assets (CSS and JavaScript bundles) using:
//
//	go generate
package wikimd

//go:generate sh -c "mkdir -p static/vendor web/src/styles && GOFLAGS= go run ./tools/generate-chroma-css > static/vendor/chroma-github-dark.min.css && cp static/vendor/chroma-github-dark.min.css web/src/styles/chroma-github-dark.css"
//go:generate sh -c "mkdir -p static/css static/js && cd web && bun run build"
