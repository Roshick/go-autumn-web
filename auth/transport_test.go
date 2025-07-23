package auth

import (
	"encoding/base64"
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

func TestDefaultBasicAuthTransportOptions(t *testing.T) {
	opts := DefaultBasicAuthTransportOptions()
	require.NotNil(t, opts)
}

func TestNewBasicAuthTransport(t *testing.T) {
	username := "testuser"
	password := "testpass"

	t.Run("with custom round tripper", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		opts := DefaultBasicAuthTransportOptions()

		transport := NewBasicAuthTransport(mockRT, username, password, opts)

		require.NotNil(t, transport)
		assert.Equal(t, mockRT, transport.base)
		assert.Equal(t, username, transport.username)
		assert.Equal(t, password, transport.password)
		assert.Equal(t, opts, transport.opts)
	})

	t.Run("with nil round tripper uses default", func(t *testing.T) {
		transport := NewBasicAuthTransport(nil, username, password, nil)

		require.NotNil(t, transport)
		assert.Equal(t, http.DefaultTransport, transport.base)
		assert.Equal(t, username, transport.username)
		assert.Equal(t, password, transport.password)
		assert.NotNil(t, transport.opts)
	})

	t.Run("with nil options uses default", func(t *testing.T) {
		mockRT := &MockRoundTripper{}

		transport := NewBasicAuthTransport(mockRT, username, password, nil)

		require.NotNil(t, transport)
		assert.NotNil(t, transport.opts)
	})
}

func TestBasicAuthTransport_RoundTrip(t *testing.T) {
	username := "testuser"
	password := "testpass"
	expectedAuth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	t.Run("adds basic auth header", func(t *testing.T) {
		mockRT := &MockRoundTripper{
			responseToReturn: &http.Response{
				StatusCode: 200,
				Body:       http.NoBody,
				Header:     make(http.Header),
			},
		}

		transport := NewBasicAuthTransport(mockRT, username, password, nil)

		req := httptest.NewRequest(http.MethodGet, "https://localhost/api", nil)

		resp, err := transport.RoundTrip(req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)

		// Verify the auth header was added to the request
		require.NotNil(t, mockRT.capturedRequest)
		authHeader := mockRT.capturedRequest.Header.Get("Authorization")
		assert.Equal(t, "Basic "+expectedAuth, authHeader)
	})

	t.Run("clones request without modifying original", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewBasicAuthTransport(mockRT, username, password, nil)

		originalReq := httptest.NewRequest(http.MethodGet, "https://localhost/api", nil)
		originalAuthHeader := originalReq.Header.Get("Authorization")

		_, err := transport.RoundTrip(originalReq)

		require.NoError(t, err)

		// Original request should not be modified
		assert.Equal(t, originalAuthHeader, originalReq.Header.Get("Authorization"))

		// But the cloned request should have the auth header
		require.NotNil(t, mockRT.capturedRequest)
		assert.Equal(t, "Basic "+expectedAuth, mockRT.capturedRequest.Header.Get("Authorization"))

		// Verify it's a different request object
		assert.NotEqual(t, originalReq, mockRT.capturedRequest)
	})

	t.Run("preserves existing headers", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewBasicAuthTransport(mockRT, username, password, nil)

		req := httptest.NewRequest(http.MethodPost, "https://localhost/api", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Custom-Header", "custom-value")

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, mockRT.capturedRequest)

		// Verify existing headers are preserved
		assert.Equal(t, "application/json", mockRT.capturedRequest.Header.Get("Content-Type"))
		assert.Equal(t, "custom-value", mockRT.capturedRequest.Header.Get("X-Custom-Header"))

		// And auth header is added
		assert.Equal(t, "Basic "+expectedAuth, mockRT.capturedRequest.Header.Get("Authorization"))
	})

	t.Run("overwrites existing authorization header", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewBasicAuthTransport(mockRT, username, password, nil)

		req := httptest.NewRequest(http.MethodGet, "https://localhost/api", nil)
		req.Header.Set("Authorization", "Bearer some-token")

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, mockRT.capturedRequest)

		// Auth header should be overwritten
		assert.Equal(t, "Basic "+expectedAuth, mockRT.capturedRequest.Header.Get("Authorization"))
	})

	t.Run("propagates errors from underlying transport", func(t *testing.T) {
		expectedErr := assert.AnError
		mockRT := &MockRoundTripper{
			errorToReturn: expectedErr,
		}
		transport := NewBasicAuthTransport(mockRT, username, password, nil)

		req := httptest.NewRequest(http.MethodGet, "https://localhost/api", nil)

		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resp)
	})

	t.Run("preserves request context", func(t *testing.T) {
		mockRT := &MockRoundTripper{}
		transport := NewBasicAuthTransport(mockRT, username, password, nil)

		req := httptest.NewRequest(http.MethodGet, "https://localhost/api", nil)
		originalCtx := req.Context()

		_, err := transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, mockRT.capturedRequest)

		// Context should be preserved
		assert.Equal(t, originalCtx, mockRT.capturedRequest.Context())
	})
}

func TestBasicAuthTransport_ImplementsRoundTripper(t *testing.T) {
	transport := NewBasicAuthTransport(nil, "user", "pass", nil)

	// Verify it implements http.RoundTripper interface
	var _ http.RoundTripper = transport
	assert.Implements(t, (*http.RoundTripper)(nil), transport)
}
