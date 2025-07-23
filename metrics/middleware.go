package metrics

import (
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RequestMetricsMiddleware //

type RequestMetricsMiddlewareOptions struct{}

func DefaultRequestMetricsMiddlewareOptions() *RequestMetricsMiddlewareOptions {
	return &RequestMetricsMiddlewareOptions{}
}

func NewRequestMetricsMiddleware(opts *RequestMetricsMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultRequestMetricsMiddlewareOptions()
	}

	meter := otel.GetMeterProvider().Meter("server")
	httpServerReqSecs, err := meter.Float64Histogram(
		"http.server.requests.seconds",
		metric.WithDescription("How long it took to process requests, partitioned by status code, method, and HTTP path."),
	)
	if err != nil {
		aulogging.Logger.NoCtx().Error().WithErr(err).Print("failed to initialize request metrics middleware")
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			next.ServeHTTP(ww, req)

			routeCtx := chi.RouteContext(req.Context())
			routePattern := strings.Join(routeCtx.RoutePatterns, "")
			routePattern = strings.Replace(routePattern, "/*/", "/", -1)

			duration := float64(time.Since(start).Microseconds()) / 1000000
			httpServerReqSecs.Record(req.Context(), duration, metric.WithAttributes(
				attribute.String("method", req.Method),
				attribute.String("status", strconv.Itoa(ww.Status())),
				attribute.String("uri", routePattern),
			))
		}
		return http.HandlerFunc(fn)
	}
}
