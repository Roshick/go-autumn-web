package logging

import (
	"context"
	"net/http"
	"time"

	aulogging "github.com/StephanHCB/go-autumn-logging"
)

// RequestLoggerTransport //

type RequestLoggerTransportOptions struct {
	// WarningStatusCodeThreshold defines the status code boundary above which
	// responses are logged as warnings instead of info. Defaults to 500 (5xx errors).
	WarningStatusCodeThreshold int
}

var _ http.RoundTripper = (*RequestLoggerTransport)(nil)

type RequestLoggerTransport struct {
	base http.RoundTripper
	opts *RequestLoggerTransportOptions
}

func DefaultRequestLoggerTransportOptions() *RequestLoggerTransportOptions {
	return &RequestLoggerTransportOptions{
		WarningStatusCodeThreshold: 500,
	}
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
	startTime := time.Now()
	res, err := t.base.RoundTrip(req)
	statusCode := 0
	if res != nil {
		statusCode = res.StatusCode
	}

	t.logResponse(req.Context(), req.Method, req.URL.String(), statusCode, err, startTime)
	return res, err
}

func (t *RequestLoggerTransport) logResponse(ctx context.Context, method string, requestUrl string, responseStatusCode int, err error, startTime time.Time) {
	reqDuration := time.Now().Sub(startTime).Milliseconds()
	if err != nil || responseStatusCode >= t.opts.WarningStatusCodeThreshold {
		aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("request %s %s -> %d (%d ms)", method, requestUrl, responseStatusCode, reqDuration)
		return
	}
	aulogging.Logger.Ctx(ctx).Info().Printf("request %s %s -> %d (%d ms)", method, requestUrl, responseStatusCode, reqDuration)
}
