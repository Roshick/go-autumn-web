package testutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Request struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Body   any    `json:"body,omitempty"`
}

type MockResponse struct {
	Body   *string
	Status int
	Header http.Header
	Time   time.Time
}

type ExpectedRequest struct {
	request  Request
	response *MockResponse
}

func (r *ExpectedRequest) WillReturnResponse(response *MockResponse) {
	r.response = response
}

type RequestAsserter struct {
	expectedRequests []*ExpectedRequest
}

func NewRequestAsserter() *RequestAsserter {
	return &RequestAsserter{
		expectedRequests: make([]*ExpectedRequest, 0),
	}
}

func (c *RequestAsserter) RoundTrip(req *http.Request) (*http.Response, error) {
	var next *ExpectedRequest
	next, c.expectedRequests = c.expectedRequests[0], c.expectedRequests[1:]

	if next.request.Method != "" && next.request.Method != req.Method {
		return nil, fmt.Errorf("expected method %s, got %s", next.request.Method, req.Method)
	}
	if next.request.URL != "" && next.request.URL != req.URL.String() {
		return nil, fmt.Errorf("expected url %s, got %s", next.request.URL, req.URL.String())
	}
	if next.request.Body != nil {
		var expectedObject any
		jsonBytes, _ := json.MarshalIndent(next.request.Body, "", "  ")
		err := json.Unmarshal(jsonBytes, &expectedObject)
		if err != nil {
			return nil, err
		}

		defer req.Body.Close()
		read, _ := io.ReadAll(req.Body)

		var actualObject any
		err = json.Unmarshal(read, &actualObject)
		if err != nil {
			return nil, err
		}

		if !assert.ObjectsAreEqual(expectedObject, actualObject) {
			return nil, fmt.Errorf("expected request body does not match received body")
		}
	}

	if next.response != nil {
		mockRes := *next.response
		var body io.ReadCloser
		if mockRes.Body != nil {
			body = io.NopCloser(bytes.NewBuffer([]byte(*mockRes.Body)))
		}
		return &http.Response{
			StatusCode: mockRes.Status,
			Header:     mockRes.Header,
			Body:       body,
		}, nil
	}
	return nil, nil
}

func (c *RequestAsserter) ExpectRequest(request Request) *ExpectedRequest {
	e := &ExpectedRequest{
		request: request,
	}
	c.expectedRequests = append(c.expectedRequests, e)
	return e
}

func (c *RequestAsserter) Reset() {
	c.expectedRequests = make([]*ExpectedRequest, 0)
}

func MustReadResponseFromFile(path string) *MockResponse {
	jsonBytes, err := os.ReadFile(filepath.Join(path))
	if err != nil {
		panic(err)
	}

	parsedResponse := MockResponse{}
	err = json.Unmarshal(jsonBytes, &parsedResponse)
	if err != nil {
		panic(err)
	}

	return &parsedResponse
}
