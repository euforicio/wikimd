package server

import (
	"net/http"
	"net/url"
	"strings"
)

// csrfMiddleware provides CSRF protection for mutation endpoints.
// It validates Origin or Referer headers for state-changing requests (POST, PUT, DELETE, PATCH).
// Safe methods (GET, HEAD, OPTIONS) and static/health endpoints bypass this check.
func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow safe methods
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF check for health and static endpoints
		path := r.URL.Path
		if path == "/healthz" || strings.HasPrefix(path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		// Validate origin or referer
		if !isValidOrigin(r) {
			http.Error(w, "Forbidden: Invalid origin", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isValidOrigin checks if the request comes from a valid origin.
func isValidOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		return false
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	// Extract host from request
	requestHost := r.Host
	if requestHost == "" {
		requestHost = r.URL.Host
	}

	// Normalize localhost variations
	originHost := normalizeHost(originURL.Host)
	targetHost := normalizeHost(requestHost)

	return originHost == targetHost
}

// normalizeHost treats localhost and 127.0.0.1 as equivalent.
func normalizeHost(host string) string {
	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Normalize localhost
	if host == "localhost" || host == "127.0.0.1" || host == "[::1]" {
		return "localhost"
	}

	return strings.ToLower(host)
}
