# syntax=docker/dockerfile:1.5

# Build frontend assets with Bun
FROM oven/bun:1 AS web
WORKDIR /src
COPY web/package.json web/bun.lock ./web/
RUN bun install --cwd web --frozen-lockfile
COPY vendor.json .
COPY web web
COPY static static
RUN bun --cwd web run build

# Build Go binary
FROM golang:1.25-alpine AS build
WORKDIR /src
ARG TARGETOS=linux
ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy all built static assets from web stage (including downloaded vendors)
COPY --from=web /src/static/css ./static/css
COPY --from=web /src/static/js ./static/js
COPY --from=web /src/static/vendor ./static/vendor
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=
RUN go build -ldflags "-s -w -X github.com/euforicio/wikimd/internal/buildinfo.Version=${VERSION} -X github.com/euforicio/wikimd/internal/buildinfo.Commit=${COMMIT} -X github.com/euforicio/wikimd/internal/buildinfo.Date=${BUILD_DATE}" -o /out/wikimd ./cmd/wikimd

# Runtime image with ripgrep for search functionality
FROM alpine:3.21
RUN apk add --no-cache ripgrep ca-certificates tzdata && \
    addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S -G nonroot -h /home/nonroot nonroot
USER nonroot
WORKDIR /data
COPY --from=build /out/wikimd /usr/local/bin/wikimd
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/wikimd"]
CMD ["--root", "/data", "--port", "8080", "--auto-open=false"]
