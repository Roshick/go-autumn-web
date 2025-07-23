package logging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultContextLoggerMiddlewareOptions(t *testing.T) {
	opts := DefaultContextLoggerMiddlewareOptions()
	require.NotNil(t, opts)
}

func TestNewContextLoggerMiddleware(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		middleware := NewContextLoggerMiddleware(nil)
		assert.NotNil(t, middleware)
	})

	t.Run("middleware execution", func(t *testing.T) {
		opts := DefaultContextLoggerMiddlewareOptions()
		middleware := NewContextLoggerMiddleware(opts)

		handlerCalled := false
		var receivedContext context.Context
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			receivedContext = r.Context()
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.True(t, handlerCalled)
		assert.NotNil(t, receivedContext)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestDefaultContextCancellationLoggerMiddlewareOptions(t *testing.T) {
	opts := DefaultContextCancellationLoggerMiddlewareOptions()

	require.NotNil(t, opts)
	assert.Equal(t, "default", opts.Description)
}

func TestNewContextCancellationLoggerMiddleware(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		middleware := NewContextCancellationLoggerMiddleware(nil)
		assert.NotNil(t, middleware)
	})

	t.Run("middleware execution", func(t *testing.T) {
		opts := &ContextCancellationLoggerMiddlewareOptions{
			Description: "test middleware",
		}
		middleware := NewContextCancellationLoggerMiddleware(opts)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
