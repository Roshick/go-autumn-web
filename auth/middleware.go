package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"github.com/Roshick/go-autumn-web/header"
	"github.com/go-chi/render"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"net/http"
	"strings"
)

// RequireAuthorization //

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

func AllowBearerTokenUser(options AllowBearerTokenUserOptions) AuthorizationFn {
	return func(req *http.Request) bool {
		_, err := jwt.ParseRequest(req, options.ParseOptions...)
		if err != nil {
			return false
		}
		return true
	}
}

type RequireAuthorizationOptions struct {
	AuthorizationFns []AuthorizationFn
	ErrorResponse    render.Renderer
}

func RequireAuthorization(options RequireAuthorizationOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			for _, authFn := range options.AuthorizationFns {
				if authFn(req) {
					next.ServeHTTP(w, req)
					return
				}
			}
			if err := render.Render(w, req, options.ErrorResponse); err != nil {
				panic(err)
			}
		}
		return http.HandlerFunc(fn)
	}
}

// AddJWTToContext //

type AddJWTToContextOptions struct {
	ErrorResponse render.Renderer
}

func AddJWTToContext(options AddJWTToContextOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			authorization := req.Header.Get(header.Authorization)
			if authorization == "" || !strings.HasPrefix(authorization, "Bearer ") {
				next.ServeHTTP(w, req)
				return
			}

			token, err := jwt.ParseRequest(req, jwt.WithVerify(false))
			if err != nil {
				if innerErr := render.Render(w, req, options.ErrorResponse); innerErr != nil {
					panic(innerErr)
				}
				return
			}
			next.ServeHTTP(w, req.WithContext(ContextWithJWT(req.Context(), token)))
		}
		return http.HandlerFunc(fn)
	}
}
