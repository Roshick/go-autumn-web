package logging

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"net/http"
	"time"
)

// RequestLoggerTransport //

type RequestLoggerTransportOptions struct {
}

var _ http.RoundTripper = (*RequestLoggerTransport)(nil)

type RequestLoggerTransport struct {
	base http.RoundTripper
	opts *RequestLoggerTransportOptions
}

func DefaultRequestLoggerTransportOptions() *RequestLoggerTransportOptions {
	return &RequestLoggerTransportOptions{}
}

func NewRequestLoggerTransport(rt http.RoundTripper, opts *RequestLoggerTransportOptions) *RequestLoggerTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}
	if opts == nil {
		opts = DefaultRequestLoggerTransportOptions()
	}

	return &RequestLoggerTransport{
		base: rt,
		opts: opts,
	}
}

func (t *RequestLoggerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.logRequest(req.Context(), req.Method, req.URL.String())

	startTime := time.Now()
	res, err := t.base.RoundTrip(req)
	statusCode := 0
	if res != nil {
		statusCode = res.StatusCode
	}

	t.logResponse(req.Context(), req.Method, req.URL.String(), statusCode, err, startTime)
	return res, err
}

func (t *RequestLoggerTransport) logRequest(ctx context.Context, method string, requestUrl string) {
	aulogging.Logger.Ctx(ctx).Info().Printf("upstream call %s %s", method, requestUrl)
}

func (t *RequestLoggerTransport) logResponse(ctx context.Context, method string, requestUrl string, responseStatusCode int, err error, startTime time.Time) {
	reqDuration := time.Now().Sub(startTime).Milliseconds()
	if err != nil {
		aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("request %s %s -> %d FAILED (%d ms)", method, requestUrl, responseStatusCode, reqDuration)
		return
	}
	aulogging.Logger.Ctx(ctx).Info().Printf("request %s %s -> %d OK (%d ms)", method, requestUrl, responseStatusCode, reqDuration)
}
