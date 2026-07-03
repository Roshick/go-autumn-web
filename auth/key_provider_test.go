package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKeyID = "test-key-id"

type fetcherFunc func(ctx context.Context, url string) (jwk.Set, error)

func (f fetcherFunc) Fetch(ctx context.Context, url string) (jwk.Set, error) {
	return f(ctx, url)
}

type testKeySink struct {
	alg jwa.SignatureAlgorithm
	key any
}

func (s *testKeySink) Key(alg jwa.SignatureAlgorithm, key any) {
	s.alg = alg
	s.key = key
}

func newTestKeyPair(t *testing.T) (jwk.Key, jwk.Set) {
	t.Helper()

	rawKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKey, err := jwk.Import[jwk.Key](rawKey)
	require.NoError(t, err)
	require.NoError(t, privateKey.Set(jwk.KeyIDKey, testKeyID))

	publicKey, err := jwk.PublicKeyOf(privateKey)
	require.NoError(t, err)

	set := jwk.NewSet()
	require.NoError(t, set.AddKey(publicKey))

	return privateKey, set
}

func newTestSignature(t *testing.T, headers map[string]any) *jws.Signature {
	t.Helper()

	hdrs := jws.NewHeaders()
	for key, value := range headers {
		require.NoError(t, hdrs.Set(key, value))
	}
	return jws.NewSignature().SetProtectedHeaders(hdrs)
}

func TestNewRemoteKeySetProvider(t *testing.T) {
	t.Run("with nil fetcher uses default fetcher", func(t *testing.T) {
		provider := NewRemoteKeySetProvider("https://example.com/jwks.json", nil)
		require.NotNil(t, provider)

		remoteProvider, ok := provider.(*RemoteKeySetProvider)
		require.True(t, ok)
		assert.NotNil(t, remoteProvider.fetcher)
	})

	t.Run("with custom fetcher", func(t *testing.T) {
		fetcher := fetcherFunc(func(context.Context, string) (jwk.Set, error) {
			return jwk.NewSet(), nil
		})

		provider := NewRemoteKeySetProvider("https://example.com/jwks.json", fetcher)
		require.NotNil(t, provider)
	})
}

func TestRemoteKeySetProviderFetchKeys(t *testing.T) {
	_, set := newTestKeyPair(t)

	setFetcher := fetcherFunc(func(context.Context, string) (jwk.Set, error) {
		return set, nil
	})

	t.Run("sinks key for matching kid and alg", func(t *testing.T) {
		provider := NewRemoteKeySetProvider("https://example.com/jwks.json", setFetcher)

		sig := newTestSignature(t, map[string]any{
			jws.KeyIDKey:     testKeyID,
			jws.AlgorithmKey: jwa.RS256(),
		})
		sink := &testKeySink{}

		err := provider.FetchKeys(t.Context(), sink, sig, nil)

		require.NoError(t, err)
		assert.Equal(t, jwa.RS256(), sink.alg)
		assert.NotNil(t, sink.key)
	})

	t.Run("fails without kid in protected header", func(t *testing.T) {
		provider := NewRemoteKeySetProvider("https://example.com/jwks.json", setFetcher)

		sig := newTestSignature(t, map[string]any{
			jws.AlgorithmKey: jwa.RS256(),
		})
		sink := &testKeySink{}

		err := provider.FetchKeys(t.Context(), sink, sig, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `"kid"`)
		assert.Nil(t, sink.key)
	})

	t.Run("fails when fetching the key set fails", func(t *testing.T) {
		fetcher := fetcherFunc(func(context.Context, string) (jwk.Set, error) {
			return nil, errors.New("connection refused")
		})
		provider := NewRemoteKeySetProvider("https://example.com/jwks.json", fetcher)

		sig := newTestSignature(t, map[string]any{
			jws.KeyIDKey:     testKeyID,
			jws.AlgorithmKey: jwa.RS256(),
		})
		sink := &testKeySink{}

		err := provider.FetchKeys(t.Context(), sink, sig, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch")
		assert.Nil(t, sink.key)
	})

	t.Run("ignores unknown kid", func(t *testing.T) {
		provider := NewRemoteKeySetProvider("https://example.com/jwks.json", setFetcher)

		sig := newTestSignature(t, map[string]any{
			jws.KeyIDKey:     "unknown-key-id",
			jws.AlgorithmKey: jwa.RS256(),
		})
		sink := &testKeySink{}

		err := provider.FetchKeys(t.Context(), sink, sig, nil)

		require.NoError(t, err)
		assert.Nil(t, sink.key)
	})

	t.Run("ignores key with mismatching algorithm", func(t *testing.T) {
		provider := NewRemoteKeySetProvider("https://example.com/jwks.json", setFetcher)

		sig := newTestSignature(t, map[string]any{
			jws.KeyIDKey:     testKeyID,
			jws.AlgorithmKey: jwa.ES256(),
		})
		sink := &testKeySink{}

		err := provider.FetchKeys(t.Context(), sink, sig, nil)

		require.NoError(t, err)
		assert.Nil(t, sink.key)
	})

	t.Run("zero value provider falls back to default fetcher", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(set))
		}))
		defer server.Close()

		provider := RemoteKeySetProvider{keySetURL: server.URL}

		sig := newTestSignature(t, map[string]any{
			jws.KeyIDKey:     testKeyID,
			jws.AlgorithmKey: jwa.RS256(),
		})
		sink := &testKeySink{}

		err := provider.FetchKeys(t.Context(), sink, sig, nil)

		require.NoError(t, err)
		assert.Equal(t, jwa.RS256(), sink.alg)
		assert.NotNil(t, sink.key)
	})
}

func TestRemoteKeySetProviderVerifiesSignedToken(t *testing.T) {
	privateKey, set := newTestKeyPair(t)

	token := jwt.New()
	require.NoError(t, token.Set(jwt.SubjectKey, "test-subject"))

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), privateKey))
	require.NoError(t, err)

	fetcher := fetcherFunc(func(context.Context, string) (jwk.Set, error) {
		return set, nil
	})
	provider := NewRemoteKeySetProvider("https://example.com/jwks.json", fetcher)

	payload, err := jws.Verify(signed, jws.WithKeyProvider(provider))

	require.NoError(t, err)
	assert.Contains(t, string(payload), "test-subject")
}
