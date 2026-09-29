package papp

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCheckDependencyWithRetryConfig(t *testing.T) {
	t.Run("success eventually", func(t *testing.T) {
		attempts := 0
		err := checkDependencyWithRetryConfig("RabbitMQ", 3, time.Millisecond, func() error {
			attempts++
			if attempts < 3 {
				return errors.New("connection refused")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if attempts != 3 {
			t.Fatalf("attempts = %d, want 3", attempts)
		}
	})

	t.Run("returns last error", func(t *testing.T) {
		lastErr := errors.New("connection refused")
		attempts := 0
		err := checkDependencyWithRetryConfig("Redis", 2, time.Millisecond, func() error {
			attempts++
			return lastErr
		})
		if !errors.Is(err, lastErr) {
			t.Fatalf("error = %v, want wrapped last error", err)
		}
		if !strings.Contains(err.Error(), "Redis unavailable after 2 attempts") {
			t.Fatalf("error lacks dependency context: %v", err)
		}
		if attempts != 2 {
			t.Fatalf("attempts = %d, want 2", attempts)
		}
	})

	t.Run("rejects invalid attempts", func(t *testing.T) {
		err := checkDependencyWithRetryConfig("Redis", 0, time.Millisecond, func() error { return nil })
		if err == nil {
			t.Fatal("expected validation error")
		}
	})
}
