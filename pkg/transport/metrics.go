package transport

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"net/http"
	"strings"
)

// MetricsRecorder //

type MetricsRecorder struct {
	http.RoundTripper

	clientName string

	httpClientCounts    metric.Int64Counter
	httpClientErrCounts metric.Int64Counter
	httpClientReqBytes  metric.Float64Histogram
	httpClientResBytes  metric.Float64Histogram
}

func NewMetricsRecorder(rt http.RoundTripper, clientName string) *MetricsRecorder {
	transport := &MetricsRecorder{
		RoundTripper: rt,
		clientName:   clientName,
	}
	transport.init()
	return transport
}

func (t *MetricsRecorder) init() {
	meterName := fmt.Sprintf("client.%s", strings.ReplaceAll(t.clientName, "-", "_"))
	meter := otel.GetMeterProvider().Meter(meterName)

	t.httpClientCounts, _ = meter.Int64Counter(
		"http.client.requests.count",
		metric.WithDescription("Number of upstream http requests by target hostname, method, and response status."),
	)
	t.httpClientErrCounts, _ = meter.Int64Counter(
		"http.client.requests.errors.count",
		metric.WithDescription("Number of upstream http requests that raised a technical error by target hostname, method, and response status."),
	)
	t.httpClientReqBytes, _ = meter.Float64Histogram(
		"http.client.requests.request.bytes",
		metric.WithDescription("Size of the request by target hostname and method."),
	)
	t.httpClientResBytes, _ = meter.Float64Histogram(
		"http.client.requests.response.bytes",
		metric.WithDescription("Size of the response by target hostname, method, outcome, and response status."),
	)
}

func (t *MetricsRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	t.recordRequest(req.Context(), req.Method, int(req.ContentLength))

	res, err := t.RoundTripper.RoundTrip(req)
	statusCode := 0
	contentLength := 0
	if res != nil {
		statusCode = res.StatusCode
		contentLength = int(res.ContentLength)
	}

	t.recordResponse(req.Context(), req.Method, statusCode, contentLength, err)
	return res, err
}

func (t *MetricsRecorder) recordRequest(ctx context.Context, method string, size int) {
	if size > 0 {
		t.httpClientReqBytes.Record(ctx, float64(size),
			metric.WithAttributes(
				attribute.String("clientName", t.clientName),
				attribute.String("method", method),
			),
		)
	}
}

func (t *MetricsRecorder) recordResponse(ctx context.Context, method string, status int, size int, err error) {
	statusStr := fmt.Sprintf("%d", status)

	t.httpClientCounts.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("clientName", t.clientName),
			attribute.String("method", method),
			attribute.String("status", statusStr),
		),
	)
	if err != nil {
		t.httpClientErrCounts.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("clientName", t.clientName),
				attribute.String("method", method),
				attribute.String("status", statusStr),
			),
		)
	}
	if size > 0 {
		t.httpClientResBytes.Record(ctx, float64(size),
			metric.WithAttributes(
				attribute.String("clientName", t.clientName),
				attribute.String("method", method),
				attribute.String("status", statusStr),
			),
		)
	}
}
