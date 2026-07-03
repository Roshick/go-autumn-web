package testutils

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestResponseRequireEqual(t *testing.T) {
	response := &TestResponse{
		Status: http.StatusOK,
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   map[string]any{"key": "value"},
	}
	other := &TestResponse{
		Status: http.StatusOK,
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   map[string]any{"key": "value"},
	}

	result := response.RequireEqual(t, other)
	assert.Same(t, response, result)
}

func TestTestResponseRequireEqualStatus(t *testing.T) {
	response := &TestResponse{Status: http.StatusOK}
	other := &TestResponse{Status: http.StatusOK, Body: "different body"}

	result := response.RequireEqualStatus(t, other)
	assert.Same(t, response, result)
}

func TestTestResponseRequireEqualHeader(t *testing.T) {
	response := &TestResponse{Header: http.Header{"Content-Type": []string{"text/plain"}}}
	other := &TestResponse{Header: http.Header{"Content-Type": []string{"text/plain"}}}

	result := response.RequireEqualHeader(t, other)
	assert.Same(t, response, result)
}

func TestTestResponseRequireEqualBody(t *testing.T) {
	response := &TestResponse{Body: "some body"}
	other := &TestResponse{Status: http.StatusTeapot, Body: "some body"}

	result := response.RequireEqualBody(t, other)
	assert.Same(t, response, result)
}

func TestTestResponseRequireContainsHeader(t *testing.T) {
	response := &TestResponse{
		Header: http.Header{"X-Custom-Header": []string{"value-a", "value-b"}},
	}

	result := response.RequireContainsHeader(t, "X-Custom-Header", "value-b")
	assert.Same(t, response, result)
}

func TestMustParseResponse(t *testing.T) {
	t.Run("json body", func(t *testing.T) {
		res := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"key":"value"}`))),
		}

		response := MustParseResponse(t, res)

		require.NotNil(t, response)
		assert.Equal(t, http.StatusOK, response.Status)
		assert.Equal(t, map[string]any{"key": "value"}, response.Body)
	})

	t.Run("plain text body", func(t *testing.T) {
		res := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       io.NopCloser(bytes.NewReader([]byte("plain text"))),
		}

		response := MustParseResponse(t, res)

		require.NotNil(t, response)
		assert.Equal(t, http.StatusOK, response.Status)
		assert.Equal(t, "plain text", response.Body)
	})
}

func TestMustReadResponseFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "response.json")
	content := []byte(`{"status":200,"header":{"Content-Type":["application/json"]},"body":{"key":"value"}}`)
	require.NoError(t, os.WriteFile(path, content, 0o600))

	response := MustReadResponseFromFile(t, path)

	require.NotNil(t, response)
	assert.Equal(t, http.StatusOK, response.Status)
	assert.Equal(t, http.Header{"Content-Type": []string{"application/json"}}, response.Header)
	assert.Equal(t, map[string]any{"key": "value"}, response.Body)
}

func TestPerformHTTPRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "header-value", r.Header.Get("X-Test-Header"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"key":"value"}`, string(body))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"result":"created"}`))
	}))
	defer server.Close()

	response := PerformHTTPRequest(t, TestRequest{
		Method: http.MethodPost,
		URL:    server.URL,
		Header: http.Header{"X-Test-Header": []string{"header-value"}},
		Body:   map[string]any{"key": "value"},
	})

	require.NotNil(t, response)
	assert.Equal(t, http.StatusCreated, response.Status)
	assert.Equal(t, map[string]any{"result": "created"}, response.Body)
}
