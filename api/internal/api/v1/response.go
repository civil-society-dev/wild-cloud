package v1

import (
	"encoding/json"
	"net/http"
)

// APIResponse is the standard response envelope for all API endpoints.
// All successful responses wrap data in the Data field.
// Error responses use the Error field.
// Message-only responses use the Message field.
type APIResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// respondJSON writes a JSON response with the given status code.
// This is the low-level helper that all other response helpers use.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// respondMessage writes a success response with only a message (no data).
// Use this for operations that succeed but don't return meaningful data.
func respondMessage(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, APIResponse{Message: message})
}

// respondError writes an error response with the given status code and message.
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, APIResponse{Error: message})
}

// respondAccepted writes a 202 Accepted response for async operations.
// It includes the operation ID so clients can poll for status.
// Note: Returns flat structure (not wrapped in APIResponse) to match CLI expectations.
func respondAccepted(w http.ResponseWriter, operationID, message string) {
	respondJSON(w, http.StatusAccepted, map[string]string{
		"operation_id": operationID,
		"message":      message,
	})
}
