package errors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorResponseRender(t *testing.T) {
	response := &ErrorResponse{
		HTTPStatusCode: http.StatusTeapot,
		StatusText:     "I'm a teapot",
		Message:        "short and stout",
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	err := render.Render(rr, req, response)

	require.NoError(t, err)
	assert.Equal(t, http.StatusTeapot, rr.Code)
	assert.JSONEq(t, `{"status":"I'm a teapot","message":"short and stout"}`, rr.Body.String())
}

func TestNewBadRequestResponse(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		response := NewBadRequestResponse("custom message")

		require.NotNil(t, response)
		assert.Equal(t, http.StatusBadRequest, response.HTTPStatusCode)
		assert.Equal(t, "Bad Request", response.StatusText)
		assert.Equal(t, "custom message", response.Message)
	})

	t.Run("with empty message", func(t *testing.T) {
		response := NewBadRequestResponse("")

		require.NotNil(t, response)
		assert.Equal(t, "Invalid request", response.Message)
	})
}

func TestNewUnauthorizedResponse(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		response := NewUnauthorizedResponse("custom message")

		require.NotNil(t, response)
		assert.Equal(t, http.StatusUnauthorized, response.HTTPStatusCode)
		assert.Equal(t, "Unauthorized", response.StatusText)
		assert.Equal(t, "custom message", response.Message)
	})

	t.Run("with empty message", func(t *testing.T) {
		response := NewUnauthorizedResponse("")

		require.NotNil(t, response)
		assert.Equal(t, "Authentication required", response.Message)
	})
}

func TestNewForbiddenResponse(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		response := NewForbiddenResponse("custom message")

		require.NotNil(t, response)
		assert.Equal(t, http.StatusForbidden, response.HTTPStatusCode)
		assert.Equal(t, "Forbidden", response.StatusText)
		assert.Equal(t, "custom message", response.Message)
	})

	t.Run("with empty message", func(t *testing.T) {
		response := NewForbiddenResponse("")

		require.NotNil(t, response)
		assert.Equal(t, "Access denied", response.Message)
	})
}

func TestNewRequestTimeoutResponse(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		response := NewRequestTimeoutResponse("custom message")

		require.NotNil(t, response)
		assert.Equal(t, http.StatusRequestTimeout, response.HTTPStatusCode)
		assert.Equal(t, "Request Timeout", response.StatusText)
		assert.Equal(t, "custom message", response.Message)
	})

	t.Run("with empty message", func(t *testing.T) {
		response := NewRequestTimeoutResponse("")

		require.NotNil(t, response)
		assert.Equal(t, "Request processing timeout", response.Message)
	})
}

func TestNewPreconditionRequiredResponse(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		response := NewPreconditionRequiredResponse("custom message")

		require.NotNil(t, response)
		assert.Equal(t, http.StatusPreconditionRequired, response.HTTPStatusCode)
		assert.Equal(t, "Precondition Required", response.StatusText)
		assert.Equal(t, "custom message", response.Message)
	})

	t.Run("with empty message", func(t *testing.T) {
		response := NewPreconditionRequiredResponse("")

		require.NotNil(t, response)
		assert.Equal(t, "Missing required precondition", response.Message)
	})
}

func TestNewInternalServerErrorResponse(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		response := NewInternalServerErrorResponse("custom message")

		require.NotNil(t, response)
		assert.Equal(t, http.StatusInternalServerError, response.HTTPStatusCode)
		assert.Equal(t, "Internal Server Error", response.StatusText)
		assert.Equal(t, "custom message", response.Message)
	})

	t.Run("with empty message", func(t *testing.T) {
		response := NewInternalServerErrorResponse("")

		require.NotNil(t, response)
		assert.Equal(t, "An unexpected error occurred", response.Message)
	})
}

func TestConvenienceResponses(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		response := NewInvalidRequestBodyResponse()

		require.NotNil(t, response)
		assert.Equal(t, http.StatusBadRequest, response.HTTPStatusCode)
		assert.Equal(t, "Invalid request body", response.Message)
	})

	t.Run("missing required header", func(t *testing.T) {
		response := NewMissingRequiredHeaderResponse()

		require.NotNil(t, response)
		assert.Equal(t, http.StatusPreconditionRequired, response.HTTPStatusCode)
		assert.Equal(t, "Missing required header", response.Message)
	})

	t.Run("authentication required", func(t *testing.T) {
		response := NewAuthenticationRequiredResponse()

		require.NotNil(t, response)
		assert.Equal(t, http.StatusUnauthorized, response.HTTPStatusCode)
		assert.Equal(t, "Authentication required", response.Message)
	})

	t.Run("access denied", func(t *testing.T) {
		response := NewAccessDeniedResponse()

		require.NotNil(t, response)
		assert.Equal(t, http.StatusForbidden, response.HTTPStatusCode)
		assert.Equal(t, "Access denied", response.Message)
	})

	t.Run("timeout", func(t *testing.T) {
		response := NewTimeoutResponse()

		require.NotNil(t, response)
		assert.Equal(t, http.StatusRequestTimeout, response.HTTPStatusCode)
		assert.Equal(t, "Request processing timeout", response.Message)
	})

	t.Run("panic recovery", func(t *testing.T) {
		response := NewPanicRecoveryResponse()

		require.NotNil(t, response)
		assert.Equal(t, http.StatusInternalServerError, response.HTTPStatusCode)
		assert.Equal(t, "An unexpected error occurred", response.Message)
	})
}
