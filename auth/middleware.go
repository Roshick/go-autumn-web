package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	weberrors "github.com/Roshick/go-autumn-web/errors"
	"github.com/Roshick/go-autumn-web/header"
	"github.com/go-chi/render"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"net/http"
	"strings"
)

// AuthorizationMiddleware //

type AuthorizationFn func(*http.Request) bool

type AllowBasicAuthUserOptions struct {
	Username string
	Password string
}

func AllowBasicAuthUser(options AllowBasicAuthUserOptions) AuthorizationFn {
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

	return func(req *http.Request) bool {
		username, password, ok := req.BasicAuth()
		if !ok {
			return false
		}
		return isBasicAuthUserCredentials(username, password)
	}
}

type AllowBearerTokenUserOptions struct {
	ParseOptions []jwt.ParseOption
}

func AllowBearerTokenUser(opts AllowBearerTokenUserOptions) AuthorizationFn {
	return func(req *http.Request) bool {
		_, err := jwt.ParseRequest(req, opts.ParseOptions...)
		if err != nil {
			return false
		}
		return true
	}
}

func RejectAll() AuthorizationFn {
	return func(req *http.Request) bool {
		return false
	}
}

type AuthorizationMiddlewareOptions struct {
	AuthorizationFns []AuthorizationFn
	ErrorResponse    render.Renderer
}

func DefaultAuthorizationMiddlewareOptions() *AuthorizationMiddlewareOptions {
	return &AuthorizationMiddlewareOptions{
		AuthorizationFns: []AuthorizationFn{RejectAll()},
		ErrorResponse:    weberrors.NewAuthenticationRequiredResponse(),
	}
}

func NewAuthorizationMiddleware(opts *AuthorizationMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultAuthorizationMiddlewareOptions()
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			for _, authFn := range opts.AuthorizationFns {
				if authFn(req) {
					next.ServeHTTP(w, req)
					return
				}
			}
			if err := render.Render(w, req, opts.ErrorResponse); err != nil {
				panic(err)
			}
		}
		return http.HandlerFunc(fn)
	}
}

// ContextJWTMiddleware //

type ContextJWTMiddlewareOptions struct {
	ErrorResponse render.Renderer
}

func DefaultContextJWTMiddlewareOptions() *ContextJWTMiddlewareOptions {
	return &ContextJWTMiddlewareOptions{
		ErrorResponse: weberrors.NewAuthenticationRequiredResponse(),
	}
}

func NewContextJWTMiddleware(opts *ContextJWTMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultContextJWTMiddlewareOptions()
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			authorization := req.Header.Get(header.Authorization)
			if authorization == "" || !strings.HasPrefix(authorization, "Bearer ") {
				next.ServeHTTP(w, req)
				return
			}

			token, err := jwt.ParseRequest(req, jwt.WithVerify(false))
			if err != nil {
				if innerErr := render.Render(w, req, opts.ErrorResponse); innerErr != nil {
					panic(innerErr)
				}
				return
			}
			next.ServeHTTP(w, req.WithContext(ContextWithJWT(req.Context(), token)))
		}
		return http.HandlerFunc(fn)
	}
}
