package resiliency

import (
	"github.com/sony/gobreaker/v2"
	"net/http"
	"time"
)

type CircuitBreakerTransportOptions struct {
	gobreaker.Settings
}

var _ http.RoundTripper = (*CircuitBreakerTransport)(nil)

type CircuitBreakerTransport struct {
	base http.RoundTripper
	cb   *gobreaker.CircuitBreaker[*http.Response]
}

func (t *CircuitBreakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.cb.Execute(func() (*http.Response, error) {
		return t.base.RoundTrip(req)
	})
}

func DefaultCircuitBreakerTransportOptions() *CircuitBreakerTransportOptions {
	return &CircuitBreakerTransportOptions{
		Settings: gobreaker.Settings{
			Name:        "default",
			MaxRequests: 5,
			Interval:    60 * time.Second,
			Timeout:     60 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				return counts.Requests >= 5 && failureRatio >= 0.6
			},
		},
	}
}

func NewCircuitBreakerTransport(rt http.RoundTripper, opts *CircuitBreakerTransportOptions) *CircuitBreakerTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}
	if opts == nil {
		opts = DefaultCircuitBreakerTransportOptions()
	}

	cb := gobreaker.NewCircuitBreaker[*http.Response](opts.Settings)
	return &CircuitBreakerTransport{
		base: rt,
		cb:   cb,
	}
}
