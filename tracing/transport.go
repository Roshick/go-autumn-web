package tracing

import (
	"github.com/Roshick/go-autumn-web/header"
	"net/http"
)

// RequestIDHeaderTransport

type RequestIDHeaderTransportOptions struct {
	HeaderName string
}

type RequestIDHeaderTransport struct {
	base http.RoundTripper
	opts *RequestIDHeaderTransportOptions
}

var _ http.RoundTripper = (*RequestIDHeaderTransport)(nil)

func DefaultRequestIDHeaderTransportOptions() *RequestIDHeaderTransportOptions {
	return &RequestIDHeaderTransportOptions{
		HeaderName: header.XRequestID,
	}
}

func NewRequestIDHeaderTransport(rt http.RoundTripper, opts *RequestIDHeaderTransportOptions) *RequestIDHeaderTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}
	if opts == nil {
		opts = DefaultRequestIDHeaderTransportOptions()
	}

	return &RequestIDHeaderTransport{
		base: rt,
		opts: opts,
	}
}

func (t *RequestIDHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	requestID := RequestIDFromContext(ctx)
	if requestID != nil && *requestID != "" {
		// Clone the request to avoid modifying the original
		reqCopy := req.Clone(req.Context())
		reqCopy.Header.Set(t.opts.HeaderName, *requestID)
		return t.base.RoundTrip(reqCopy)
	}

	return t.base.RoundTrip(req)
}
