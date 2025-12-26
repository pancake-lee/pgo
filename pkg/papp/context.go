package papp

import (
	"context"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

/*
--------------------------------------------------
封装一个请求的上下文

问题1：什么情况下把数据写入context.Context，什么情况另一个结构体持有ctx，然后传递整个结构体

	只传递数据时，直接写入context.Context，提供Set/Get方法
	除了数据，还需要附带逻辑时，使用后者，传递的是一整个有具体功能的对象

--------------------------------------------------
*/
type AppCtx struct {
	context.Context

	// 业务通用传递字段
	UserId int32

	// 需要与ctx绑定的工具对象
	Log *plogger.PLogWarper

	// 生命周期和请求一致的对象
	cache map[string]any // 缓存任意对象，但必须定义key/get/set方法
}

func NewAppCtx(ctx context.Context) *AppCtx {
	appCtx := &AppCtx{
		Context: ctx,
		cache:   make(map[string]any),
	}
	if uid, ok := putil.GetUserIdFromCtx(ctx); ok {
		appCtx.UserId = uid
	}
	appCtx.Log = plogger.GetDefaultLogWarper().WithContext(ctx)
	return appCtx
}
