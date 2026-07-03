package auth

import (
	"context"
	"fmt"

	"github.com/jwx-go/jwkfetch/v4"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
)

func NewRemoteKeySetProvider(keySetURL string, f jwk.Fetcher) jws.KeyProvider {
	if f == nil {
		f = newDefaultFetcher(keySetURL)
	}

	return &RemoteKeySetProvider{
		keySetURL: keySetURL,
		fetcher:   f,
	}
}

func newDefaultFetcher(keySetURL string) jwk.Fetcher {
	whitelist := jwkfetch.NewMapWhitelist().Add(keySetURL)
	return jwkfetch.NewClient(jwkfetch.WithWhitelist(whitelist))
}

type RemoteKeySetProvider struct {
	keySetURL string
	fetcher   jwk.Fetcher
}

func (p RemoteKeySetProvider) FetchKeys(ctx context.Context, sink jws.KeySink, sig *jws.Signature, _ *jws.Message) error {
	fetcher := p.fetcher
	if fetcher == nil {
		fetcher = newDefaultFetcher(p.keySetURL)
	}

	kid, ok := sig.ProtectedHeaders().KeyID()
	if !ok {
		return fmt.Errorf(`use of remote key set requires that the payload contains a "kid" field in the protected header`)
	}

	set, err := fetcher.Fetch(ctx, p.keySetURL)
	if err != nil {
		return fmt.Errorf(`failed to fetch %q: %w`, p.keySetURL, err)
	}

	key, ok := set.LookupKeyID(kid)
	if !ok {
		// It is not an error if the key with the kid doesn't exist
		return nil
	}

	algs, err := jws.AlgorithmsForKey(key)
	if err != nil {
		return fmt.Errorf(`failed to get a list of signature methods for key type %s: %w`, key.KeyType(), err)
	}

	hdrAlg, ok := sig.ProtectedHeaders().Algorithm()
	if ok {
		for _, alg := range algs {

			if hdrAlg != alg {
				continue
			}

			sink.Key(alg, key)
			break
		}
	}
	return nil
}
