package putil

import "context"

// 定义context存储的数据
// 比起papp.context.go，这里只定义了基于context.Context的数据存取的key和方法

// --------------------------------------------------
type pgoUserID struct{}

var PgoUserIDKey = pgoUserID{}

func SetUserIdToCtx(ctx context.Context, userId int32) context.Context {
	return context.WithValue(ctx, PgoUserIDKey, userId)
}
func GetUserIdFromCtx(ctx context.Context) (int32, bool) {
	v, ok := ctx.Value(PgoUserIDKey).(int32)
	return v, ok
}

// --------------------------------------------------
type pgoTraceID struct{}

var PgoTraceIDKey = pgoTraceID{}

func SetTraceIdToCtx(ctx context.Context, traceId string) context.Context {
	return context.WithValue(ctx, PgoTraceIDKey, traceId)
}
func GetTraceIdFromCtx(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(PgoTraceIDKey).(string)
	return v, ok
}
