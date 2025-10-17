package server

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// respondJSON sends a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", slog.Any("err", err))
	}
}

// decodeJSON reads and decodes JSON from the request body with size limit.
func decodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 4<<20)) // 4MB limit
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is required")
		}
		return err
	}
	// Ensure only one JSON object
	if err := decoder.Decode(new(struct{})); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

// isHTMXRequest checks if the request was made by HTMX.
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") != ""
}

// setHXTrigger sets the HX-Trigger header for HTMX events.
func setHXTrigger(w http.ResponseWriter, events map[string]any) {
	if len(events) == 0 {
		return
	}
	payload, err := json.Marshal(events)
	if err != nil {
		slog.Warn("failed to encode HX-Trigger", slog.Any("err", err))
		return
	}
	w.Header().Set("HX-Trigger", string(payload))
}

// encodeJSON encodes data to JSON string.
func encodeJSON(data any) (string, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
