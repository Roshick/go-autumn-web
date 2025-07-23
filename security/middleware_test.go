package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCORSMiddlewareOptions(t *testing.T) {
	opts := DefaultCORSMiddlewareOptions()

	require.NotNil(t, opts)
	assert.Equal(t, "*", opts.AllowOrigin)
	assert.False(t, opts.AllowCredentials) // FIXED: Should be false by default for security
	assert.Equal(t, 3600, opts.MaxAge)
	assert.NotNil(t, opts.AdditionalAllowHeaders)
	assert.NotNil(t, opts.AdditionalExposeHeaders)
}

func TestNewCORSMiddleware(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		middleware := NewCORSMiddleware(nil)
		assert.NotNil(t, middleware)
	})

	t.Run("regular request", func(t *testing.T) {
		opts := DefaultCORSMiddlewareOptions()
		middleware := NewCORSMiddleware(opts)

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

		// Check CORS headers
		assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Headers"), "Accept")
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		// FIXED: Credentials should not be set for wildcard origin
		assert.Empty(t, rr.Header().Get("Access-Control-Allow-Credentials"))
		assert.NotEmpty(t, rr.Header().Get("Access-Control-Expose-Headers"))
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		opts := DefaultCORSMiddlewareOptions()
		middleware := NewCORSMiddleware(opts)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		// Handler should not be called for OPTIONS requests
		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusNoContent, rr.Code) // FIXED: Should be 204, not 200

		// Check CORS headers are still present
		assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "3600", rr.Header().Get("Access-Control-Max-Age")) // ADDED: Cache control
	})

	t.Run("custom origin", func(t *testing.T) {
		opts := &CORSMiddlewareOptions{
			AllowOrigin:             "https://localhost",
			AllowCredentials:        true, // This should work with specific origin
			MaxAge:                  7200,
			AdditionalAllowHeaders:  []string{"X-Custom-Header"},
			AdditionalExposeHeaders: []string{"X-Custom-Response"},
		}
		middleware := NewCORSMiddleware(opts)

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

		// Check custom headers
		assert.Equal(t, "https://localhost", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials")) // Should work with specific origin
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Headers"), "X-Custom-Header")
		assert.Contains(t, rr.Header().Get("Access-Control-Expose-Headers"), "X-Custom-Response")
	})

	// NEW TEST: Verify security fix for wildcard + credentials
	t.Run("security fix - wildcard origin with credentials should disable credentials", func(t *testing.T) {
		opts := &CORSMiddlewareOptions{
			AllowOrigin:      "*",
			AllowCredentials: true, // This should be automatically disabled
		}
		middleware := NewCORSMiddleware(opts)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rr, req)

		// Credentials should NOT be set due to security validation
		assert.Empty(t, rr.Header().Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	})
}
