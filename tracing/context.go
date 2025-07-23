package tracing

import (
	"context"
	"github.com/Roshick/go-autumn-web/contextutils"
)

type RequestID string

func RequestIDFromContext(ctx context.Context) *string {
	requestID := contextutils.GetValue[RequestID](ctx)
	if requestID != nil {
		requestIDString := string(*requestID) // Direct string conversion instead of String() method
		return &requestIDString
	}
	return nil
}

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return contextutils.WithValue(ctx, RequestID(requestID))
}
