package testutils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMockInteractionTransportOptions(t *testing.T) {
	opts := DefaultMockInteractionTransportOptions()

	require.NotNil(t, opts)
	assert.Equal(t, Exact, opts.Algorithm)
}

func TestNewMockInteractionRoundTripper(t *testing.T) {
	t.Run("with custom options", func(t *testing.T) {
		opts := &MockInteractionTransportOptions{
			Algorithm: FirstMatch,
		}

		transport := NewMockInteractionTransport(t, opts)

		require.NotNil(t, transport)
		assert.Equal(t, t, transport.t)
		assert.Equal(t, opts, transport.opts)
		assert.NotNil(t, transport.expectedInteractions)
		assert.Len(t, transport.expectedInteractions, 0)
	})

	t.Run("with nil options uses default", func(t *testing.T) {
		transport := NewMockInteractionTransport(t, nil)

		require.NotNil(t, transport)
		assert.NotNil(t, transport.opts)
		assert.Equal(t, Exact, transport.opts.Algorithm)
	})
}

func TestMockInteractionTransport_ExpectRequest(t *testing.T) {
	transport := NewMockInteractionTransport(t, nil)

	testReq := TestRequest{
		Method: "GET",
		URL:    "https://api.localhost/users",
	}

	interaction := transport.ExpectRequest(testReq)

	require.NotNil(t, interaction)
	assert.Equal(t, testReq, interaction.request)
	assert.False(t, interaction.ignoreQueryParams)
	assert.Len(t, transport.expectedInteractions, 1)
}

func TestMockInteractionTransport_Reset(t *testing.T) {
	transport := NewMockInteractionTransport(t, nil)

	// Add some interactions
	transport.ExpectRequest(TestRequest{Method: "GET", URL: "https://api.localhost/1"})
	transport.ExpectRequest(TestRequest{Method: "POST", URL: "https://api.localhost/2"})

	assert.Len(t, transport.expectedInteractions, 2)

	transport.Reset()

	assert.Len(t, transport.expectedInteractions, 0)
}

func TestExpectedInteraction_WillReturnResponse(t *testing.T) {
	transport := NewMockInteractionTransport(t, nil)

	testReq := TestRequest{Method: "GET", URL: "https://api.localhost/test"}
	testResp := &TestResponse{
		Status: 200,
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   map[string]string{"message": "success"},
	}

	interaction := transport.ExpectRequest(testReq)
	interaction.WillReturnResponse(testResp)

	assert.Equal(t, testResp, interaction.response)
}

func TestExpectedInteraction_IgnoreQueryParams(t *testing.T) {
	transport := NewMockInteractionTransport(t, nil)

	testReq := TestRequest{Method: "GET", URL: "https://api.localhost/test"}
	interaction := transport.ExpectRequest(testReq)

	// Initially should be false
	assert.False(t, interaction.ignoreQueryParams)

	// Set to true
	result := interaction.IgnoreQueryParams(true)

	assert.True(t, interaction.ignoreQueryParams)
	assert.Equal(t, interaction, result) // Should return self for chaining

	// Set to false
	interaction.IgnoreQueryParams(false)
	assert.False(t, interaction.ignoreQueryParams)
}

func TestExpectedInteraction_extractBaseURL(t *testing.T) {
	interaction := &ExpectedInteraction{}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with query params",
			input:    "https://api.localhost/users?page=1&limit=10",
			expected: "https://api.localhost/users",
		},
		{
			name:     "URL with fragment",
			input:    "https://api.localhost/users#section",
			expected: "https://api.localhost/users",
		},
		{
			name:     "URL with both query and fragment",
			input:    "https://api.localhost/users?page=1#section",
			expected: "https://api.localhost/users",
		},
		{
			name:     "URL without query or fragment",
			input:    "https://api.localhost/users",
			expected: "https://api.localhost/users",
		},
		{
			name:     "invalid URL returns original",
			input:    "not-a-valid-url",
			expected: "not-a-valid-url",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := interaction.extractBaseURL(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExpectedInteraction_matches(t *testing.T) {
	t.Run("matches method and URL exactly", func(t *testing.T) {
		interaction := &ExpectedInteraction{
			request: TestRequest{
				Method: "GET",
				URL:    "https://api.localhost/users",
			},
		}

		req := httptest.NewRequest("GET", "https://api.localhost/users", nil)

		assert.True(t, interaction.matches(req))
	})

	t.Run("does not match different method", func(t *testing.T) {
		interaction := &ExpectedInteraction{
			request: TestRequest{
				Method: "GET",
				URL:    "https://api.localhost/users",
			},
		}

		req := httptest.NewRequest("POST", "https://api.localhost/users", nil)

		assert.False(t, interaction.matches(req))
	})

	t.Run("does not match different URL", func(t *testing.T) {
		interaction := &ExpectedInteraction{
			request: TestRequest{
				Method: "GET",
				URL:    "https://api.localhost/users",
			},
		}

		req := httptest.NewRequest("GET", "https://api.localhost/posts", nil)

		assert.False(t, interaction.matches(req))
	})

	t.Run("matches with empty method in expectation", func(t *testing.T) {
		interaction := &ExpectedInteraction{
			request: TestRequest{
				URL: "https://api.localhost/users",
			},
		}

		req := httptest.NewRequest("POST", "https://api.localhost/users", nil)

		assert.True(t, interaction.matches(req))
	})

	t.Run("matches with empty URL in expectation", func(t *testing.T) {
		interaction := &ExpectedInteraction{
			request: TestRequest{
				Method: "GET",
			},
		}

		req := httptest.NewRequest("GET", "https://api.localhost/anything", nil)

		assert.True(t, interaction.matches(req))
	})

	t.Run("ignores query params when configured", func(t *testing.T) {
		interaction := &ExpectedInteraction{
			request: TestRequest{
				Method: "GET",
				URL:    "https://api.localhost/users",
			},
			ignoreQueryParams: true,
		}

		req := httptest.NewRequest("GET", "https://api.localhost/users?page=1&limit=10", nil)

		assert.True(t, interaction.matches(req))
	})

	t.Run("does not ignore query params when not configured", func(t *testing.T) {
		interaction := &ExpectedInteraction{
			request: TestRequest{
				Method: "GET",
				URL:    "https://api.localhost/users",
			},
			ignoreQueryParams: false,
		}

		req := httptest.NewRequest("GET", "https://api.localhost/users?page=1", nil)

		assert.False(t, interaction.matches(req))
	})
}

func TestMockInteractionTransport_RoundTrip_ExactAlgorithm(t *testing.T) {
	t.Run("matches requests in exact order", func(t *testing.T) {
		transport := NewMockInteractionTransport(t, &MockInteractionTransportOptions{
			Algorithm: Exact,
		})

		// Set up expectations in order
		transport.ExpectRequest(TestRequest{Method: "GET", URL: "https://api.localhost/first"}).
			WillReturnResponse(&TestResponse{Status: 200, Body: "first"})
		transport.ExpectRequest(TestRequest{Method: "POST", URL: "https://api.localhost/second"}).
			WillReturnResponse(&TestResponse{Status: 201, Body: "second"})

		// First request
		req1 := httptest.NewRequest("GET", "https://api.localhost/first", nil)
		resp1, err1 := transport.RoundTrip(req1)

		require.NoError(t, err1)
		assert.Equal(t, 200, resp1.StatusCode)

		// Second request
		req2 := httptest.NewRequest("POST", "https://api.localhost/second", nil)
		resp2, err2 := transport.RoundTrip(req2)

		require.NoError(t, err2)
		assert.Equal(t, 201, resp2.StatusCode)
	})

	t.Run("consumes interactions in exact order", func(t *testing.T) {
		transport := NewMockInteractionTransport(t, &MockInteractionTransportOptions{
			Algorithm: Exact,
		})

		transport.ExpectRequest(TestRequest{Method: "GET", URL: "https://api.localhost/test"}).
			WillReturnResponse(&TestResponse{Status: 200})

		// First request should work
		req1 := httptest.NewRequest("GET", "https://api.localhost/test", nil)
		_, err1 := transport.RoundTrip(req1)
		require.NoError(t, err1)

		// Verify the interaction was consumed (list should be empty)
		assert.Len(t, transport.expectedInteractions, 0)
	})
}

func TestMockInteractionTransport_RoundTrip_FirstMatchAlgorithm(t *testing.T) {
	t.Run("matches first matching interaction", func(t *testing.T) {
		transport := NewMockInteractionTransport(t, &MockInteractionTransportOptions{
			Algorithm: FirstMatch,
		})

		// Set up multiple expectations
		transport.ExpectRequest(TestRequest{Method: "GET", URL: "https://api.localhost/users"}).
			WillReturnResponse(&TestResponse{Status: 200, Body: "users"})
		transport.ExpectRequest(TestRequest{Method: "POST", URL: "https://api.localhost/posts"}).
			WillReturnResponse(&TestResponse{Status: 201, Body: "posts"})

		// Request matching second expectation
		req := httptest.NewRequest("POST", "https://api.localhost/posts", nil)
		resp, err := transport.RoundTrip(req)

		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)
	})

	t.Run("keeps interactions available for reuse", func(t *testing.T) {
		transport := NewMockInteractionTransport(t, &MockInteractionTransportOptions{
			Algorithm: FirstMatch,
		})

		transport.ExpectRequest(TestRequest{Method: "GET", URL: "https://api.localhost/test"}).
			WillReturnResponse(&TestResponse{Status: 200})

		// Multiple identical requests should all work
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "https://api.localhost/test", nil)
			_, err := transport.RoundTrip(req)
			require.NoError(t, err)
		}
	})
}

func TestMockInteractionTransport_RoundTrip_ResponseHandling(t *testing.T) {
	t.Run("returns JSON response body", func(t *testing.T) {
		transport := NewMockInteractionTransport(t, nil)

		responseBody := map[string]interface{}{
			"id":   123,
			"name": "test user",
		}

		transport.ExpectRequest(TestRequest{Method: "GET", URL: "https://api.localhost/user"}).
			WillReturnResponse(&TestResponse{
				Status: 200,
				Header: http.Header{"Content-Type": []string{"application/json"}},
				Body:   responseBody,
			})

		req := httptest.NewRequest("GET", "https://api.localhost/user", nil)
		resp, err := transport.RoundTrip(req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		// Read and verify body
		body, err := httptest.NewRecorder().Body.ReadFrom(resp.Body)
		require.NoError(t, err)
		assert.Greater(t, body, int64(0))
	})

	t.Run("returns string response body", func(t *testing.T) {
		transport := NewMockInteractionTransport(t, nil)

		transport.ExpectRequest(TestRequest{Method: "GET", URL: "https://api.localhost/text"}).
			WillReturnResponse(&TestResponse{
				Status: 200,
				Header: http.Header{"Content-Type": []string{"text/plain"}},
				Body:   "plain text response",
			})

		req := httptest.NewRequest("GET", "https://api.localhost/text", nil)
		resp, err := transport.RoundTrip(req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
		assert.NotNil(t, resp.Body)
	})

	t.Run("returns nil response when no response configured", func(t *testing.T) {
		transport := NewMockInteractionTransport(t, nil)

		transport.ExpectRequest(TestRequest{Method: "GET", URL: "https://api.localhost/nil"})

		req := httptest.NewRequest("GET", "https://api.localhost/nil", nil)
		resp, err := transport.RoundTrip(req)

		require.NoError(t, err)
		assert.Nil(t, resp)
	})
}

func TestMockInteractionTransport_ImplementsRoundTripper(t *testing.T) {
	transport := NewMockInteractionTransport(t, nil)

	// Verify it implements http.RoundTripper interface
	var _ http.RoundTripper = transport
	assert.Implements(t, (*http.RoundTripper)(nil), transport)
}
