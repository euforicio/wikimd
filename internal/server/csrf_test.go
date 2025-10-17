package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFProtection(t *testing.T) {
	t.Parallel()
	srv, cleanup := newTestServer(t)
	t.Cleanup(cleanup)

	tests := []struct { //nolint:govet // test cases prefer readability over memory layout
		name           string
		method         string
		path           string
		setOrigin      bool
		originValue    string
		setReferer     bool
		refererValue   string
		setHost        bool
		hostValue      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "GET requests bypass CSRF check",
			method:         http.MethodGet,
			path:           "/api/page/index.md",
			setHost:        false,
			setOrigin:      false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST without Origin or Referer is rejected",
			method:         http.MethodPost,
			path:           "/api/page",
			setHost:        true,
			hostValue:      "localhost:8080",
			setOrigin:      false,
			setReferer:     false,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Invalid origin",
		},
		{
			name:           "POST with valid Origin succeeds",
			method:         http.MethodPost,
			path:           "/api/page",
			setHost:        true,
			hostValue:      "localhost:8080",
			setOrigin:      true,
			originValue:    "http://localhost:8080",
			expectedStatus: http.StatusCreated, // Document created successfully, CSRF passed
		},
		{
			name:           "POST with invalid Origin is rejected",
			method:         http.MethodPost,
			path:           "/api/page",
			setHost:        true,
			hostValue:      "localhost:8080",
			setOrigin:      true,
			originValue:    "http://evil.com",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Invalid origin",
		},
		{
			name:           "POST with valid Referer succeeds",
			method:         http.MethodPost,
			path:           "/api/page",
			setHost:        true,
			hostValue:      "localhost:8080",
			setReferer:     true,
			refererValue:   "http://localhost:8080/some-page",
			expectedStatus: http.StatusConflict, // Document already exists from previous test, but CSRF passed
		},
		{
			name:           "POST with invalid Referer is rejected",
			method:         http.MethodPost,
			path:           "/api/page",
			setHost:        true,
			hostValue:      "localhost:8080",
			setReferer:     true,
			refererValue:   "http://evil.com/attack",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Invalid origin",
		},
		{
			name:           "PUT with valid Origin succeeds",
			method:         http.MethodPut,
			path:           "/api/page/index.md",
			setHost:        true,
			hostValue:      "localhost:8080",
			setOrigin:      true,
			originValue:    "http://localhost:8080",
			expectedStatus: http.StatusBadRequest, // Bad request due to missing body, not CSRF
		},
		{
			name:           "DELETE with valid Origin succeeds",
			method:         http.MethodDelete,
			path:           "/api/page/test.md",
			setHost:        true,
			hostValue:      "localhost:8080",
			setOrigin:      true,
			originValue:    "http://localhost:8080",
			expectedStatus: http.StatusOK, // Document deleted successfully, CSRF passed
		},
		{
			name:           "localhost and 127.0.0.1 are equivalent",
			method:         http.MethodPost,
			path:           "/api/page",
			setHost:        true,
			hostValue:      "127.0.0.1:8080",
			setOrigin:      true,
			originValue:    "http://localhost:8080",
			expectedStatus: http.StatusCreated, // Document created (was deleted in previous test), CSRF passed
		},
		{
			name:           "healthz endpoint bypasses CSRF",
			method:         http.MethodPost,
			path:           "/healthz",
			setHost:        false,
			setOrigin:      false,
			expectedStatus: http.StatusMethodNotAllowed, // Method not allowed, but passed CSRF
		},
		{
			name:           "static files bypass CSRF",
			method:         http.MethodPost,
			path:           "/static/test.css",
			setHost:        false,
			setOrigin:      false,
			expectedStatus: http.StatusMethodNotAllowed, // Method not allowed, but passed CSRF
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body string
			if tt.method == http.MethodPost && strings.Contains(tt.path, "/api/page") && tt.expectedStatus != http.StatusForbidden {
				body = `{"path":"test.md","content":"# Test\n"}`
			}

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(body))
			if tt.setHost {
				req.Host = tt.hostValue
			}
			if tt.setOrigin {
				req.Header.Set("Origin", tt.originValue)
			}
			if tt.setReferer {
				req.Header.Set("Referer", tt.refererValue)
			}
			if body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d with body: %s",
					tt.expectedStatus, rec.Code, rec.Body.String())
			}

			if tt.expectedError != "" && !strings.Contains(rec.Body.String(), tt.expectedError) {
				t.Errorf("expected error containing %q, got: %s",
					tt.expectedError, rec.Body.String())
			}
		})
	}
}
