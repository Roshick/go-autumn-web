package validation

import (
	"context"
	"encoding/json"
	weberrors "github.com/Roshick/go-autumn-web/errors"
	"github.com/go-chi/render"
	"net/http"
)

// ContextRequestBodyMiddleware //

type ContextRequestBodyMiddlewareOptions struct {
	ErrorResponse render.Renderer
}

func DefaultContextRequestBodyMiddlewareOptions() *ContextRequestBodyMiddlewareOptions {
	return &ContextRequestBodyMiddlewareOptions{
		ErrorResponse: weberrors.NewInvalidRequestBodyResponse(),
	}
}

func NewContextRequestBodyMiddleware[B any](opts *ContextRequestBodyMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultContextRequestBodyMiddlewareOptions()
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			body := new(B)
			if err := json.NewDecoder(req.Body).Decode(body); err != nil {
				if err = render.Render(w, req, opts.ErrorResponse); err != nil {
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

// RequiredHeaderMiddleware //

type RequiredHeaderMiddlewareOptions struct {
	ErrorResponse render.Renderer
}

func DefaultRequiredHeaderMiddlewareOptions() *RequiredHeaderMiddlewareOptions {
	return &RequiredHeaderMiddlewareOptions{
		ErrorResponse: weberrors.NewMissingRequiredHeaderResponse(),
	}
}

func NewRequiredHeaderMiddleware(headerName string, opts *RequiredHeaderMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultRequiredHeaderMiddlewareOptions()
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			if req.Header.Get(headerName) == "" {
				if err := render.Render(w, req, opts.ErrorResponse); err != nil {
					panic(err)
				}
				return
			}
			next.ServeHTTP(w, req)
		}
		return http.HandlerFunc(fn)
	}
}
