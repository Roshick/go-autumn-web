package logging

import (
	"context"
	"fmt"
	"github.com/Roshick/go-autumn-slog/pkg/logging"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"regexp"
	"time"
)

const (
	LogFieldRequestMethod  = "request-method"
	LogFieldRequestID      = "request-id"
	LogFieldResponseStatus = "response-status"
	LogFieldURLPath        = "url-path"
	LogFieldUserAgent      = "user-agent"
	LogFieldEventDuration  = "event-duration"
	LogFieldLogger         = "logger"
	LogFieldStackTrace     = "stack-trace"
	LogFieldTraceID        = "trace-id"
	LogFieldSpanID         = "span-id"
)

// AddLoggerToContext //

func AddLoggerToContext(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if slogging, ok := aulogging.Logger.(*logging.Logging); ok {
			ctx = logging.ContextWithLogger(ctx, slogging.Logger())
		}

		next.ServeHTTP(w, req.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// AddContextCancelLogging //

type LogContextCancellationOptions struct {
	Description string
}

func LogContextCancellation(options LogContextCancellationOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			next.ServeHTTP(w, req)

			cause := context.Cause(ctx)
			if cause != nil {
				aulogging.Logger.NoCtx().Info().WithErr(cause).Printf("context '%s' is cancelled", options.Description)
			}
		}
		return http.HandlerFunc(fn)
	}
}

// LogRequest //

type LogRequestOptions struct {
	Exclusions []string
}

func LogRequest(options LogRequestOptions) func(next http.Handler) http.Handler {
	excludeRegexes := make([]*regexp.Regexp, 0)
	for _, pattern := range options.Exclusions {
		fullMatchPattern := "^" + pattern + "$"
		re, err := regexp.Compile(fullMatchPattern)
		if err != nil {
			aulogging.Logger.NoCtx().Error().WithErr(err).Printf("failed to compile exclude logging pattern '%s', skipping pattern", fullMatchPattern)
		} else {
			excludeRegexes = append(excludeRegexes, re)
		}
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			t1 := time.Now()
			defer func() {
				ctx := req.Context()

				requestInfo := fmt.Sprintf("%s %s %d", req.Method, req.URL.Path, ww.Status())
				for _, re := range excludeRegexes {
					if re.MatchString(requestInfo) {
						return
					}
				}

				if logger := logging.FromContext(ctx); logger != nil {
					logger = logger.With(
						LogFieldRequestMethod, req.Method,
						LogFieldResponseStatus, ww.Status(),
						LogFieldURLPath, req.URL.Path,
						LogFieldUserAgent, req.UserAgent(),
						LogFieldLogger, "request.incoming",
						LogFieldEventDuration, time.Since(t1).Microseconds(),
					)
					subCtx := logging.ContextWithLogger(ctx, logger)
					if ww.Status() >= http.StatusInternalServerError {
						aulogging.Logger.Ctx(subCtx).Error().Print("request")
					} else {
						aulogging.Logger.Ctx(subCtx).Info().Print("request")
					}
				}
			}()

			next.ServeHTTP(ww, req)
		}
		return http.HandlerFunc(fn)
	}
}
