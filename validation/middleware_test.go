package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestRequestBody struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func TestRequestBodyFromContext(t *testing.T) {
	ctx := context.Background()
	testBody := TestRequestBody{Name: "John", Email: "john@localhost"}

	// Test with value in context
	ctxWithValue := context.WithValue(ctx, requestBodyContextKey[TestRequestBody]{}, testBody)
	result := RequestBodyFromContext[TestRequestBody](ctxWithValue)

	assert.Equal(t, testBody, result)
}

func TestDefaultContextRequestBodyMiddlewareOptions(t *testing.T) {
	opts := DefaultContextRequestBodyMiddlewareOptions()

	require.NotNil(t, opts)
	assert.NotNil(t, opts.ErrorResponse)
}

func TestNewContextRequestBodyMiddleware(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		middleware := NewContextRequestBodyMiddleware[TestRequestBody](nil)
		assert.NotNil(t, middleware)
	})

	t.Run("valid JSON body", func(t *testing.T) {
		opts := DefaultContextRequestBodyMiddlewareOptions()
		middleware := NewContextRequestBodyMiddleware[TestRequestBody](opts)

		testBody := TestRequestBody{Name: "John", Email: "john@localhost"}
		bodyBytes, _ := json.Marshal(testBody)

		handlerCalled := false
		var receivedBody TestRequestBody
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			receivedBody = RequestBodyFromContext[TestRequestBody](r.Context())
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.True(t, handlerCalled)
		assert.Equal(t, testBody, receivedBody)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		opts := DefaultContextRequestBodyMiddlewareOptions()
		middleware := NewContextRequestBodyMiddleware[TestRequestBody](opts)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("empty body", func(t *testing.T) {
		opts := DefaultContextRequestBodyMiddlewareOptions()
		middleware := NewContextRequestBodyMiddleware[TestRequestBody](opts)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestDefaultRequiredHeaderMiddlewareOptions(t *testing.T) {
	opts := DefaultRequiredHeaderMiddlewareOptions()

	require.NotNil(t, opts)
	assert.NotNil(t, opts.ErrorResponse)
}

func TestNewRequiredHeaderMiddleware(t *testing.T) {
	headerName := "X-Required-Header"

	t.Run("with nil options", func(t *testing.T) {
		middleware := NewRequiredHeaderMiddleware(headerName, nil)
		assert.NotNil(t, middleware)
	})

	t.Run("header present", func(t *testing.T) {
		opts := DefaultRequiredHeaderMiddlewareOptions()
		middleware := NewRequiredHeaderMiddleware(headerName, opts)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(headerName, "header-value")
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("header missing", func(t *testing.T) {
		opts := DefaultRequiredHeaderMiddlewareOptions()
		middleware := NewRequiredHeaderMiddleware(headerName, opts)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusPreconditionRequired, rr.Code) // Changed from StatusBadRequest
	})

	t.Run("header empty", func(t *testing.T) {
		opts := DefaultRequiredHeaderMiddlewareOptions()
		middleware := NewRequiredHeaderMiddleware(headerName, opts)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(headerName, "")
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusPreconditionRequired, rr.Code) // Changed from StatusBadRequest
	})
}
