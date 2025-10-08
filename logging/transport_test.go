package logging

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRoundTripper is a test double for http.RoundTripper
type MockRoundTripper struct {
	capturedRequest  *http.Request
	responseToReturn *http.Response
	errorToReturn    error
	delay            time.Duration
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.capturedRequest = req
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
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

func TestDefaultRequestLoggerTransportOptions(t *testing.T) {
	opts := DefaultRequestLoggerTransportOptions()
	require.NotNil(t, opts)
}

func TestNewRequestLoggerTransport(t *testing.T) {
	t.Run("with custom round tripper", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		opts := DefaultRequestLoggerTransportOptions()

		transport := NewRequestLoggerTransport(mockRT, opts)

		require.NotNil(t, transport)
		assert.Equal(t, mockRT, transport.base)
		assert.Equal(t, opts, transport.opts)
	})

	t.Run("with nil round tripper uses default", func(t *testing.T) {
		transport := NewRequestLoggerTransport(nil, nil)

		require.NotNil(t, transport)
		assert.Equal(t, http.DefaultTransport, transport.base)
		assert.NotNil(t, transport.opts)
	})

	t.Run("with nil options uses default", func(t *testing.T) {
		mockRT := &MockRoundTripper{}

		transport := NewRequestLoggerTransport(mockRT, nil)

		require.NotNil(t, transport)
		assert.NotNil(t, transport.opts)
	})
}

func TestRequestLoggerTransport_RoundTrip(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		mockRT := &MockRoundTripper{
			responseToReturn: &http.Response{
				StatusCode: 200,
				Body:       http.NoBody,
				Header:     make(http.Header),
			},
			delay: 10 * time.Millisecond, // Add small delay to test timing
		}

		transport := NewRequestLoggerTransport(mockRT, nil)

		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/users", nil)

		start := time.Now()
		resp, err := transport.RoundTrip(req)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)

		// Verify the request was passed through
		require.NotNil(t, mockRT.capturedRequest)
		assert.Equal(t, http.MethodGet, mockRT.capturedRequest.Method)
		assert.Equal(t, "https://api.localhost/users", mockRT.capturedRequest.URL.String())

		// Verify timing is reasonable (should be at least the delay we added)
		assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
	})

	t.Run("failed request with error", func(t *testing.T) {
		expectedErr := errors.New("network error")
		mockRT := &MockRoundTripper{
			errorToReturn: expectedErr,
		}

		transport := NewRequestLoggerTransport(mockRT, nil)

		req := httptest.NewRequest(http.MethodPost, "https://api.localhost/data", nil)

		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resp)

		// Verify the request was still captured
		require.NotNil(t, mockRT.capturedRequest)
		assert.Equal(t, http.MethodPost, mockRT.capturedRequest.Method)
	})

	t.Run("request with different status codes", func(t *testing.T) {
		testCases := []struct {
			name       string
			statusCode int
		}{
			{"success 200", 200},
			{"created 201", 201},
			{"bad request 400", 400},
			{"unauthorized 401", 401},
			{"not found 404", 404},
			{"server error 500", 500},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockRT := &MockRoundTripper{
					responseToReturn: &http.Response{
						StatusCode: tc.statusCode,
						Body:       http.NoBody,
						Header:     make(http.Header),
					},
				}

				transport := NewRequestLoggerTransport(mockRT, nil)

				req := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)

				resp, err := transport.RoundTrip(req)

				require.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tc.statusCode, resp.StatusCode)
			})
		}
	})

	t.Run("preserves request context", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewRequestLoggerTransport(mockRT, nil)

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

	t.Run("handles nil response from underlying transport", func(t *testing.T) {
		expectedErr := errors.New("connection failed")
		mockRT := &MockRoundTripper{
			responseToReturn: nil,
			errorToReturn:    expectedErr,
		}

		transport := NewRequestLoggerTransport(mockRT, nil)

		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)

		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resp)
	})
}

func TestRequestLoggerTransport_LogMethods(t *testing.T) {
	t.Run("logResponse logs successful response", func(t *testing.T) {
		transport := NewRequestLoggerTransport(nil, nil)
		ctx := context.Background()
		startTime := time.Now().Add(-100 * time.Millisecond)

		assert.NotPanics(t, func() {
			transport.logResponse(ctx, "POST", "https://api.localhost/data", 201, nil, startTime)
		})
	})

	t.Run("logResponse logs failed response", func(t *testing.T) {
		transport := NewRequestLoggerTransport(nil, nil)
		ctx := context.Background()
		startTime := time.Now().Add(-200 * time.Millisecond)
		err := errors.New("request failed")

		assert.NotPanics(t, func() {
			transport.logResponse(ctx, "DELETE", "https://api.localhost/item/123", 500, err, startTime)
		})
	})
}

func TestRequestLoggerTransport_ImplementsRoundTripper(t *testing.T) {
	transport := NewRequestLoggerTransport(nil, nil)

	// Verify it implements http.RoundTripper interface
	var _ http.RoundTripper = transport
	assert.Implements(t, (*http.RoundTripper)(nil), transport)
}
