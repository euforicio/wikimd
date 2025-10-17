package server

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// middleware is a function that wraps an http.Handler.
type middleware func(http.Handler) http.Handler

// chain applies multiple middleware in order.
func chain(h http.Handler, mw ...middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// recoveryMiddleware recovers from panics and returns a 500 error.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.ErrorContext(r.Context(), "panic recovered",
					slog.Any("err", err),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// gzipMiddleware compresses responses if the client accepts gzip encoding.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzip.NewWriter(w)
		defer func() {
			if err := gz.Close(); err != nil {
				slog.Error("failed to close gzip writer", slog.Any("err", err))
			}
		}()

		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gz,
		}

		next.ServeHTTP(gzw, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer        io.Writer
	headerWritten bool
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if !w.headerWritten {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		w.headerWritten = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.Writer.Write(b)
}

// Flush implements http.Flusher.
func (w *gzipResponseWriter) Flush() {
	if f, ok := w.Writer.(*gzip.Writer); ok {
		if err := f.Flush(); err != nil {
			slog.Error("failed to flush gzip writer", slog.Any("err", err))
		}
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// loggingMiddleware logs HTTP requests with slog.
func loggingMiddleware(logger *slog.Logger, verbose bool) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !verbose {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			duration := time.Since(start)
			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("uri", r.RequestURI),
				slog.Int("status", sw.status),
				slog.Duration("latency", duration),
				slog.String("remote_ip", r.RemoteAddr),
				slog.String("protocol", r.Proto),
			}
			if cl := r.Header.Get("Content-Length"); cl != "" {
				if n, err := strconv.ParseInt(cl, 10, 64); err == nil {
					attrs = append(attrs, slog.Int64("bytes_in", n))
				}
			}

			logger.LogAttrs(r.Context(), slog.LevelInfo, "http request", attrs...)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
