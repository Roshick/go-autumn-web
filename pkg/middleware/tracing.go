package middleware

import (
	"github.com/Roshick/go-autumn-slog/pkg/logging"
	aucontext "github.com/Roshick/go-autumn-web/pkg/context"
	"go.opentelemetry.io/otel/trace"
	"net/http"
)

// AddTracingToContextLogger //

func AddTracingToContextLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if logger := logging.FromContext(ctx); logger != nil {
			spanCtx := trace.SpanContextFromContext(ctx)
			if spanCtx.HasTraceID() {
				logger = logger.With(LogFieldTraceID, spanCtx.TraceID().String())
			}
			if spanCtx.HasSpanID() {
				logger = logger.With(LogFieldSpanID, spanCtx.SpanID().String())
			}
			ctx = logging.ContextWithLogger(ctx, logger)
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
			ctx = aucontext.WithRequestID(ctx, requestID)

			next.ServeHTTP(w, req.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// AddRequestIDToContextLogger //

func AddRequestIDToContextLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if logger := logging.FromContext(ctx); logger != nil {
			requestID := aucontext.GetRequestID(ctx)
			if requestID != nil && *requestID != "" {
				logger = logger.With(LogFieldRequestID, *requestID)
			}
			ctx = logging.ContextWithLogger(ctx, logger)
		}

		next.ServeHTTP(w, req.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
