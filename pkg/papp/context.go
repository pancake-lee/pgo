package papp

import "context"

// ctxKeyUserId is the type used for storing user ID in context to avoid key collisions.
type ctxKeyUserId struct{}

func SetUserIdToCtx(ctx context.Context, userId int32) context.Context {
	return context.WithValue(ctx, ctxKeyUserId{}, userId)
}
func GetUserIdFromCtx(ctx context.Context) (int32, bool) {
	v, ok := ctx.Value(ctxKeyUserId{}).(int32)
	return v, ok
}
