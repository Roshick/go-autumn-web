package middleware

import (
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

// RecordRequestMetrics //

func RecordRequestMetrics() func(next http.Handler) http.Handler {
	meter := otel.GetMeterProvider().Meter("server")
	httpServerReqSecs, _ := meter.Float64Histogram(
		"http.server.requests.seconds",
		metric.WithDescription("How long it took to process requests, partitioned by status code, method, and HTTP path."),
	)

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
