// Package static embeds and serves frontend assets (CSS, JS, vendor files).
package static

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed css/*.css js/*.js js/chunks/*.js vendor/*
var assets embed.FS

// FS exposes the embedded static assets.
func FS() fs.FS {
	return assets
}

// HTTP returns an http.FileSystem backed by the embedded assets.
func HTTP() http.FileSystem {
	return http.FS(assets)
}

// Has reports whether the given relative path exists in the embedded assets.
func Has(name string) bool {
	name = strings.TrimPrefix(name, "/")
	f, err := assets.Open(name)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// CopyAll writes all embedded assets into the destination directory (relative layout preserved).
func CopyAll(dest string) error {
	return fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		target := filepath.Join(dest, path)
		if err := ensureDir(target); err != nil {
			return err
		}
		data, err := fs.ReadFile(assets, path)
		if err != nil {
			return err
		}
		return writeFile(target, data)
	})
}

func ensureDir(target string) error {
	dir := filepath.Dir(target)
	return os.MkdirAll(dir, 0o755) //nolint:gosec // standard directory permissions
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644) //nolint:gosec // standard file permissions
}
