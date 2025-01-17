package context

import (
	"context"
)

type contextKey[B any] struct{}

func WithValue[B any](ctx context.Context, value B) context.Context {
	return context.WithValue(ctx, contextKey[B]{}, value)
}

func GetValue[B any](ctx context.Context) *B {
	if value := ctx.Value(contextKey[B]{}); value != nil {
		typedValue := value.(B)
		return &typedValue
	}
	return nil
}

func MustGetValue[B any](ctx context.Context) B {
	return ctx.Value(contextKey[B]{}).(B)
}
