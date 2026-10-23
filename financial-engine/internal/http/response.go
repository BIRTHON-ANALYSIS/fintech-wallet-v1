package http

import (
	"encoding/json"
	"net/http"
	"time"
)

// APIError is the standard error format returned by the engine.
type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// APIResponse is the standard envelope for API responses.
type APIResponse[T any] struct {
	Data      T         `json:"data,omitempty"`
	Error     *APIError `json:"error,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// WriteJSON writes a JSON response with the supplied HTTP status.
func WriteJSON[T any](
	w http.ResponseWriter,
	status int,
	response APIResponse[T],
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// The status has already been written. At this point,
		// we cannot safely replace it with a different response.
		return
	}
}

// WriteError sends a consistent JSON error response.
func WriteError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
	requestID string,
) {
	WriteJSON(w, status, APIResponse[any]{
		Error: &APIError{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		},
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	})
}
