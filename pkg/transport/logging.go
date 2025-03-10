package transport

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"net/http"
	"time"
)

// RequestLogger //

type RequestLogger struct {
	http.RoundTripper
}

func NewRequestLogger(rt http.RoundTripper) *RequestLogger {
	return &RequestLogger{
		RoundTripper: rt,
	}
}

func (t *RequestLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	t.logRequest(req.Context(), req.Method, req.URL.String())

	startTime := time.Now()
	res, err := t.RoundTripper.RoundTrip(req)
	statusCode := 0
	if res != nil {
		statusCode = res.StatusCode
	}

	t.logResponse(req.Context(), req.Method, req.URL.String(), statusCode, err, startTime)
	return res, err
}

func (t *RequestLogger) logRequest(ctx context.Context, method string, requestUrl string) {
	aulogging.Logger.Ctx(ctx).Info().Printf("upstream call %s %s", method, requestUrl)
}

func (t *RequestLogger) logResponse(ctx context.Context, method string, requestUrl string, responseStatusCode int, err error, startTime time.Time) {
	reqDuration := time.Now().Sub(startTime).Milliseconds()
	if err != nil {
		aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("upstream call %s %s -> %d FAILED (%d ms)", method, requestUrl, responseStatusCode, reqDuration)
		return
	}
	aulogging.Logger.Ctx(ctx).Info().Printf("upstream call %s %s -> %d OK (%d ms)", method, requestUrl, responseStatusCode, reqDuration)
}
