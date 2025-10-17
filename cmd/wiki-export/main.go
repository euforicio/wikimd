// Package main provides the wikimd static site export CLI.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/pflag"

	"github.com/euforicio/wikimd/internal/buildinfo"
	"github.com/euforicio/wikimd/internal/config"
	"github.com/euforicio/wikimd/internal/exporter"
)

func main() {
	cfg := config.Default()
	config.ApplyEnvOverrides(&cfg)

	flags := pflag.NewFlagSet("wikimd-export", pflag.ExitOnError)
	flags.StringVarP(&cfg.RootDir, "root", "r", cfg.RootDir, "root directory containing markdown files to export")
	flags.StringVar(&cfg.StaticOutput, "out", cfg.StaticOutput, "output directory for generated static site")
	flags.StringVar(&cfg.AssetsDir, "assets", cfg.AssetsDir, "directory containing prepared static assets to copy")

	includeHidden := flags.Bool("hidden", false, "include hidden files when scanning the content tree")
	title := flags.String("title", "wikimd", "site title to use for exported pages")
	darkMode := flags.Bool("dark", cfg.DarkModeFirst, "enable dark mode by default in the exported site")
	searchIndex := flags.Bool("search-index", false, "generate a simple JSON search index alongside the export")
	clean := true
	flags.BoolVar(&clean, "clean", true, "wipe the output directory before exporting")
	assetPrefix := flags.String("asset-prefix", "assets", "relative directory name for copied assets within the export output")
	baseURL := flags.String("base-url", "", "optional absolute base URL for canonical link tags")

	if err := flags.Parse(os.Args[1:]); err != nil {
		slog.Error("flag parsing failed", slog.Any("err", err))
		os.Exit(1)
	}

	if err := config.Finalize(&cfg); err != nil {
		slog.Error("invalid configuration", slog.Any("err", err))
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	logger.Info("starting wikimd-export", slog.String("version", buildinfo.Summary()))

	assetsOverride := ""
	if flags.Changed("assets") {
		assetsOverride = cfg.AssetsDir
	}

	exp, err := exporter.New(logger)
	if err != nil {
		logger.Error("init exporter failed", slog.Any("err", err))
		os.Exit(1)
	}

	ctx := context.Background()
	if err := exp.Export(ctx, exporter.Options{
		Root:                cfg.RootDir,
		OutputDir:           cfg.StaticOutput,
		AssetsDir:           assetsOverride,
		IncludeHidden:       *includeHidden,
		SiteTitle:           *title,
		DarkModeFirst:       *darkMode,
		GenerateSearchIndex: *searchIndex,
		CleanOutput:         clean,
		AssetPrefix:         *assetPrefix,
		BaseURL:             *baseURL,
	}); err != nil {
		logger.Error("export failed", slog.Any("err", err))
		os.Exit(1)
	}

	logger.Info("export succeeded", slog.String("output", cfg.StaticOutput))
}
