package tracing

import (
	"context"
	"github.com/Roshick/go-autumn-web/contextutils"
)

type RequestID string

func (c RequestID) String() string {
	return string(c)
}

func GetRequestID(ctx context.Context) *string {
	requestID := contextutils.GetValue[RequestID](ctx)
	if requestID != nil {
		requestIDString := (*requestID).String()
		return &requestIDString
	}
	return nil
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestID(requestID), requestID)
}
