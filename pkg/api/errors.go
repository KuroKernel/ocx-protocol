package api

import (
	"encoding/json"
	"net/http"
)

type ErrorCode string

const (
	ErrInvalidInput     ErrorCode = "E001"
	ErrInvalidReceipt   ErrorCode = "E002"
	ErrRateLimit        ErrorCode = "E003"
	ErrExecutionFailed  ErrorCode = "E004"
	ErrStorageFailed    ErrorCode = "E005"
	ErrInternalError    ErrorCode = "E006"
	ErrIdempotencyMismatch ErrorCode = "E007"
)

type APIError struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e APIError) Error() string {
	return e.Message
}

func WriteError(w http.ResponseWriter, code ErrorCode, message string, httpStatus int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	
	err := APIError{
		Code:    code,
		Message: message,
	}
	
	json.NewEncoder(w).Encode(err)
}

func WriteErrorWithDetails(w http.ResponseWriter, code ErrorCode, message string, details interface{}, httpStatus int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	
	err := APIError{
		Code:    code,
		Message: message,
		Details: details,
	}
	
	json.NewEncoder(w).Encode(err)
}
