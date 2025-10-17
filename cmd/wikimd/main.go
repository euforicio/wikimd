// Package main provides the wikimd server application entrypoint.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"github.com/euforicio/wikimd/internal/buildinfo"
	"github.com/euforicio/wikimd/internal/config"
	"github.com/euforicio/wikimd/internal/content"
	"github.com/euforicio/wikimd/internal/renderer"
	"github.com/euforicio/wikimd/internal/search"
	"github.com/euforicio/wikimd/internal/server"
)

func main() {
	cfg := config.Default()
	config.ApplyEnvOverrides(&cfg)

	flags := pflag.NewFlagSet("wikimd", pflag.ExitOnError)
	config.RegisterFlags(flags, &cfg)
	versionFlag := flags.Bool("version", false, "Print version information and exit")
	if err := flags.Parse(os.Args[1:]); err != nil {
		slog.Error("parse flags", slog.Any("err", err))
		os.Exit(1)
	}
	if *versionFlag {
		println(buildinfo.Summary())
		os.Exit(0)
	}
	if err := config.Finalize(&cfg); err != nil {
		slog.Error("invalid configuration", slog.Any("err", err))
		os.Exit(1)
	}

	logLevel := slog.LevelWarn
	if cfg.Verbose {
		logLevel = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	logger = logger.With("app", "wikimd")
	slog.SetDefault(logger)
	logger.Log(context.Background(), slog.LevelInfo-1, "starting wikimd", slog.String("version", buildinfo.Summary()))

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	rendererSvc := renderer.NewService(logger)
	contentSvc, err := content.NewService(ctx, cfg.RootDir, rendererSvc, logger, content.Options{})
	if err != nil {
		cancel()
		logger.Error("content service init failed", slog.Any("err", err))
		//nolint:gocritic // exitAfterDefer: cancel() explicitly called before os.Exit
		os.Exit(1)
	}
	defer func() {
		if err := contentSvc.Close(); err != nil {
			logger.Error("close content service", slog.Any("err", err))
		}
	}()

	searchSvc, err := search.NewService(cfg.RootDir, logger)
	if err != nil {
		cancel()
		logger.Error("search service init failed", slog.Any("err", err))
		os.Exit(1)
	}

	srv, err := server.New(cfg, logger, contentSvc, searchSvc)
	if err != nil {
		cancel()
		logger.Error("server init failed", slog.Any("err", err))
		os.Exit(1)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown error", slog.Any("err", err))
		}
	}()

	if err := srv.Start(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info("shutdown complete")
			return
		}
		logger.Error("server error", slog.Any("err", err))
		os.Exit(1)
	}
}
