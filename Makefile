SHELL := /bin/bash
BIN_DIR := bin
BIN := $(BIN_DIR)/wikimd
EXPORT_BIN := $(BIN_DIR)/wikimd-export

VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X github.com/euforicio/wikimd/internal/buildinfo.Version=$(VERSION) -X github.com/euforicio/wikimd/internal/buildinfo.Commit=$(COMMIT) -X github.com/euforicio/wikimd/internal/buildinfo.Date=$(BUILD_DATE)

EXPORT_ROOT ?= $(or $(ROOT),.)
EXPORT_OUT ?= $(or $(OUT),dist)
EXPORT_ARGS ?=

DOCKER_IMAGE ?= wikimd:latest

.PHONY: dev build web-build lint tidy test export release docker

## Run the dev server with live reload hooks (Tailwind + Bun watchers).
dev:
	@set -euo pipefail; \
	mkdir -p static/css static/js; \
	(cd web && bun run dev:css) & \
	css_pid=$$!; \
	(cd web && bun run dev:js) & \
	js_pid=$$!; \
	trap "kill $$css_pid $$js_pid 2>/dev/null || true" EXIT; \
	GOFLAGS=-tags=dev go run ./cmd/wiki --root "$(or $(ROOT),.)" --port $(or $(PORT),8080)

## Build the server and exporter binaries with embedded assets.
build: web-build
	mkdir -p $(BIN_DIR)
	GOFLAGS= go build -o $(BIN) ./cmd/wiki
	GOFLAGS= go build -o $(EXPORT_BIN) ./cmd/wiki-export

## Run go test suite in parallel.
test:
	GOFLAGS= go test -p 4 -parallel 4 ./...

## Run golangci-lint if available.
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; skipping"; \
	fi

## Keep go.mod / go.sum tidy.
tidy:
	GOFLAGS= go mod tidy

## Build the Tailwind + Bun bundles once (auto-downloads vendors).
web-build:
	mkdir -p static/css static/js
	cd web && bun run build

## Export a static site bundle.
export: web-build
	GOFLAGS= go run ./cmd/wiki-export --root "$(EXPORT_ROOT)" --out "$(EXPORT_OUT)" $(if $(ASSETS),--assets "$(ASSETS)",) $(if $(ASSET_PREFIX),--asset-prefix "$(ASSET_PREFIX)",) $(if $(TITLE),--title "$(TITLE)",) $(if $(BASE_URL),--base-url "$(BASE_URL)",) $(if $(HIDDEN),--hidden,) $(if $(SEARCH_INDEX),--search-index,) $(EXPORT_ARGS)

## Produce stripped release binaries with version metadata embedded.
release: web-build
	mkdir -p $(BIN_DIR)
	GOFLAGS= CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o $(BIN) ./cmd/wiki
	GOFLAGS= CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o $(EXPORT_BIN) ./cmd/wiki-export

## Build a container image containing the wikimd server binary.
docker: web-build
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -t $(DOCKER_IMAGE) .
