package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RequestMetricsTransport //

type RequestMetricsTransportOptions struct{}

func DefaultRequestMetricsTransportOptions() *RequestMetricsTransportOptions {
	return &RequestMetricsTransportOptions{}
}

type RequestMetricsTransport struct {
	base       http.RoundTripper
	clientName string
	opts       *RequestMetricsTransportOptions

	httpClientCounts    metric.Int64Counter
	httpClientErrCounts metric.Int64Counter
	httpClientReqBytes  metric.Float64Histogram
	httpClientResBytes  metric.Float64Histogram
}

func NewRequestMetricsTransport(base http.RoundTripper, clientName string, opts *RequestMetricsTransportOptions) *RequestMetricsTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if opts == nil {
		opts = DefaultRequestMetricsTransportOptions()
	}

	t := &RequestMetricsTransport{
		base:       base,
		clientName: clientName,
		opts:       opts,
	}
	t.init()
	return t
}

func (t *RequestMetricsTransport) init() {
	meterName := "http.client"
	if t.clientName != "" {
		meterName = fmt.Sprintf("http.client.%s", strings.ReplaceAll(t.clientName, "-", "_"))
	}
	meter := otel.GetMeterProvider().Meter(meterName)

	t.httpClientCounts, _ = meter.Int64Counter(
		"http.client.request.total",
		metric.WithDescription("Total number of HTTP client requests by method and status code"),
	)
	t.httpClientErrCounts, _ = meter.Int64Counter(
		"http.client.request.errors.total",
		metric.WithDescription("Total number of HTTP client request errors by method and status code"),
	)
	t.httpClientReqBytes, _ = meter.Float64Histogram(
		"http.client.request.size",
		metric.WithDescription("Size of HTTP client request bodies in bytes"),
	)
	t.httpClientResBytes, _ = meter.Float64Histogram(
		"http.client.response.size",
		metric.WithDescription("Size of HTTP client response bodies in bytes"),
	)
}

func (t *RequestMetricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.recordRequest(req.Context(), req)

	resp, err := t.base.RoundTrip(req)
	t.recordResponse(req.Context(), req, resp, err)

	return resp, err
}

func (t *RequestMetricsTransport) recordRequest(ctx context.Context, req *http.Request) {
	attributes := []attribute.KeyValue{
		attribute.String("http.request.method", req.Method),
	}
	if t.clientName != "" {
		attributes = append(attributes, attribute.String("client.name", t.clientName))
	}

	size := int(req.ContentLength)
	if size > 0 {
		t.httpClientReqBytes.Record(ctx, float64(size), metric.WithAttributes(attributes...))
	}
}

func (t *RequestMetricsTransport) recordResponse(ctx context.Context, req *http.Request, resp *http.Response, err error) {
	var statusCode, size int
	if resp != nil {
		statusCode = resp.StatusCode
		size = int(resp.ContentLength)
	}

	attributes := []attribute.KeyValue{
		attribute.String("http.request.method", req.Method),
	}
	if statusCode > 0 {
		attributes = append(attributes, attribute.Int("http.response.status_code", statusCode))
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
