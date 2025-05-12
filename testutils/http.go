package testutils

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"testing"
)

type TestRequest struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Body   any    `json:"body,omitempty"`
}

type TestResponse struct {
	Status int         `json:"status"`
	Header http.Header `json:"header"`
	Body   any         `json:"body,omitempty"`
}

func (r *TestResponse) RequireEqual(t *testing.T, other *TestResponse) *TestResponse {
	require.Equal(t, r, other)
	return r
}

func (r *TestResponse) RequireEqualStatus(t *testing.T, other *TestResponse) *TestResponse {
	require.Equal(t, r.Status, other.Status)
	return r
}

func (r *TestResponse) RequireEqualHeader(t *testing.T, other *TestResponse) *TestResponse {
	require.Equal(t, r.Header, other.Header)
	return r
}

func (r *TestResponse) RequireEqualBody(t *testing.T, other *TestResponse) *TestResponse {
	require.Equal(t, r.Body, other.Body)
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
