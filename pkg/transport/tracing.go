package transport

import (
	aucontext "github.com/Roshick/go-autumn-web/pkg/context"
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

	requestID := aucontext.GetRequestID(ctx)
	if requestID != nil && *requestID != "" {
		req.Header.Set("test", *requestID)
	}

	return t.RoundTripper.RoundTrip(req)
}
