package errors

import (
	"github.com/go-chi/render"
	"net/http"
)

// Base error response structure
type ErrorResponse struct {
	HTTPStatusCode int    `json:"-"`
	StatusText     string `json:"status"`
	Message        string `json:"message"`
}

func (e *ErrorResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

// Common HTTP error responses

// BadRequestResponse represents a 400 Bad Request error
type BadRequestResponse struct {
	ErrorResponse
}

func NewBadRequestResponse(message string) *BadRequestResponse {
	if message == "" {
		message = "Invalid request"
	}
	return &BadRequestResponse{
		ErrorResponse: ErrorResponse{
			HTTPStatusCode: http.StatusBadRequest,
			StatusText:     "Bad Request",
			Message:        message,
		},
	}
}

// UnauthorizedResponse represents a 401 Unauthorized error
type UnauthorizedResponse struct {
	ErrorResponse
}

func NewUnauthorizedResponse(message string) *UnauthorizedResponse {
	if message == "" {
		message = "Authentication required"
	}
	return &UnauthorizedResponse{
		ErrorResponse: ErrorResponse{
			HTTPStatusCode: http.StatusUnauthorized,
			StatusText:     "Unauthorized",
			Message:        message,
		},
	}
}

// ForbiddenResponse represents a 403 Forbidden error
type ForbiddenResponse struct {
	ErrorResponse
}

func NewForbiddenResponse(message string) *ForbiddenResponse {
	if message == "" {
		message = "Access denied"
	}
	return &ForbiddenResponse{
		ErrorResponse: ErrorResponse{
			HTTPStatusCode: http.StatusForbidden,
			StatusText:     "Forbidden",
			Message:        message,
		},
	}
}

// RequestTimeoutResponse represents a 408 Request Timeout error
type RequestTimeoutResponse struct {
	ErrorResponse
}

func NewRequestTimeoutResponse(message string) *RequestTimeoutResponse {
	if message == "" {
		message = "Request processing timeout"
	}
	return &RequestTimeoutResponse{
		ErrorResponse: ErrorResponse{
			HTTPStatusCode: http.StatusRequestTimeout,
			StatusText:     "Request Timeout",
			Message:        message,
		},
	}
}

// PreconditionRequiredResponse represents a 428 Precondition Required error
type PreconditionRequiredResponse struct {
	ErrorResponse
}

func NewPreconditionRequiredResponse(message string) *PreconditionRequiredResponse {
	if message == "" {
		message = "Missing required precondition"
	}
	return &PreconditionRequiredResponse{
		ErrorResponse: ErrorResponse{
			HTTPStatusCode: http.StatusPreconditionRequired,
			StatusText:     "Precondition Required",
			Message:        message,
		},
	}
}

// InternalServerErrorResponse represents a 500 Internal Server Error
type InternalServerErrorResponse struct {
	ErrorResponse
}

func NewInternalServerErrorResponse(message string) *InternalServerErrorResponse {
	if message == "" {
		message = "An unexpected error occurred"
	}
	return &InternalServerErrorResponse{
		ErrorResponse: ErrorResponse{
			HTTPStatusCode: http.StatusInternalServerError,
			StatusText:     "Internal Server Error",
			Message:        message,
		},
	}
}

// Convenience functions for common use cases

func NewInvalidRequestBodyResponse() *BadRequestResponse {
	return NewBadRequestResponse("Invalid request body")
}

func NewMissingRequiredHeaderResponse() *PreconditionRequiredResponse {
	return NewPreconditionRequiredResponse("Missing required header")
}

func NewAuthenticationRequiredResponse() *UnauthorizedResponse {
	return NewUnauthorizedResponse("Authentication required")
}

func NewAccessDeniedResponse() *ForbiddenResponse {
	return NewForbiddenResponse("Access denied")
}

func NewTimeoutResponse() *RequestTimeoutResponse {
	return NewRequestTimeoutResponse("Request processing timeout")
}

func NewPanicRecoveryResponse() *InternalServerErrorResponse {
	return NewInternalServerErrorResponse("An unexpected error occurred")
}
