package tokka

import (
	"encoding/json"
	"net/http"
)

type JSONError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

const (
	ErrorCodeRateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
	ErrorCodePayloadTooLarge     = "PAYLOAD_TOO_LARGE"
	ErrorCodeUpstreamUnavailable = "UPSTREAM_UNAVAILABLE"
	ErrorCodeUpstreamError       = "UPSTREAM_ERROR"
	ErrorCodeInternal            = "INTERNAL"
)

func WriteError(w http.ResponseWriter, code, message, requestID string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	jsonError := JSONError{
		Code:      code,
		Message:   message,
		RequestID: requestID,
	}

	if err := json.NewEncoder(w).Encode(jsonError); err != nil {
		// Fallback on error.
		http.Error(w, http.StatusText(status), status)
	}
}
