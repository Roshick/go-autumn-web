package auth

import (
	"context"
	"github.com/Roshick/go-autumn-web/contextutils"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func JWTFromContext(ctx context.Context) jwt.Token {
	token := contextutils.GetValue[jwt.Token](ctx)
	if token != nil {
		return *token
	}
	return nil
}

func ContextWithJWT(ctx context.Context, token jwt.Token) context.Context {
	return contextutils.WithValue(ctx, token)
}
