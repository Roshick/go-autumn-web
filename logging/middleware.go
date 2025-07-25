package logging

import (
	"context"
	"github.com/Roshick/go-autumn-slog/pkg/logging"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"time"
)

// ContextLoggerMiddleware //

type ContextLoggerMiddlewareOptions struct {
}

func DefaultContextLoggerMiddlewareOptions() *ContextLoggerMiddlewareOptions {
	return &ContextLoggerMiddlewareOptions{}
}

func NewContextLoggerMiddleware(opts *ContextLoggerMiddlewareOptions) func(http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultContextLoggerMiddlewareOptions()
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			if slogging, ok := aulogging.Logger.(*logging.Logging); ok {
				ctx = logging.ContextWithLogger(ctx, slogging.Logger())
			}

			next.ServeHTTP(w, req.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// ContextCancellationLoggerMiddleware //

type ContextCancellationLoggerMiddlewareOptions struct {
	Description string
}

func DefaultContextCancellationLoggerMiddlewareOptions() *ContextCancellationLoggerMiddlewareOptions {
	return &ContextCancellationLoggerMiddlewareOptions{
		Description: "default",
	}
}

func NewContextCancellationLoggerMiddleware(opts *ContextCancellationLoggerMiddlewareOptions) func(http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultContextCancellationLoggerMiddlewareOptions()
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			if ctx.Err() != nil {
				cause := context.Cause(ctx)
				if cause != nil {
					aulogging.Logger.NoCtx().Info().WithErr(cause).Printf("context '%s' is already cancelled", opts.Description)
				}
				return
			}

			next.ServeHTTP(w, req)

			if ctx.Err() != nil {
				cause := context.Cause(ctx)
				if cause != nil {
					aulogging.Logger.NoCtx().Info().WithErr(cause).Printf("context '%s' was cancelled during request processing", opts.Description)
				}
			}
		}
		return http.HandlerFunc(fn)
	}
}

// RequestLoggerMiddleware //

type RequestLoggerMiddlewareOptions struct {
}

func DefaultRequestLoggerMiddlewareOptions() *RequestLoggerMiddlewareOptions {
	return &RequestLoggerMiddlewareOptions{}
}

func NewRequestLoggerMiddleware(opts *RequestLoggerMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultRequestLoggerMiddlewareOptions()
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			t1 := time.Now()

			next.ServeHTTP(ww, req)

			ctx := req.Context()
			if logger := logging.FromContext(ctx); logger != nil {
				duration := time.Since(t1).Microseconds()

				logger = logger.With(
					LogFieldRequestMethod, req.Method,
					LogFieldResponseStatus, ww.Status(),
					LogFieldURLPath, req.URL.Path,
					LogFieldUserAgent, req.UserAgent(),
					LogFieldLogger, "request.incoming",
					LogFieldEventDuration, duration,
				)
				subCtx := logging.ContextWithLogger(ctx, logger)

				if ww.Status() >= http.StatusInternalServerError {
					aulogging.Logger.Ctx(subCtx).Warn().Printf("downstream call %s %s -> %d FAILED (%d ms)", req.Method, req.URL.Path, ww.Status(), duration)
				} else {
					aulogging.Logger.Ctx(subCtx).Info().Printf("downstream call %s %s -> %d OK (%d ms)", req.Method, req.URL.Path, ww.Status(), duration)
				}
			}
		}
		return http.HandlerFunc(fn)
	}
}
