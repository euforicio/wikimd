// Package config manages application configuration from environment variables and flags.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

const envPrefix = "WIKIMD_"

// Config holds runtime configuration for the wiki server and exporter.
type Config struct {
	RootDir       string
	StaticOutput  string
	AssetsDir     string
	Port          int
	AutoOpen      bool
	DarkModeFirst bool
	Verbose       bool
}

// Default returns ready-to-use defaults prior to env/flag overrides.
func Default() Config {
	return Config{
		RootDir:       ".",
		Port:          0, // 0 = auto-select random available port
		AutoOpen:      true,
		DarkModeFirst: true,
		StaticOutput:  "dist",
		AssetsDir:     "static",
	}
}

// RegisterFlags attaches configuration flags to the provided FlagSet.
func RegisterFlags(fs *pflag.FlagSet, cfg *Config) {
	fs.StringVarP(&cfg.RootDir, "root", "r", cfg.RootDir, "root directory containing markdown files")
	fs.IntVarP(&cfg.Port, "port", "p", cfg.Port, "port to bind the HTTP server (0 = auto-assign, default: auto)")
	fs.BoolVar(&cfg.AutoOpen, "auto-open", cfg.AutoOpen, "open the browser automatically after start")
	fs.BoolVar(&cfg.DarkModeFirst, "dark", cfg.DarkModeFirst, "enable dark theme by default")
	fs.StringVar(&cfg.StaticOutput, "out", cfg.StaticOutput, "default output directory for static export")
	fs.StringVar(&cfg.AssetsDir, "assets", cfg.AssetsDir, "directory containing built frontend assets")
	fs.BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "enable verbose logging (HTTP requests)")
}

// ApplyEnvOverrides reads supported environment variables and overrides cfg in place.
func ApplyEnvOverrides(cfg *Config) {
	applyStringEnv("ROOT", func(v string) { cfg.RootDir = v })
	applyIntEnv("PORT", func(v int) { cfg.Port = v })
	applyBoolEnv("AUTO_OPEN", func(v bool) { cfg.AutoOpen = v })
	applyBoolEnv("DARK", func(v bool) { cfg.DarkModeFirst = v })
	applyStringEnv("OUT", func(v string) { cfg.StaticOutput = v })
	applyStringEnv("ASSETS", func(v string) { cfg.AssetsDir = v })
	applyBoolEnv("VERBOSE", func(v bool) { cfg.Verbose = v })
}

func applyStringEnv(key string, apply func(string)) {
	if raw, ok := lookupNonEmpty(key); ok {
		apply(raw)
	}
}

func applyIntEnv(key string, apply func(int)) {
	if raw, ok := lookupNonEmpty(key); ok {
		if value, err := strconv.Atoi(raw); err == nil {
			apply(value)
		}
	}
}

func applyBoolEnv(key string, apply func(bool)) {
	if raw, ok := lookupNonEmpty(key); ok {
		if value, err := strconv.ParseBool(raw); err == nil {
			apply(value)
		}
	}
}

func lookupNonEmpty(key string) (string, bool) {
	raw, ok := os.LookupEnv(envPrefix + key)
	if !ok {
		return "", false
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	return value, true
}

// Finalize validates and normalizes paths.
func Finalize(cfg *Config) error {
	root, err := filepath.Abs(cfg.RootDir)
	if err != nil {
		return fmt.Errorf("resolve root directory: %w", err)
	}
	cfg.RootDir = root

	// Allow port 0 for dynamic allocation, otherwise validate range
	if cfg.Port < 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}

	if cfg.StaticOutput == "" {
		cfg.StaticOutput = "dist"
	}

	if cfg.AssetsDir == "" {
		cfg.AssetsDir = "static"
	}
	assets, err := filepath.Abs(cfg.AssetsDir)
	if err != nil {
		return fmt.Errorf("resolve assets directory: %w", err)
	}
	cfg.AssetsDir = assets

	return nil
}
