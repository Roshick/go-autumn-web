package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRequestMetricsMiddlewareOptions(t *testing.T) {
	opts := DefaultRequestMetricsMiddlewareOptions()
	require.NotNil(t, opts)
}

func TestNewRequestMetricsMiddleware(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		middleware := NewRequestMetricsMiddleware(nil)
		assert.NotNil(t, middleware)
	})

	t.Run("middleware execution", func(t *testing.T) {
		opts := DefaultRequestMetricsMiddlewareOptions()
		middleware := NewRequestMetricsMiddleware(opts)

		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		// Create a router with chi to simulate route context
		r := chi.NewRouter()
		r.Use(middleware)
		r.Get("/test", testHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("middleware with different status codes", func(t *testing.T) {
		opts := DefaultRequestMetricsMiddlewareOptions()
		middleware := NewRequestMetricsMiddleware(opts)

		testCases := []int{200, 404, 500}

		for _, statusCode := range testCases {
			t.Run(http.StatusText(statusCode), func(t *testing.T) {
				handlerCalled := false
				testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handlerCalled = true
					w.WriteHeader(statusCode)
				})

				r := chi.NewRouter()
				r.Use(middleware)
				r.Get("/test", testHandler)

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				rr := httptest.NewRecorder()

				r.ServeHTTP(rr, req)

				assert.True(t, handlerCalled)
				assert.Equal(t, statusCode, rr.Code)
			})
		}
	})

	t.Run("middleware with different HTTP methods", func(t *testing.T) {
		opts := DefaultRequestMetricsMiddlewareOptions()
		middleware := NewRequestMetricsMiddleware(opts)

		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				handlerCalled := false
				testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handlerCalled = true
					w.WriteHeader(http.StatusOK)
				})

				r := chi.NewRouter()
				r.Use(middleware)
				r.MethodFunc(method, "/test", testHandler)

				req := httptest.NewRequest(method, "/test", nil)
				rr := httptest.NewRecorder()

				r.ServeHTTP(rr, req)

				assert.True(t, handlerCalled)
				assert.Equal(t, http.StatusOK, rr.Code)
			})
		}
	})
}
