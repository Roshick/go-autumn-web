package metrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
		StatusCode:    200,
		Body:          http.NoBody,
		Header:        make(http.Header),
		ContentLength: 0,
	}, nil
}

func TestDefaultRequestMetricsTransportOptions(t *testing.T) {
	opts := DefaultRequestMetricsTransportOptions()
	require.NotNil(t, opts)
}

func TestNewRequestMetricsTransport(t *testing.T) {
	clientName := "test-client"

	t.Run("with custom round tripper and client name", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		opts := DefaultRequestMetricsTransportOptions()

		transport := NewRequestMetricsTransport(mockRT, clientName, opts)

		require.NotNil(t, transport)
		assert.Equal(t, mockRT, transport.base)
		assert.Equal(t, clientName, transport.clientName)
		assert.Equal(t, opts, transport.opts)

		// Verify metrics are initialized
		assert.NotNil(t, transport.httpClientCounts)
		assert.NotNil(t, transport.httpClientErrCounts)
		assert.NotNil(t, transport.httpClientReqBytes)
		assert.NotNil(t, transport.httpClientResBytes)
	})

	t.Run("with nil round tripper uses default", func(t *testing.T) {
		transport := NewRequestMetricsTransport(nil, clientName, nil)

		require.NotNil(t, transport)
		assert.Equal(t, http.DefaultTransport, transport.base)
		assert.Equal(t, clientName, transport.clientName)
		assert.NotNil(t, transport.opts)
	})

	t.Run("with nil options uses default", func(t *testing.T) {
		mockRT := &MockRoundTripper{}

		transport := NewRequestMetricsTransport(mockRT, clientName, nil)

		require.NotNil(t, transport)
		assert.NotNil(t, transport.opts)
	})

	t.Run("with empty client name", func(t *testing.T) {
		transport := NewRequestMetricsTransport(nil, "", nil)

		require.NotNil(t, transport)
		assert.Equal(t, "", transport.clientName)
	})

	t.Run("client name with hyphens gets sanitized in meter name", func(t *testing.T) {
		transport := NewRequestMetricsTransport(nil, "my-client-name", nil)

		require.NotNil(t, transport)
		assert.Equal(t, "my-client-name", transport.clientName)
		// The meter name sanitization happens in init() but we can't easily test it
		// without more complex OpenTelemetry mocking
	})
}

func TestRequestMetricsTransport_RoundTrip(t *testing.T) {
	t.Run("successful request with content", func(t *testing.T) {
		mockRT := &MockRoundTripper{
			responseToReturn: &http.Response{
				StatusCode:    200,
				Body:          http.NoBody,
				Header:        make(http.Header),
				ContentLength: 150,
			},
		}

		transport := NewRequestMetricsTransport(mockRT, "test-client", nil)

		req := httptest.NewRequest(http.MethodPost, "https://api.localhost/data", strings.NewReader("test body"))
		req.ContentLength = 9 // Length of "test body"

		resp, err := transport.RoundTrip(req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, int64(150), resp.ContentLength)

		// Verify the request was passed through
		require.NotNil(t, mockRT.capturedRequest)
		assert.Equal(t, http.MethodPost, mockRT.capturedRequest.Method)
		assert.Equal(t, "https://api.localhost/data", mockRT.capturedRequest.URL.String())
	})

	t.Run("failed request with error", func(t *testing.T) {
		expectedErr := errors.New("network timeout")
		mockRT := &MockRoundTripper{
			errorToReturn: expectedErr,
		}

		transport := NewRequestMetricsTransport(mockRT, "error-client", nil)

		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/fail", nil)

		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resp)

		// Verify the request was still captured
		require.NotNil(t, mockRT.capturedRequest)
		assert.Equal(t, http.MethodGet, mockRT.capturedRequest.Method)
	})

	t.Run("request with different HTTP methods", func(t *testing.T) {
		methods := []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				mockRT := &MockRoundTripper{
					responseToReturn: &http.Response{
						StatusCode: 200,
						Body:       http.NoBody,
						Header:     make(http.Header),
					},
				}

				transport := NewRequestMetricsTransport(mockRT, "method-test", nil)

				req := httptest.NewRequest(method, "https://api.localhost/test", nil)

				resp, err := transport.RoundTrip(req)

				require.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, 200, resp.StatusCode)

				require.NotNil(t, mockRT.capturedRequest)
				assert.Equal(t, method, mockRT.capturedRequest.Method)
			})
		}
	})

	t.Run("request with different status codes", func(t *testing.T) {
		statusCodes := []int{200, 201, 400, 401, 404, 500, 502}

		for _, statusCode := range statusCodes {
			t.Run(http.StatusText(statusCode), func(t *testing.T) {
				mockRT := &MockRoundTripper{
					responseToReturn: &http.Response{
						StatusCode: statusCode,
						Body:       http.NoBody,
						Header:     make(http.Header),
					},
				}

				transport := NewRequestMetricsTransport(mockRT, "status-test", nil)

				req := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)

				resp, err := transport.RoundTrip(req)

				require.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, statusCode, resp.StatusCode)
			})
		}
	})

	t.Run("preserves request context", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewRequestMetricsTransport(mockRT, "context-test", nil)

		ctx := context.WithValue(context.Background(), "test-key", "test-value")
		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)
		req = req.WithContext(ctx)

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, mockRT.capturedRequest)

		// Context should be preserved
		assert.Equal(t, ctx, mockRT.capturedRequest.Context())
		assert.Equal(t, "test-value", mockRT.capturedRequest.Context().Value("test-key"))
	})

	t.Run("handles zero content length", func(t *testing.T) {
		mockRT := &MockRoundTripper{
			responseToReturn: &http.Response{
				StatusCode:    204, // No Content
				Body:          http.NoBody,
				Header:        make(http.Header),
				ContentLength: 0,
			},
		}

		transport := NewRequestMetricsTransport(mockRT, "zero-content", nil)

		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/empty", nil)
		req.ContentLength = 0

		resp, err := transport.RoundTrip(req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 204, resp.StatusCode)
		assert.Equal(t, int64(0), resp.ContentLength)
	})
}

func TestRequestMetricsTransport_RecordMethods(t *testing.T) {
	t.Run("recordRequest with positive size", func(t *testing.T) {
		transport := NewRequestMetricsTransport(nil, "record-test", nil)
		ctx := context.Background()
		req := httptest.NewRequest(http.MethodPost, "https://api.localhost/test", strings.NewReader("test body"))
		req.ContentLength = 9

		// This test verifies the method doesn't panic
		assert.NotPanics(t, func() {
			transport.recordRequest(ctx, req)
		})
	})

	t.Run("recordRequest with zero size", func(t *testing.T) {
		transport := NewRequestMetricsTransport(nil, "record-test", nil)
		ctx := context.Background()
		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)
		req.ContentLength = 0

		assert.NotPanics(t, func() {
			transport.recordRequest(ctx, req)
		})
	})

	t.Run("recordResponse with success", func(t *testing.T) {
		transport := NewRequestMetricsTransport(nil, "record-test", nil)
		ctx := context.Background()
		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)
		resp := &http.Response{
			StatusCode:    200,
			ContentLength: 50,
		}

		assert.NotPanics(t, func() {
			transport.recordResponse(ctx, req, resp, nil)
		})
	})

	t.Run("recordResponse with error", func(t *testing.T) {
		transport := NewRequestMetricsTransport(nil, "record-test", nil)
		ctx := context.Background()
		req := httptest.NewRequest(http.MethodPost, "https://api.localhost/test", nil)
		err := errors.New("test error")

		assert.NotPanics(t, func() {
			transport.recordResponse(ctx, req, nil, err)
		})
	})

	t.Run("recordResponse without client name", func(t *testing.T) {
		transport := NewRequestMetricsTransport(nil, "", nil)
		ctx := context.Background()
		req := httptest.NewRequest(http.MethodPut, "https://api.localhost/test", nil)
		resp := &http.Response{
			StatusCode:    201,
			ContentLength: 75,
		}

		assert.NotPanics(t, func() {
			transport.recordResponse(ctx, req, resp, nil)
		})
	})
}

func TestRequestMetricsTransport_ImplementsRoundTripper(t *testing.T) {
	transport := NewRequestMetricsTransport(nil, "interface-test", nil)

	// Verify it implements http.RoundTripper interface
	var _ http.RoundTripper = transport
	assert.Implements(t, (*http.RoundTripper)(nil), transport)
}
