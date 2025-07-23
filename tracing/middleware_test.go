package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestDefaultTracingLoggerMiddlewareOptions(t *testing.T) {
	opts := DefaultTracingLoggerMiddlewareOptions()

	require.NotNil(t, opts)
	assert.NotEmpty(t, opts.LogFieldTraceID)
	assert.NotEmpty(t, opts.LogFieldSpanID)
}

func TestNewTracingLoggerMiddleware(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		middleware := NewTracingLoggerMiddleware(nil)
		assert.NotNil(t, middleware)
	})

	t.Run("middleware execution without span context", func(t *testing.T) {
		opts := DefaultTracingLoggerMiddlewareOptions()
		middleware := NewTracingLoggerMiddleware(opts)

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

	t.Run("middleware execution with span context", func(t *testing.T) {
		opts := &TracingLoggerMiddlewareOptions{
			LogFieldTraceID: "trace_id",
			LogFieldSpanID:  "span_id",
		}
		middleware := NewTracingLoggerMiddleware(opts)

		// Create a mock span context
		traceID, _ := trace.TraceIDFromHex("12345678901234567890123456789012")
		spanID, _ := trace.SpanIDFromHex("1234567890123456")
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: traceID,
			SpanID:  spanID,
		})

		handlerCalled := false
		var receivedContext context.Context
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			receivedContext = r.Context()
			w.WriteHeader(http.StatusOK)
		})

		ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.True(t, handlerCalled)
		assert.NotNil(t, receivedContext)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestDefaultRequestIDHeaderMiddlewareOptions(t *testing.T) {
	opts := DefaultRequestIDHeaderMiddlewareOptions()

	require.NotNil(t, opts)
	assert.NotEmpty(t, opts.HeaderName)
	assert.NotNil(t, opts.GeneratorFn)
}

func TestNewRequestIDHeaderMiddleware(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		middleware := NewRequestIDHeaderMiddleware(nil)
		assert.NotNil(t, middleware)
	})

	t.Run("middleware execution with existing request ID", func(t *testing.T) {
		opts := DefaultRequestIDHeaderMiddlewareOptions()
		middleware := NewRequestIDHeaderMiddleware(opts)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(opts.HeaderName, "existing-request-id")
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "existing-request-id", rr.Header().Get(opts.HeaderName))
	})

	t.Run("middleware execution without request ID", func(t *testing.T) {
		opts := DefaultRequestIDHeaderMiddlewareOptions()
		middleware := NewRequestIDHeaderMiddleware(opts)

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
		assert.NotEmpty(t, rr.Header().Get(opts.HeaderName))
	})
}

func TestDefaultRequestIDLoggerMiddlewareOptions(t *testing.T) {
	opts := DefaultRequestIDLoggerMiddlewareOptions()
	assert.NotEmpty(t, opts.LogFieldName)
}

func TestNewRequestIDLoggerMiddleware(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		middleware := NewRequestIDLoggerMiddleware(nil)
		assert.NotNil(t, middleware)
	})

	t.Run("middleware execution", func(t *testing.T) {
		opts := &RequestIDLoggerMiddlewareOptions{
			LogFieldName: "request_id",
		}
		middleware := NewRequestIDLoggerMiddleware(opts)

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

func TestDefaultRequestIDGenerator(t *testing.T) {
	// Test that the generator produces valid UUID-like strings
	id1 := DefaultRequestIDGenerator()
	id2 := DefaultRequestIDGenerator()

	// Should not be empty
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)

	// Should be different
	assert.NotEqual(t, id1, id2)

	// Should be in UUID format (check for hyphens in correct positions)
	parts := strings.Split(id1, "-")
	assert.Len(t, parts, 5)
}
