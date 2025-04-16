package resiliency

import (
	"context"
	"errors"
	"github.com/Roshick/go-autumn-web/logging"
	"github.com/go-chi/render"
	"net/http"
	"runtime/debug"
	"time"

	aulogging "github.com/StephanHCB/go-autumn-logging"
)

// TimeoutContext //

type TimeoutContextOptions struct {
	Timeout       time.Duration
	ErrorResponse render.Renderer
}

func TimeoutContext(options TimeoutContextOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			ctx, cancel := context.WithTimeout(ctx, options.Timeout)
			defer cancel()

			next.ServeHTTP(w, req.WithContext(ctx))

			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				if err := render.Render(w, req, options.ErrorResponse); err != nil {
					panic(err)
				}
			}
		}
		return http.HandlerFunc(fn)
	}
}

// RecoverPanic //

type RecoverPanicOptions struct {
	ErrorResponse render.Renderer
}

func RecoverPanic(options RecoverPanicOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			defer func() {
				ctx := req.Context()
				rvr := recover()
				if rvr != nil && rvr != http.ErrAbortHandler {
					aulogging.Logger.Ctx(ctx).Error().With(logging.LogFieldStackTrace, string(debug.Stack())).Print("recovered from panic")
					if err := render.Render(w, req, options.ErrorResponse); err != nil {
						panic(err)
					}
				}
			}()

			next.ServeHTTP(w, req)
		}
		return http.HandlerFunc(fn)
	}
}
