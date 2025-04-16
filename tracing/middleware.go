package tracing

import (
	slogging "github.com/Roshick/go-autumn-slog/pkg/logging"
	"github.com/Roshick/go-autumn-web/logging"
	"go.opentelemetry.io/otel/trace"
	"net/http"
)

// AddTracingToContextLogger //

func AddTracingToContextLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if logger := slogging.FromContext(ctx); logger != nil {
			spanCtx := trace.SpanContextFromContext(ctx)
			if spanCtx.HasTraceID() {
				logger = logger.With(logging.LogFieldTraceID, spanCtx.TraceID().String())
			}
			if spanCtx.HasSpanID() {
				logger = logger.With(logging.LogFieldSpanID, spanCtx.SpanID().String())
			}
			ctx = slogging.ContextWithLogger(ctx, logger)
		}

		next.ServeHTTP(w, req.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// AddRequestID //

type AddRequestIDOptions struct {
	Header      string
	GeneratorFn func() string
}

func AddRequestID(options AddRequestIDOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			requestID := req.Header.Get(options.Header)
			if requestID == "" {
				requestID = options.GeneratorFn()
			}
			w.Header().Set(options.Header, requestID)
			ctx = WithRequestID(ctx, requestID)

			next.ServeHTTP(w, req.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// AddRequestIDToContextLogger //

func AddRequestIDToContextLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if logger := slogging.FromContext(ctx); logger != nil {
			requestID := GetRequestID(ctx)
			if requestID != nil && *requestID != "" {
				logger = logger.With(logging.LogFieldRequestID, *requestID)
			}
			ctx = slogging.ContextWithLogger(ctx, logger)
		}

		next.ServeHTTP(w, req.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
