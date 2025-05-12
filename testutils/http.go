package testutils

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"testing"
)

type TestRequest struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Header http.Header `json:"header"`
	Body   any         `json:"body,omitempty"`
}

type TestResponse struct {
	Status int         `json:"status"`
	Header http.Header `json:"header"`
	Body   any         `json:"body,omitempty"`
}

func (r *TestResponse) RequireEqual(t *testing.T, other *TestResponse) *TestResponse {
	require.Equal(t, other, r)
	return r
}

func (r *TestResponse) RequireEqualStatus(t *testing.T, other *TestResponse) *TestResponse {
	require.Equal(t, other.Status, r.Status)
	return r
}

func (r *TestResponse) RequireEqualHeader(t *testing.T, other *TestResponse) *TestResponse {
	require.Equal(t, other.Header, r.Header)
	return r
}

func (r *TestResponse) RequireEqualBody(t *testing.T, other *TestResponse) *TestResponse {
	require.Equal(t, other.Body, r.Body)
	return r
}

func (r *TestResponse) RequireContainsHeader(t *testing.T, key string, value string) *TestResponse {
	require.Contains(t, r.Header, key)
	require.Contains(t, r.Header[key], value)
	return r
}

func MustParseResponse(t *testing.T, res *http.Response) *TestResponse {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %s", err)
	}
	defer res.Body.Close()

	var parsedBody any
	switch res.Header.Get("Content-Type") {
	case "application/json":
		if innerErr := json.Unmarshal(body, &parsedBody); innerErr != nil {
			t.Fatalf("failed to parse response: %s", err)
		}
	default:
		parsedBody = string(body)
	}

	return &TestResponse{
		Status: res.StatusCode,
		Header: res.Header,
		Body:   parsedBody,
	}
}

func MustReadResponseFromFile(t *testing.T, path string) *TestResponse {
	jsonBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read response: %s", err)
	}

	response := TestResponse{}
	err = json.Unmarshal(jsonBytes, &response)
	if err != nil {
		t.Fatalf("failed to parse response: %s", err)
	}
	return &response
}

func PerformHTTPRequest(t *testing.T, req TestRequest) *TestResponse {
	bodyBytes, err := json.Marshal(req.Body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %s", err.Error())
	}

	request, err := http.NewRequest(req.Method, req.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("failed to create request: %s", err.Error())
	}

	request.Header = req.Header

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("failed to perform request: %s", err.Error())
	}

	return MustParseResponse(t, res)
}
