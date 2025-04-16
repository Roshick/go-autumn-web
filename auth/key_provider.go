package auth

import (
	"context"
	"fmt"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
)

func NewRemoteKeySetProvider(keySetURL string, f jwk.Fetcher, options ...jwk.FetchOption) jws.KeyProvider {
	options = append(append([]jwk.FetchOption(nil), jwk.WithFetchWhitelist(jwk.NewBlockAllWhitelist())), options...)

	whitelist := jwk.NewMapWhitelist()
	whitelist.Add(keySetURL)
	options = append(options, jwk.WithFetchWhitelist(whitelist))

	return &RemoteKeySetProvider{
		keySetURL: keySetURL,
		fetcher:   f,
		options:   options,
	}
}

type RemoteKeySetProvider struct {
	keySetURL string
	fetcher   jwk.Fetcher
	options   []jwk.FetchOption
}

func (p RemoteKeySetProvider) FetchKeys(ctx context.Context, sink jws.KeySink, sig *jws.Signature, _ *jws.Message) error {
	if p.fetcher == nil {
		p.fetcher = jwk.FetchFunc(jwk.Fetch)
	}

	kid, ok := sig.ProtectedHeaders().KeyID()
	if !ok {
		return fmt.Errorf(`use of remote key set requires that the payload contains a "kid" field in the protected header`)
	}

	set, err := p.fetcher.Fetch(ctx, p.keySetURL, p.options...)
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
