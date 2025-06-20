package auth

import (
	"encoding/base64"
	"net/http"
)

type BasicAuthOptions struct {
	Username string
	Password string
}

var _ http.RoundTripper = (*BasicAuth)(nil)

type BasicAuth struct {
	http.RoundTripper

	Username string
	Password string
}

// RoundTrip implements the http.RoundTripper interface
// It adds Basic Authentication header to the request before passing it to the underlying RoundTripper
func (t *BasicAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	reqCopy := req.Clone(req.Context())

	// Add Basic Auth header
	auth := t.Username + ":" + t.Password
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	reqCopy.Header.Set("Authorization", "Basic "+encoded)

	return t.RoundTripper.RoundTrip(reqCopy)
}

// NewBasicAuth creates a new BasicAuth with the given credentials
func NewBasicAuth(rt http.RoundTripper, o BasicAuthOptions) *BasicAuth {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return &BasicAuth{
		RoundTripper: rt,
		Username:     o.Username,
		Password:     o.Password,
	}
}
