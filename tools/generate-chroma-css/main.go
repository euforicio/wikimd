// Package main generates Chroma CSS stylesheets for syntax highlighting.
package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

func main() {
	style := styles.Get("github-dark")
	if style == nil {
		fmt.Fprintf(os.Stderr, "Style 'github-dark' not found\n")
		os.Exit(1)
	}

	formatter := html.New(
		html.WithClasses(true),
		html.ClassPrefix(""),
	)

	if err := formatter.WriteCSS(os.Stdout, style); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating CSS: %v\n", err)
		os.Exit(1)
	}
}
