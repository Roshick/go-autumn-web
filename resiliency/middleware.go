package resiliency

import (
	weberrors "github.com/Roshick/go-autumn-web/errors"
	"github.com/Roshick/go-autumn-web/logging"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/go-chi/render"
	"net/http"
	"runtime/debug"
)

// PanicRecoveryMiddleware //

type RecoveryMiddlewareOptions struct {
	ErrorResponse render.Renderer
}

func DefaultRecoveryMiddlewareOptions() *RecoveryMiddlewareOptions {
	return &RecoveryMiddlewareOptions{
		ErrorResponse: weberrors.NewPanicRecoveryResponse(),
	}
}

func NewPanicRecoveryMiddleware(opts *RecoveryMiddlewareOptions) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultRecoveryMiddlewareOptions()
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			defer func() {
				ctx := req.Context()
				rvr := recover()
				if rvr != nil && rvr != http.ErrAbortHandler {
					aulogging.Logger.Ctx(ctx).Error().With(logging.LogFieldStackTrace, string(debug.Stack())).Print("recovered from panic")
					if err := render.Render(w, req, opts.ErrorResponse); err != nil {
						panic(err)
					}
				}
			}()

			next.ServeHTTP(w, req)
		}
		return http.HandlerFunc(fn)
	}
}
