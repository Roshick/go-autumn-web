package security

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	aucontext "github.com/Roshick/go-autumn-web/pkg/context"
	"github.com/Roshick/go-autumn-web/pkg/middleware"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"net/http"
)

type AllowAuthorizedUserOptions struct {
	AllowBasicAuthUserOptions
	AllowBearerTokenUserOptions
}

func AllowAuthorizedUser(options AllowAuthorizedUserOptions) middleware.AuthorizationFn {
	allowBasicAuthUser := AllowBasicAuthUser(options.AllowBasicAuthUserOptions)
	allowBearerTokenUser := AllowBearerTokenUser(options.AllowBearerTokenUserOptions)

	return func(req *http.Request) (context.Context, bool) {
		if _, _, ok := req.BasicAuth(); ok {
			return allowBasicAuthUser(req)
		}
		return allowBearerTokenUser(req)
	}
}

type AllowBasicAuthUserOptions struct {
	Username string
	Password string
}

func AllowBasicAuthUser(options AllowBasicAuthUserOptions) middleware.AuthorizationFn {
	isBasicAuthUserCredentials := func(username string, password string) bool {
		if username == "" || password == "" {
			return false
		}

		expectedUsernameHash := sha256.Sum256([]byte(options.Username))
		expectedPasswordHash := sha256.Sum256([]byte(options.Password))

		usernameHash := sha256.Sum256([]byte(username))
		passwordHash := sha256.Sum256([]byte(password))

		usernameMatch := subtle.ConstantTimeCompare(expectedUsernameHash[:], usernameHash[:]) == 1
		passwordMatch := subtle.ConstantTimeCompare(expectedPasswordHash[:], passwordHash[:]) == 1

		return usernameMatch && passwordMatch
	}

	return func(req *http.Request) (context.Context, bool) {
		ctx := req.Context()

		username, password, ok := req.BasicAuth()
		if !ok {
			return ctx, false
		}
		return ctx, isBasicAuthUserCredentials(username, password)
	}
}

type AllowBearerTokenUserOptions struct {
	ParseOptions []jwt.ParseOption
}

func AllowBearerTokenUser(options AllowBearerTokenUserOptions) middleware.AuthorizationFn {
	return func(req *http.Request) (context.Context, bool) {
		ctx := req.Context()

		token, err := jwt.ParseRequest(req, options.ParseOptions...)
		if err != nil {
			return ctx, false
		}
		ctx = aucontext.WithJWT(ctx, token)

		return ctx, true
	}
}
