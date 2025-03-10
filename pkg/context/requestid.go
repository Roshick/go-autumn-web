package context

import (
	"context"
)

type RequestID string

func (c RequestID) String() string {
	return string(c)
}

func GetRequestID(ctx context.Context) *string {
	requestID := GetValue[RequestID](ctx)
	if requestID != nil {
		requestIDString := (*requestID).String()
		return &requestIDString
	}
	return nil
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestID(requestID), requestID)
}
