package middleware

import (
	"context"
	"encoding/json"
	"github.com/go-chi/render"
	"net/http"
)

// ParseRequestBody //

type requestBodyContextKey[B any] struct{}

func RequestBodyFromContext[B any](ctx context.Context) B {
	return ctx.Value(requestBodyContextKey[B]{}).(B)
}

type ParseRequestBodyOptions struct {
	ErrorResponse render.Renderer
}

func ParseRequestBody[B any](options ParseRequestBodyOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			body := new(B)
			if err := json.NewDecoder(req.Body).Decode(body); err != nil {
				if err = render.Render(w, req, options.ErrorResponse); err != nil {
					panic(err)
				}
				return
			}
			ctx := context.WithValue(req.Context(), requestBodyContextKey[B]{}, *body)
			next.ServeHTTP(w, req.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// RequireHeader //

type RequireHeaderOptions struct {
	Header        string
	ErrorResponse render.Renderer
}

func RequireHeader(options RequireHeaderOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			if req.Header.Get(options.Header) == "" {
				if err := render.Render(w, req, options.ErrorResponse); err != nil {
					panic(err)
				}
				return
			}
			next.ServeHTTP(w, req)
		}
		return http.HandlerFunc(fn)
	}
}
