package testutils

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

type ExpectedInteraction struct {
	request  TestRequest
	response *TestResponse
}

func (r *ExpectedInteraction) WillReturnResponse(response *TestResponse) {
	r.response = response
}

type MockInteractionRoundTripper struct {
	t                    *testing.T
	expectedInteractions []*ExpectedInteraction
}

func NewMockInteractionRoundTripper(t *testing.T) *MockInteractionRoundTripper {
	return &MockInteractionRoundTripper{
		t:                    t,
		expectedInteractions: make([]*ExpectedInteraction, 0),
	}
}

func (c *MockInteractionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	require.NotEmpty(c.t, c.expectedInteractions)

	var next *ExpectedInteraction
	next, c.expectedInteractions = c.expectedInteractions[0], c.expectedInteractions[1:]

	if next.request.Method != "" {
		require.Equal(c.t, next.request.Method, req.Method)
	}
	if next.request.URL != "" {
		require.Equal(c.t, next.request.URL, req.URL.String())
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

func (c *MockInteractionRoundTripper) ExpectRequest(req TestRequest) *ExpectedInteraction {
	e := &ExpectedInteraction{
		request: req,
	}
	c.expectedInteractions = append(c.expectedInteractions, e)
	return e
}

func (c *MockInteractionRoundTripper) Reset() {
	c.expectedInteractions = make([]*ExpectedInteraction, 0)
}
