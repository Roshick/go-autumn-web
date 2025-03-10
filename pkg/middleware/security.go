package middleware

import (
	"context"
	"github.com/Roshick/go-autumn-web/pkg/header"
	"github.com/go-chi/render"
	"net/http"
	"strings"
)

// RequireAuthorization //

type AuthorizationFn func(*http.Request) (context.Context, bool)

type RequireAuthorizationOptions struct {
	AuthorizationFn AuthorizationFn
	ErrorResponse   render.Renderer
}

func RequireAuthorization(options RequireAuthorizationOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			if ctx, ok := options.AuthorizationFn(req); ok {
				next.ServeHTTP(w, req.WithContext(ctx))
				return
			}
			if err := render.Render(w, req, options.ErrorResponse); err != nil {
				panic(err)
			}
		}
		return http.HandlerFunc(fn)
	}
}

// HandleCORS //

type HandleCORSOptions struct {
	AllowOrigin             string
	AdditionalAllowHeaders  []string
	AdditionalExposeHeaders []string
}

func HandleCORS(options HandleCORSOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set(header.AccessControlAllowOrigin, options.AllowOrigin)

			w.Header().Set(header.AccessControlAllowMethods, strings.Join([]string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
			}, ", "))

			w.Header().Set(header.AccessControlAllowHeaders, strings.Join(append([]string{
				header.Accept,
				header.ContentType,
			}, options.AdditionalAllowHeaders...), ", "))

			w.Header().Set(header.AccessControlAllowCredentials, "true")

			w.Header().Set(header.AccessControlExposeHeaders, strings.Join(append([]string{
				header.CacheControl,
				header.ContentSecurityPolicy,
				header.ContentType,
				header.Location,
			}, options.AdditionalExposeHeaders...), ", "))

			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, req)
		}
		return http.HandlerFunc(fn)
	}
}
