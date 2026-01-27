package papp

import (
	"context"
	"sync/atomic"
	"time"
)

// runner 实现了多种常见的运行模式
type runner struct {
	name string
}

func NewRunner(name string) *runner {
	return &runner{name: name}
}

// RunMax 只运行n次，之后不再运行
func (r *runner) RunMax(n int64, fn func()) func() {
	var count int64
	return func() {
		if atomic.LoadInt64(&count) >= n {
			return
		}
		if atomic.AddInt64(&count, 1) > n {
			return
		}
		fn()
	}
}

// RunEvery 每调用n次，就实际运行一次
func (r *runner) RunEvery(n int64, fn func()) func() {
	if n <= 0 {
		n = 1
	}
	var count int64
	return func() {
		if atomic.AddInt64(&count, 1)%n == 0 {
			fn()
		}
	}
}

// RunInterval 间隔interval时间后调用一次，首次立刻调用
func (r *runner) RunInterval(ctx context.Context, interval time.Duration, fn func()) {
	fn() // 首次立刻调用
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}

// RunRetry 错误时间隔interval时间，重试n次
func (r *runner) RunRetry(n int, interval time.Duration, fn func() error) error {
	var err error
	for i := 0; i <= n; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < n {
			time.Sleep(interval)
		}
	}
	return err
}

// RunTimeout 运行等待timeout后强制结束，返回错误
func (r *runner) RunTimeout(timeout time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
