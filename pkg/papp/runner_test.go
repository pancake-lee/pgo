package papp

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunner_RunMax(t *testing.T) {
	r := NewRunner("test")

	t.Run("concurrency", func(t *testing.T) {
		var executed int64
		maxRun := int64(10)
		fn := func() {
			atomic.AddInt64(&executed, 1)
		}

		runWrapper := r.RunMax(maxRun, fn)

		var wg sync.WaitGroup
		// 启动 100 个协程，每个协程调用 10 次
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					runWrapper()
				}
			}()
		}
		wg.Wait()

		if executed != maxRun {
			t.Errorf("RunMax executed %d times, expected %d", executed, maxRun)
		}
	})
}

func TestRunner_RunEvery(t *testing.T) {
	r := NewRunner("test")

	t.Run("logic", func(t *testing.T) {
		var executed int64
		n := int64(3)
		fn := func() {
			atomic.AddInt64(&executed, 1)
		}

		runWrapper := r.RunEvery(n, fn)

		// 调用 10 次，应该执行 3 次 (3, 6, 9)
		for i := 0; i < 10; i++ {
			runWrapper()
		}

		if executed != 3 {
			t.Errorf("RunEvery executed %d times, expected 3", executed)
		}
	})

	t.Run("concurrency", func(t *testing.T) {
		var executed int64
		n := int64(5)
		totalCalls := 1000
		fn := func() {
			atomic.AddInt64(&executed, 1)
		}

		runWrapper := r.RunEvery(n, fn)

		var wg sync.WaitGroup
		for i := 0; i < totalCalls; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				runWrapper()
			}()
		}
		wg.Wait()

		expected := int64(totalCalls) / n
		if executed != expected {
			t.Errorf("RunEvery executed %d times, expected %d", executed, expected)
		}
	})
}

func TestRunner_RunInterval(t *testing.T) {
	r := NewRunner("test")
	var executed int64
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fn := func() {
		atomic.AddInt64(&executed, 1)
	}

	go r.RunInterval(ctx, 10*time.Millisecond, fn)

	// 等待足够长的时间让它运行几次
	time.Sleep(55 * time.Millisecond)
	cancel()
	// 给一点时间让协程退出
	time.Sleep(10 * time.Millisecond)

	finalCount := atomic.LoadInt64(&executed)
	// 0ms(首次), 10ms, 20ms, 30ms, 40ms, 50ms -> 大约 6 次
	// 由于时间调度不一定精确，这里只需要验证它执行了多次并且停下来了即可
	if finalCount < 4 {
		t.Errorf("RunInterval executed too few times: %d", finalCount)
	}
}

func TestRunner_RunRetry(t *testing.T) {
	r := NewRunner("test")

	t.Run("success_eventually", func(t *testing.T) {
		var calls int
		err := r.RunRetry(3, time.Millisecond, func() error {
			calls++
			if calls < 2 {
				return errors.New("fail")
			}
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if calls != 2 {
			t.Errorf("expected 2 calls, got %d", calls)
		}
	})

	t.Run("fail_all_retry", func(t *testing.T) {
		var calls int
		err := r.RunRetry(3, time.Millisecond, func() error {
			calls++
			return errors.New("fail")
		})
		if err == nil {
			t.Error("expected error, got nil")
		}
		if calls != 4 { // 1次初始 + 3次重试
			t.Errorf("expected 4 calls, got %d", calls)
		}
	})
}

func TestRunner_RunTimeout(t *testing.T) {
	r := NewRunner("test")

	t.Run("timeout", func(t *testing.T) {
		err := r.RunTimeout(10*time.Millisecond, func() error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
		if err != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		expectedErr := errors.New("business error")
		err := r.RunTimeout(50*time.Millisecond, func() error {
			return expectedErr
		})
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}
