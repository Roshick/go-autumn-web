package resiliency

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRoundTripper is a test double for http.RoundTripper
type MockRoundTripper struct {
	capturedRequests []*http.Request
	responseToReturn *http.Response
	errorToReturn    error
	callCount        int
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.capturedRequests = append(m.capturedRequests, req)
	m.callCount++

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

func (m *MockRoundTripper) Reset() {
	m.capturedRequests = nil
	m.callCount = 0
}

func TestDefaultCircuitBreakerTransportOptions(t *testing.T) {
	opts := DefaultCircuitBreakerTransportOptions()

	require.NotNil(t, opts)
	assert.Equal(t, "default", opts.Settings.Name)
	assert.Equal(t, uint32(5), opts.Settings.MaxRequests)
	assert.Equal(t, 60*time.Second, opts.Settings.Interval)
	assert.Equal(t, 60*time.Second, opts.Settings.Timeout)
	assert.NotNil(t, opts.Settings.ReadyToTrip)
}

func TestNewCircuitBreakerTransport(t *testing.T) {
	t.Run("with custom round tripper and options", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		opts := &CircuitBreakerTransportOptions{
			Settings: gobreaker.Settings{
				Name:        "test-breaker",
				MaxRequests: 3,
				Interval:    30 * time.Second,
				Timeout:     30 * time.Second,
			},
		}

		transport := NewCircuitBreakerTransport(mockRT, opts)

		require.NotNil(t, transport)
		assert.Equal(t, mockRT, transport.base)
		assert.NotNil(t, transport.cb)
	})

	t.Run("with nil round tripper uses default", func(t *testing.T) {
		transport := NewCircuitBreakerTransport(nil, nil)

		require.NotNil(t, transport)
		assert.Equal(t, http.DefaultTransport, transport.base)
		assert.NotNil(t, transport.cb)
	})

	t.Run("with nil options uses default", func(t *testing.T) {
		mockRT := &MockRoundTripper{}

		transport := NewCircuitBreakerTransport(mockRT, nil)

		require.NotNil(t, transport)
		assert.NotNil(t, transport.cb)
	})
}

func TestCircuitBreakerTransport_RoundTrip(t *testing.T) {
	t.Run("successful request passes through", func(t *testing.T) {
		mockRT := &MockRoundTripper{
			responseToReturn: &http.Response{
				StatusCode: 200,
				Body:       http.NoBody,
				Header:     make(http.Header),
			},
		}

		transport := NewCircuitBreakerTransport(mockRT, nil)

		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/data", nil)

		resp, err := transport.RoundTrip(req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)

		// Verify the request was passed through
		require.Len(t, mockRT.capturedRequests, 1)
		assert.Equal(t, http.MethodGet, mockRT.capturedRequests[0].Method)
		assert.Equal(t, "https://api.localhost/data", mockRT.capturedRequests[0].URL.String())
	})

	t.Run("failed request returns error", func(t *testing.T) {
		expectedErr := errors.New("network error")
		mockRT := &MockRoundTripper{
			errorToReturn: expectedErr,
		}

		transport := NewCircuitBreakerTransport(mockRT, nil)

		req := httptest.NewRequest(http.MethodPost, "https://api.localhost/fail", nil)

		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resp)

		// Verify the request was attempted
		require.Len(t, mockRT.capturedRequests, 1)
		assert.Equal(t, http.MethodPost, mockRT.capturedRequests[0].Method)
	})

	t.Run("circuit breaker trips after failures", func(t *testing.T) {
		// Configure circuit breaker to trip quickly for testing
		opts := &CircuitBreakerTransportOptions{
			Settings: gobreaker.Settings{
				Name:        "test-breaker",
				MaxRequests: 1,
				Interval:    100 * time.Millisecond,
				Timeout:     100 * time.Millisecond,
				ReadyToTrip: func(counts gobreaker.Counts) bool {
					// Trip after 2 failures
					return counts.TotalFailures >= 2
				},
			},
		}

		mockRT := &MockRoundTripper{
			errorToReturn: errors.New("service unavailable"),
		}

		transport := NewCircuitBreakerTransport(mockRT, opts)

		req1 := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)
		req2 := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)
		req3 := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)

		// First request fails
		_, err1 := transport.RoundTrip(req1)
		assert.Error(t, err1)
		assert.Equal(t, 1, mockRT.callCount)

		// Second request fails and trips the circuit breaker
		_, err2 := transport.RoundTrip(req2)
		assert.Error(t, err2)
		assert.Equal(t, 2, mockRT.callCount)

		// Third request should be blocked by circuit breaker
		_, err3 := transport.RoundTrip(req3)
		assert.Error(t, err3)
		// Should still be 2 calls to underlying transport (third was blocked)
		assert.Equal(t, 2, mockRT.callCount)

		// Error should indicate circuit breaker is open
		assert.Contains(t, err3.Error(), "circuit breaker is open")
	})

	t.Run("circuit breaker allows requests when closed", func(t *testing.T) {
		mockRT := &MockRoundTripper{
			responseToReturn: &http.Response{
				StatusCode: 200,
				Body:       http.NoBody,
				Header:     make(http.Header),
			},
		}

		transport := NewCircuitBreakerTransport(mockRT, nil)

		// Make multiple successful requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)
			resp, err := transport.RoundTrip(req)

			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, 200, resp.StatusCode)
		}

		// All requests should have been passed through
		assert.Equal(t, 5, mockRT.callCount)
	})

	t.Run("preserves request context", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewCircuitBreakerTransport(mockRT, nil)

		req := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)
		originalCtx := req.Context()

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.Len(t, mockRT.capturedRequests, 1)

		// Context should be preserved
		assert.Equal(t, originalCtx, mockRT.capturedRequests[0].Context())
	})

	t.Run("circuit breaker recovery after timeout", func(t *testing.T) {
		// Configure circuit breaker with very short timeout for testing
		opts := &CircuitBreakerTransportOptions{
			Settings: gobreaker.Settings{
				Name:        "recovery-test",
				MaxRequests: 1,
				Interval:    10 * time.Millisecond,
				Timeout:     20 * time.Millisecond, // Very short timeout
				ReadyToTrip: func(counts gobreaker.Counts) bool {
					return counts.TotalFailures >= 1
				},
			},
		}

		mockRT := &MockRoundTripper{
			errorToReturn: errors.New("initial failure"),
		}

		transport := NewCircuitBreakerTransport(mockRT, opts)

		// First request fails and trips circuit breaker
		req1 := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)
		_, err1 := transport.RoundTrip(req1)
		assert.Error(t, err1)
		assert.Equal(t, 1, mockRT.callCount)

		// Second request should be blocked
		req2 := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)
		_, err2 := transport.RoundTrip(req2)
		assert.Error(t, err2)
		assert.Contains(t, err2.Error(), "circuit breaker is open")
		assert.Equal(t, 1, mockRT.callCount) // Still blocked

		// Wait for circuit breaker timeout
		time.Sleep(25 * time.Millisecond)

		// Configure mock to succeed now
		mockRT.errorToReturn = nil
		mockRT.responseToReturn = &http.Response{
			StatusCode: 200,
			Body:       http.NoBody,
			Header:     make(http.Header),
		}

		// Third request should succeed (circuit breaker half-open)
		req3 := httptest.NewRequest(http.MethodGet, "https://api.localhost/test", nil)
		resp3, err3 := transport.RoundTrip(req3)

		require.NoError(t, err3)
		assert.NotNil(t, resp3)
		assert.Equal(t, 200, resp3.StatusCode)
		assert.Equal(t, 2, mockRT.callCount) // Now allowed through
	})
}

func TestCircuitBreakerTransport_ImplementsRoundTripper(t *testing.T) {
	transport := NewCircuitBreakerTransport(nil, nil)

	// Verify it implements http.RoundTripper interface
	var _ http.RoundTripper = transport
	assert.Implements(t, (*http.RoundTripper)(nil), transport)
}
