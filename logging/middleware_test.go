package logging

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	slogging "github.com/Roshick/go-autumn-slog"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestLogger(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	previousLogger := aulogging.Logger
	aulogging.Logger = slogging.New().WithLogger(logger)
	t.Cleanup(func() {
		aulogging.Logger = previousLogger
	})

	return buf
}

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

	t.Run("context already cancelled", func(t *testing.T) {
		buf := setupTestLogger(t)

		middleware := NewContextCancellationLoggerMiddleware(nil)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		ctx, cancel := context.WithCancelCause(t.Context())
		cancel(errors.New("cancellation cause"))

		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.False(t, handlerCalled)
		assert.Contains(t, buf.String(), "is already cancelled")
		assert.Contains(t, buf.String(), "cancellation cause")
	})

	t.Run("context cancelled during request processing", func(t *testing.T) {
		buf := setupTestLogger(t)

		middleware := NewContextCancellationLoggerMiddleware(nil)

		ctx, cancel := context.WithCancelCause(t.Context())

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cancel(errors.New("cancellation cause"))
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, buf.String(), "was cancelled during request processing")
		assert.Contains(t, buf.String(), "cancellation cause")
	})
}

func TestDefaultRequestLoggerMiddlewareOptions(t *testing.T) {
	opts := DefaultRequestLoggerMiddlewareOptions()

	require.NotNil(t, opts)
	assert.Equal(t, 500, opts.WarningStatusCodeThreshold)
}

func TestNewRequestLoggerMiddleware(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		middleware := NewRequestLoggerMiddleware(nil)
		assert.NotNil(t, middleware)
	})

	t.Run("logs successful response as info", func(t *testing.T) {
		buf := setupTestLogger(t)

		middleware := NewRequestLoggerMiddleware(nil)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/some/path", nil)
		ctx := slogging.ContextWithLogger(req.Context(), slog.New(slog.NewTextHandler(buf, nil)))
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req.WithContext(ctx))

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, buf.String(), "level=INFO")
		assert.Contains(t, buf.String(), "response GET /some/path -> 200")
	})

	t.Run("logs server error response as warning", func(t *testing.T) {
		buf := setupTestLogger(t)

		middleware := NewRequestLoggerMiddleware(nil)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		req := httptest.NewRequest(http.MethodGet, "/some/path", nil)
		ctx := slogging.ContextWithLogger(req.Context(), slog.New(slog.NewTextHandler(buf, nil)))
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req.WithContext(ctx))

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, buf.String(), "level=WARN")
		assert.Contains(t, buf.String(), "response GET /some/path -> 500")
	})

	t.Run("does not log without context logger", func(t *testing.T) {
		buf := setupTestLogger(t)

		middleware := NewRequestLoggerMiddleware(nil)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Empty(t, buf.String())
	})
}
