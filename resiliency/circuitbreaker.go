package resiliency

import (
	"github.com/sony/gobreaker/v2"
	"net/http"
)

type Settings = gobreaker.Settings

type CircuitBreakerOptions struct {
	Settings Settings
}

var _ http.RoundTripper = (*CircuitBreaker)(nil)

type CircuitBreaker struct {
	http.RoundTripper
	cb *gobreaker.CircuitBreaker[*http.Response]
}

// RoundTrip implements the http.RoundTripper interface
// It adds Basic Authentication header to the request before passing it to the underlying RoundTripper
func (t *CircuitBreaker) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.cb.Execute(func() (*http.Response, error) {
		return t.RoundTripper.RoundTrip(req)
	})
}

// NewCircuitBreaker creates a new CircuitBreaker with the given credentials
func NewCircuitBreaker(rt http.RoundTripper, o CircuitBreakerOptions) *CircuitBreaker {
	cb := gobreaker.NewCircuitBreaker[*http.Request](o.Settings)
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &CircuitBreaker{
		RoundTripper: rt,
		cb:           (*gobreaker.CircuitBreaker[*http.Response])(cb),
	}
}
