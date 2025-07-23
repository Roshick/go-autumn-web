package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRoundTripper is a test double for http.RoundTripper
type MockRoundTripper struct {
	capturedRequest  *http.Request
	responseToReturn *http.Response
	errorToReturn    error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.capturedRequest = req
	if m.errorToReturn != nil {
		return nil, m.errorToReturn
	}
	if m.responseToReturn != nil {
		return m.responseToReturn, nil
	}
	// Default response
	return &http.Response{
		StatusCode: 200,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}, nil
}

func TestDefaultRequestIDHeaderTransportOptions(t *testing.T) {
	opts := DefaultRequestIDHeaderTransportOptions()

	require.NotNil(t, opts)
	assert.NotEmpty(t, opts.HeaderName)
}

func TestNewRequestIDHeaderTransport(t *testing.T) {
	t.Run("with custom round tripper and options", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		opts := &RequestIDHeaderTransportOptions{
			HeaderName: "X-Custom-Request-ID",
		}

		transport := NewRequestIDHeaderTransport(mockRT, opts)

		require.NotNil(t, transport)
		assert.Equal(t, mockRT, transport.base)
		assert.Equal(t, opts, transport.opts)
		assert.Equal(t, "X-Custom-Request-ID", transport.opts.HeaderName)
	})

	t.Run("with nil round tripper uses default", func(t *testing.T) {
		transport := NewRequestIDHeaderTransport(nil, nil)

		require.NotNil(t, transport)
		assert.Equal(t, http.DefaultTransport, transport.base)
		assert.NotNil(t, transport.opts)
	})

	t.Run("with nil options uses default", func(t *testing.T) {
		mockRT := &MockRoundTripper{}

		transport := NewRequestIDHeaderTransport(mockRT, nil)

		require.NotNil(t, transport)
		assert.NotNil(t, transport.opts)
		assert.NotEmpty(t, transport.opts.HeaderName)
	})
}

func TestRequestIDHeaderTransport_RoundTrip(t *testing.T) {
	t.Run("adds request ID header when present in context", func(t *testing.T) {
		mockRT := &MockRoundTripper{
			responseToReturn: &http.Response{
				StatusCode: 200,
				Body:       http.NoBody,
				Header:     make(http.Header),
			},
		}

		opts := &RequestIDHeaderTransportOptions{
			HeaderName: "X-Request-ID",
		}
		transport := NewRequestIDHeaderTransport(mockRT, opts)

		// Create context with request ID
		ctx := ContextWithRequestID(context.Background(), "test-request-id-123")
		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/data", nil)
		req = req.WithContext(ctx)

		resp, err := transport.RoundTrip(req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)

		// Verify the request ID header was added
		require.NotNil(t, mockRT.capturedRequest)
		assert.Equal(t, "test-request-id-123", mockRT.capturedRequest.Header.Get("X-Request-ID"))
	})

	t.Run("does not add header when request ID not in context", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewRequestIDHeaderTransport(mockRT, nil)

		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/data", nil)

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, mockRT.capturedRequest)

		// Header should not be set
		assert.Empty(t, mockRT.capturedRequest.Header.Get(transport.opts.HeaderName))
	})

	t.Run("does not add header when request ID is empty", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewRequestIDHeaderTransport(mockRT, nil)

		// Create context with empty request ID
		ctx := ContextWithRequestID(context.Background(), "")
		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/data", nil)
		req = req.WithContext(ctx)

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, mockRT.capturedRequest)

		// Header should not be set for empty request ID
		assert.Empty(t, mockRT.capturedRequest.Header.Get(transport.opts.HeaderName))
	})

	t.Run("uses custom header name", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		opts := &RequestIDHeaderTransportOptions{
			HeaderName: "X-Trace-ID",
		}
		transport := NewRequestIDHeaderTransport(mockRT, opts)

		ctx := ContextWithRequestID(context.Background(), "custom-trace-id")
		req := httptest.NewRequest(http.MethodPost, "https://api.localhost/submit", nil)
		req = req.WithContext(ctx)

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, mockRT.capturedRequest)

		// Should use custom header name
		assert.Equal(t, "custom-trace-id", mockRT.capturedRequest.Header.Get("X-Trace-ID"))
		assert.Empty(t, mockRT.capturedRequest.Header.Get("X-Request-ID")) // Default header should not be set
	})

	t.Run("preserves existing headers", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewRequestIDHeaderTransport(mockRT, nil)

		ctx := ContextWithRequestID(context.Background(), "preserve-test-id")
		req := httptest.NewRequest(http.MethodPut, "https://api.localhost/update", nil)
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer token123")

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, mockRT.capturedRequest)

		// Existing headers should be preserved
		assert.Equal(t, "application/json", mockRT.capturedRequest.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer token123", mockRT.capturedRequest.Header.Get("Authorization"))

		// Request ID header should be added
		assert.Equal(t, "preserve-test-id", mockRT.capturedRequest.Header.Get(transport.opts.HeaderName))
	})

	t.Run("overwrites existing request ID header", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		opts := &RequestIDHeaderTransportOptions{
			HeaderName: "X-Request-ID",
		}
		transport := NewRequestIDHeaderTransport(mockRT, opts)

		ctx := ContextWithRequestID(context.Background(), "context-request-id")
		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/data", nil)
		req = req.WithContext(ctx)
		req.Header.Set("X-Request-ID", "original-header-value")

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, mockRT.capturedRequest)

		// Context value should overwrite existing header
		assert.Equal(t, "context-request-id", mockRT.capturedRequest.Header.Get("X-Request-ID"))
	})

	t.Run("propagates errors from underlying transport", func(t *testing.T) {
		expectedErr := assert.AnError
		mockRT := &MockRoundTripper{
			errorToReturn: expectedErr,
		}
		transport := NewRequestIDHeaderTransport(mockRT, nil)

		ctx := ContextWithRequestID(context.Background(), "error-test-id")
		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/data", nil)
		req = req.WithContext(ctx)

		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resp)

		// Request should still have been processed
		require.NotNil(t, mockRT.capturedRequest)
		assert.Equal(t, "error-test-id", mockRT.capturedRequest.Header.Get(transport.opts.HeaderName))
	})

	t.Run("preserves request context", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewRequestIDHeaderTransport(mockRT, nil)

		ctx := context.WithValue(context.Background(), "test-key", "test-value")
		ctx = ContextWithRequestID(ctx, "context-test-id")
		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/data", nil)
		req = req.WithContext(ctx)

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, mockRT.capturedRequest)

		// Context should be preserved
		assert.Equal(t, ctx, mockRT.capturedRequest.Context())
		assert.Equal(t, "test-value", mockRT.capturedRequest.Context().Value("test-key"))

		// Request ID should still be extractable from context
		requestID := RequestIDFromContext(mockRT.capturedRequest.Context())
		require.NotNil(t, requestID)
		assert.Equal(t, "context-test-id", *requestID)
	})
}

func TestRequestIDHeaderTransport_ImplementsRoundTripper(t *testing.T) {
	transport := NewRequestIDHeaderTransport(nil, nil)

	// Verify it implements http.RoundTripper interface
	var _ http.RoundTripper = transport
	assert.Implements(t, (*http.RoundTripper)(nil), transport)
}
