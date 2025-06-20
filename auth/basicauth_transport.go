package auth

import (
	"encoding/base64"
	"net/http"
)

type BasicAuthTransport struct {
	http.RoundTripper

	Username string
	Password string
}

// RoundTrip implements the http.RoundTripper interface
// It adds Basic Authentication header to the request before passing it to the underlying RoundTripper
func (t *BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	reqCopy := req.Clone(req.Context())

	// Add Basic Auth header
	auth := t.Username + ":" + t.Password
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	reqCopy.Header.Set("Authorization", "Basic "+encoded)

	return t.RoundTripper.RoundTrip(reqCopy)
}

// NewBasicAuthTransport creates a new BasicAuthTransport with the given credentials
func NewBasicAuthTransport(rt http.RoundTripper, username, password string) *BasicAuthTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &BasicAuthTransport{
		RoundTripper: rt,
		Username:     username,
		Password:     password,
	}
}
