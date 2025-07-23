package metrics

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"net/http"
	"strings"
)

// RequestMetricsTransport //

type RequestMetricsTransportOptions struct{}

var _ http.RoundTripper = (*RequestMetricsTransport)(nil)

type RequestMetricsTransport struct {
	base http.RoundTripper

	opts *RequestMetricsTransportOptions

	clientName string

	httpClientCounts    metric.Int64Counter
	httpClientErrCounts metric.Int64Counter
	httpClientReqBytes  metric.Float64Histogram
	httpClientResBytes  metric.Float64Histogram
}

func DefaultRequestMetricsTransportOptions() *RequestMetricsTransportOptions {
	return &RequestMetricsTransportOptions{}
}

func NewRequestMetricsTransport(rt http.RoundTripper, clientName string, opts *RequestMetricsTransportOptions) *RequestMetricsTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}
	if opts == nil {
		opts = DefaultRequestMetricsTransportOptions()
	}

	transport := &RequestMetricsTransport{
		base:       rt,
		opts:       opts,
		clientName: clientName,
	}
	transport.init()
	return transport
}

func (t *RequestMetricsTransport) init() {
	meterName := "client.default"
	if t.clientName != "" {
		meterName = fmt.Sprintf("client.%s", strings.ReplaceAll(t.clientName, "-", "_"))
	}
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

func (t *RequestMetricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.recordRequest(req.Context(), req.Method, int(req.ContentLength))

	res, err := t.base.RoundTrip(req)
	statusCode := 0
	contentLength := 0
	if res != nil {
		statusCode = res.StatusCode
		contentLength = int(res.ContentLength)
	}

	t.recordResponse(req.Context(), req.Method, statusCode, contentLength, err)
	return res, err
}

func (t *RequestMetricsTransport) recordRequest(ctx context.Context, method string, size int) {
	attributes := []attribute.KeyValue{
		attribute.String("http.method", method),
	}
	if t.clientName != "" {
		attributes = append(attributes, attribute.String("client.name", t.clientName))
	}

	if size > 0 {
		t.httpClientReqBytes.Record(ctx, float64(size), metric.WithAttributes(attributes...))
	}
}

func (t *RequestMetricsTransport) recordResponse(ctx context.Context, method string, status int, size int, err error) {
	attributes := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.Int("response.status", status),
	}
	if t.clientName != "" {
		attributes = append(attributes, attribute.String("client.name", t.clientName))
	}

	t.httpClientCounts.Add(ctx, 1, metric.WithAttributes(attributes...))
	if err != nil {
		t.httpClientErrCounts.Add(ctx, 1, metric.WithAttributes(attributes...))
	}
	if size > 0 {
		t.httpClientResBytes.Record(ctx, float64(size), metric.WithAttributes(attributes...))
	}
}
