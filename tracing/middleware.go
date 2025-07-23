package tracing

import (
	"crypto/rand"
	"fmt"
	slogging "github.com/Roshick/go-autumn-slog/pkg/logging"
	"github.com/Roshick/go-autumn-web/header"
	"github.com/Roshick/go-autumn-web/logging"
	"go.opentelemetry.io/otel/trace"
	mathrand "math/rand/v2"
	"net/http"
	"time"
)

// TracingLoggerMiddleware //

type TracingLoggerMiddlewareOptions struct {
	LogFieldTraceID string
	LogFieldSpanID  string
}

func DefaultTracingLoggerMiddlewareOptions() *TracingLoggerMiddlewareOptions {
	return &TracingLoggerMiddlewareOptions{
		LogFieldTraceID: logging.LogFieldTraceID,
		LogFieldSpanID:  logging.LogFieldSpanID,
	}
}

func NewTracingLoggerMiddleware(opts *TracingLoggerMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultTracingLoggerMiddlewareOptions()
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			if logger := slogging.FromContext(ctx); logger != nil {
				spanCtx := trace.SpanContextFromContext(ctx)
				if spanCtx.HasTraceID() {
					logger = logger.With(opts.LogFieldTraceID, spanCtx.TraceID().String())
				}
				if spanCtx.HasSpanID() {
					logger = logger.With(opts.LogFieldSpanID, spanCtx.SpanID().String())
				}
				ctx = slogging.ContextWithLogger(ctx, logger)
			}

			next.ServeHTTP(w, req.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// RequestIDHeaderMiddleware //

type RequestIDHeaderMiddlewareOptions struct {
	HeaderName  string
	GeneratorFn func() string
}

func DefaultRequestIDHeaderMiddlewareOptions() *RequestIDHeaderMiddlewareOptions {
	return &RequestIDHeaderMiddlewareOptions{
		HeaderName:  header.XRequestID,
		GeneratorFn: DefaultRequestIDGenerator,
	}
}

func NewRequestIDHeaderMiddleware(opts *RequestIDHeaderMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultRequestIDHeaderMiddlewareOptions()
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			requestID := req.Header.Get(opts.HeaderName)
			if requestID == "" {
				requestID = opts.GeneratorFn()
			}
			w.Header().Set(opts.HeaderName, requestID)
			ctx = ContextWithRequestID(ctx, requestID)

			next.ServeHTTP(w, req.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// RequestIDLoggerMiddleware //

type RequestIDLoggerMiddlewareOptions struct {
	LogFieldName string
}

func DefaultRequestIDLoggerMiddlewareOptions() RequestIDLoggerMiddlewareOptions {
	return RequestIDLoggerMiddlewareOptions{
		LogFieldName: logging.LogFieldRequestID,
	}
}

func NewRequestIDLoggerMiddleware(opts *RequestIDLoggerMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = &RequestIDLoggerMiddlewareOptions{
			LogFieldName: logging.LogFieldRequestID,
		}
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			if logger := slogging.FromContext(ctx); logger != nil {
				requestID := RequestIDFromContext(ctx)
				if requestID != nil && *requestID != "" {
					logger = logger.With(opts.LogFieldName, *requestID)
				}
				ctx = slogging.ContextWithLogger(ctx, logger)
			}

			next.ServeHTTP(w, req.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// DefaultRequestIDGenerator generates a UUID v4 style request ID
func DefaultRequestIDGenerator() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to a simple timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), mathrand.IntN(10000))
	}

	// Set version (4) and variant bits according to RFC 4122
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
