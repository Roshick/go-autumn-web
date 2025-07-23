package auth

import (
	"encoding/base64"
	"net/http"
)

type BasicAuthTransportOptions struct {
}

var _ http.RoundTripper = (*BasicAuthTransport)(nil)

type BasicAuthTransport struct {
	base http.RoundTripper
	opts *BasicAuthTransportOptions

	username string
	password string
}

func (t *BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqCopy := req.Clone(req.Context())

	auth := t.username + ":" + t.password
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	reqCopy.Header.Set("Authorization", "Basic "+encoded)

	return t.base.RoundTrip(reqCopy)
}

func DefaultBasicAuthTransportOptions() *BasicAuthTransportOptions {
	return &BasicAuthTransportOptions{}
}

func NewBasicAuthTransport(rt http.RoundTripper, username, password string, opts *BasicAuthTransportOptions) *BasicAuthTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}
	if opts == nil {
		opts = &BasicAuthTransportOptions{}
	}

	return &BasicAuthTransport{
		base: rt,
		opts: opts,

		username: username,
		password: password,
	}
}
