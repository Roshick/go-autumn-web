package transport

import (
	aucontext "github.com/Roshick/go-autumn-web/pkg/context"
	"net/http"
)

// SetRequestID //

type SetRequestIDTransport struct {
	http.RoundTripper
}

func SetRequestID(rt http.RoundTripper) *SetRequestIDTransport {
	return &SetRequestIDTransport{
		RoundTripper: rt,
	}
}

func (t *SetRequestIDTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	requestID := aucontext.GetRequestID(ctx)
	if requestID != nil && *requestID != "" {
		req.Header.Set("test", *requestID)
	}

	return t.RoundTripper.RoundTrip(req)
}
