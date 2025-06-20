package resiliency

import (
	"github.com/sony/gobreaker/v2"
	"net/http"
)

type Settings = gobreaker.Settings

type CircuitBreakerTransport struct {
	http.RoundTripper
	cb *gobreaker.CircuitBreaker[*http.Response]
}

// RoundTrip implements the http.RoundTripper interface
// It adds Basic Authentication header to the request before passing it to the underlying RoundTripper
func (t *CircuitBreakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.cb.Execute(func() (*http.Response, error) {
		return t.RoundTripper.RoundTrip(req)
	})
}

// NewCircuitBreakerTransport creates a new BasicAuthTransport with the given credentials
func NewCircuitBreakerTransport(rt http.RoundTripper, s Settings) *CircuitBreakerTransport {
	cb := gobreaker.NewCircuitBreaker[*http.Request](s)
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &CircuitBreakerTransport{
		RoundTripper: rt,
		cb:           (*gobreaker.CircuitBreaker[*http.Response])(cb),
	}
}
