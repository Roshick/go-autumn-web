package metrics

import (
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"net/http"
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
	httpServerReqDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of HTTP server requests in seconds, partitioned by status code, method, and route."),
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
			httpServerReqDuration.Record(req.Context(), duration, metric.WithAttributes(
				attribute.String("http.request.method", req.Method),
				attribute.Int("http.response.status_code", ww.Status()),
				attribute.String("http.route", routePattern),
			))
		}
		return http.HandlerFunc(fn)
	}
}
