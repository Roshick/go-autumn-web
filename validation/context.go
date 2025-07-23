package validation

import "context"

type requestBodyContextKey[B any] struct{}

func RequestBodyFromContext[B any](ctx context.Context) B {
	value := ctx.Value(requestBodyContextKey[B]{})
	if value == nil {
		var zero B
		return zero
	}
	return value.(B)
}
