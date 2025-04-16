package tracing

import (
	"net/http"
)

// SetRequestIDHeader

type SetRequestIDHeader struct {
	http.RoundTripper
}

func NewSetRequestIDHeader(rt http.RoundTripper) *SetRequestIDHeader {
	return &SetRequestIDHeader{
		RoundTripper: rt,
	}
}

func (t *SetRequestIDHeader) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	requestID := GetRequestID(ctx)
	if requestID != nil && *requestID != "" {
		req.Header.Set("test", *requestID)
	}

	return t.RoundTripper.RoundTrip(req)
}
