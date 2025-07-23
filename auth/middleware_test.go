package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowBasicAuthUser(t *testing.T) {
	tests := []struct {
		name           string
		options        AllowBasicAuthUserOptions
		authHeader     string
		expectedResult bool
	}{
		{
			name:           "valid credentials",
			options:        AllowBasicAuthUserOptions{Username: "testuser", Password: "testpass"},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass")),
			expectedResult: true,
		},
		{
			name:           "invalid username",
			options:        AllowBasicAuthUserOptions{Username: "testuser", Password: "testpass"},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("wronguser:testpass")),
			expectedResult: false,
		},
		{
			name:           "invalid password",
			options:        AllowBasicAuthUserOptions{Username: "testuser", Password: "testpass"},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:wrongpass")),
			expectedResult: false,
		},
		{
			name:           "no auth header",
			options:        AllowBasicAuthUserOptions{Username: "testuser", Password: "testpass"},
			authHeader:     "",
			expectedResult: false,
		},
		{
			name:           "empty credentials in options",
			options:        AllowBasicAuthUserOptions{Username: "", Password: ""},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte(":")),
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authFn := AllowBasicAuthUser(tt.options)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			result := authFn(req)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestRejectAll(t *testing.T) {
	authFn := RejectAll()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	result := authFn(req)
	assert.False(t, result)
}

func TestDefaultAuthorizationMiddlewareOptions(t *testing.T) {
	opts := DefaultAuthorizationMiddlewareOptions()

	require.NotNil(t, opts)
	assert.Len(t, opts.AuthorizationFns, 1)
	assert.NotNil(t, opts.ErrorResponse)
}

func TestNewAuthorizationMiddleware(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		middleware := NewAuthorizationMiddleware(nil)
		assert.NotNil(t, middleware)
	})

	t.Run("authorization success", func(t *testing.T) {
		opts := &AuthorizationMiddlewareOptions{
			AuthorizationFns: []AuthorizationFn{
				func(*http.Request) bool { return true },
			},
		}

		middleware := NewAuthorizationMiddleware(opts)

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

	t.Run("authorization failure", func(t *testing.T) {
		opts := &AuthorizationMiddlewareOptions{
			AuthorizationFns: []AuthorizationFn{
				func(*http.Request) bool { return false },
			},
			ErrorResponse: DefaultAuthorizationMiddlewareOptions().ErrorResponse, // Add missing ErrorResponse
		}

		middleware := NewAuthorizationMiddleware(opts)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("multiple authorization functions", func(t *testing.T) {
		opts := &AuthorizationMiddlewareOptions{
			AuthorizationFns: []AuthorizationFn{
				func(*http.Request) bool { return false }, // First one fails
				func(*http.Request) bool { return true },  // Second one succeeds
			},
		}

		middleware := NewAuthorizationMiddleware(opts)

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
