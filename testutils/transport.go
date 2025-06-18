package testutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/stretchr/testify/require"

	"net/url"
	"strings"
	"sync"
	"testing"
)

// MatchingAlgorithm represents the strategy for selecting expected interactions
type MatchingAlgorithm int

const (
	// Exact uses interactions in the exact order they were added, consuming them as they are used
	Exact MatchingAlgorithm = iota
	// FirstMatch returns the first interaction that matches the request, keeping the interaction in the pool
	FirstMatch
)

type ExpectedInteraction struct {
	request           TestRequest
	response          *TestResponse
	ignoreQueryParams bool
}

func (r *ExpectedInteraction) WillReturnResponse(response *TestResponse) {
	r.response = response
}

// IgnoreQueryParams sets whether to ignore query parameters when matching URLs
func (r *ExpectedInteraction) IgnoreQueryParams(ignore bool) *ExpectedInteraction {
	r.ignoreQueryParams = ignore
	return r
}

// extractBaseURL removes query parameters from a URL string
func (r *ExpectedInteraction) extractBaseURL(urlStr string) string {
	if parsedURL, err := url.Parse(urlStr); err == nil {
		parsedURL.RawQuery = ""
		parsedURL.Fragment = ""
		return parsedURL.String()
	}
	return urlStr
}

// matches checks if this interaction matches the given request
func (r *ExpectedInteraction) matches(req *http.Request) bool {
	if r.request.Method != "" && r.request.Method != req.Method {
		return false
	}

	if r.request.URL != "" {
		expectedURL := r.request.URL
		actualURL := req.URL.String()

		if r.ignoreQueryParams {
			expectedURL = r.extractBaseURL(expectedURL)
			actualURL = r.extractBaseURL(actualURL)
		}

		if expectedURL != actualURL {
			return false
		}
	}

	return true
}

type MockInteractionRoundTripper struct {
	t                    *testing.T
	expectedInteractions []*ExpectedInteraction
	algorithm            MatchingAlgorithm
	m                    sync.RWMutex
}

type MockInteractionRoundTripperOptions struct {
	Algorithm MatchingAlgorithm // defaults to Exact
}

func NewMockInteractionRoundTripper(t *testing.T, options MockInteractionRoundTripperOptions) *MockInteractionRoundTripper {
	return &MockInteractionRoundTripper{
		t:                    t,
		expectedInteractions: make([]*ExpectedInteraction, 0),
		algorithm:            options.Algorithm,
		m:                    sync.RWMutex{},
	}
}

func (c *MockInteractionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var next *ExpectedInteraction

	switch c.algorithm {
	case Exact:
		c.m.Lock()
		defer c.m.Unlock()
		next = c.selectExact()
	case FirstMatch:
		c.m.RLock()
		defer c.m.RUnlock()
		next = c.selectFirstMatch(req)
	default:
		c.t.Fatalf("unknown matching algorithm: %v", c.algorithm)
	}

	require.NotNil(c.t, next, fmt.Sprintf("no matching expected interaction found for %s to %s", req.Method, req.URL.String()))

	// Validate the request matches the expectation
	if next.request.Method != "" {
		require.Equal(c.t, next.request.Method, req.Method)
	}
	if next.request.URL != "" {
		expectedURL := next.request.URL
		actualURL := req.URL.String()

		if next.ignoreQueryParams {
			expectedURL = next.extractBaseURL(expectedURL)
			actualURL = next.extractBaseURL(actualURL)
		}

		require.Equal(c.t, expectedURL, actualURL)
	}

	if next.response != nil {
		mockRes := *next.response
		var body io.ReadCloser
		if mockRes.Body != nil {
			var bodyBytes []byte
			ct := mockRes.Header.Get("Content-Type")
			switch {
			case strings.HasPrefix(ct, "application/json"):
				var innerErr error
				if bodyBytes, innerErr = json.Marshal(mockRes.Body); innerErr != nil {
					c.t.Fatalf("failed to parse response: %s", innerErr)
				}
				break
			default:
				if bodyString, ok := mockRes.Body.(string); ok {
					bodyBytes = []byte(bodyString)
				}
			}
			body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		return &http.Response{
			StatusCode: mockRes.Status,
			Header:     mockRes.Header,
			Body:       body,
		}, nil
	}
	return nil, nil
}

// selectExact returns the first unused interaction
func (c *MockInteractionRoundTripper) selectExact() *ExpectedInteraction {
	if len(c.expectedInteractions) == 0 {
		return nil
	}
	i := c.expectedInteractions[0]
	c.expectedInteractions = c.expectedInteractions[1:]
	return i
}

// selectFirstMatch returns the first interaction that matches the request
func (c *MockInteractionRoundTripper) selectFirstMatch(req *http.Request) *ExpectedInteraction {
	for _, interaction := range c.expectedInteractions {
		if interaction.matches(req) {
			return interaction
		}
	}
	return nil
}

func (c *MockInteractionRoundTripper) ExpectRequest(req TestRequest) *ExpectedInteraction {
	c.m.Lock()
	defer c.m.Unlock()
	e := &ExpectedInteraction{
		request:           req,
		ignoreQueryParams: false,
	}
	c.expectedInteractions = append(c.expectedInteractions, e)
	return e
}

func (c *MockInteractionRoundTripper) Reset() {
	c.m.Lock()
	defer c.m.Unlock()
	c.expectedInteractions = make([]*ExpectedInteraction, 0)
}
