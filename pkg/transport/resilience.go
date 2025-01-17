package transport

import (
	"context"
	"net/http"
	"time"
)

// TimeoutContext //

type TimeoutContextTransport struct {
	http.RoundTripper

	timeout time.Duration
}

func NewTimeoutContextTransport(rt http.RoundTripper, timeout time.Duration) *TimeoutContextTransport {
	return &TimeoutContextTransport{
		RoundTripper: rt,
		timeout:      timeout,
	}
}

func (t *TimeoutContextTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, t.timeout)
	defer cancel()

	return t.RoundTripper.RoundTrip(req.WithContext(ctx))
}
