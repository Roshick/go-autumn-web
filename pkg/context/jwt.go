package context

import (
	"context"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func GetJWT(ctx context.Context) jwt.Token {
	token := GetValue[jwt.Token](ctx)
	if token != nil {
		return *token
	}
	return nil
}

func WithJWT(ctx context.Context, token jwt.Token) context.Context {
	return WithValue(ctx, token)
}
