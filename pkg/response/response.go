// Package response provides utilities for writing standardized JSON API responses and decoding JSON request bodies.
package response

import (
	"encoding/json"
	"net/http"
)

// APIResponse defines the standard structure for API responses, including a message and optional data.
type APIResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// WriteJSON writes a standard API response with status code, message, and data.
func WriteJSON(w http.ResponseWriter, status int, message string, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(APIResponse{
		Message: message,
		Data:    data,
	})
}

// DecodeJSON decodes JSON body into the given struct, returns error if fails.
func DecodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
