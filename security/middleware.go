package security

import (
	"fmt"
	"github.com/Roshick/go-autumn-web/header"
	"net/http"
	"strings"
)

// CORSMiddleware //

type CORSMiddlewareOptions struct {
	AllowOrigin             string
	AllowCredentials        bool
	MaxAge                  int
	AdditionalAllowHeaders  []string
	AdditionalExposeHeaders []string
}

func DefaultCORSMiddlewareOptions() *CORSMiddlewareOptions {
	return &CORSMiddlewareOptions{
		AllowOrigin:             "*",
		AllowCredentials:        false, // SECURITY FIX: Cannot be true with wildcard origin
		MaxAge:                  3600,  // Cache preflight for 1 hour
		AdditionalAllowHeaders:  []string{},
		AdditionalExposeHeaders: []string{},
	}
}

func NewCORSMiddleware(opts *CORSMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultCORSMiddlewareOptions()
	}

	if opts.AllowOrigin == "*" && opts.AllowCredentials {
		opts.AllowCredentials = false
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set(header.AccessControlAllowOrigin, opts.AllowOrigin)

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
			}, opts.AdditionalAllowHeaders...), ", "))

			if opts.AllowCredentials && opts.AllowOrigin != "*" {
				w.Header().Set(header.AccessControlAllowCredentials, "true")
			}

			w.Header().Set(header.AccessControlExposeHeaders, strings.Join(append([]string{
				header.CacheControl,
				header.ContentSecurityPolicy,
				header.ContentType,
				header.Location,
			}, opts.AdditionalExposeHeaders...), ", "))

			if req.Method == http.MethodOptions {
				// Add preflight cache control
				if opts.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", opts.MaxAge))
				}
				// FIX: Use 204 No Content instead of 200 OK for OPTIONS
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, req)
		}
		return http.HandlerFunc(fn)
	}
}
