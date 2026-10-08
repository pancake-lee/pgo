package pdb

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	glebarez "github.com/glebarez/sqlite"
	kLog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/internal/pkg/db/query"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
	dbLogger "gorm.io/gorm/logger"
)

func captureGormLogs(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	original := plogger.GetDefaultLoggerNoCaller()
	core, logs := observer.New(zap.DebugLevel)
	plogger.SetDefaultLogger(kLog.With(plogger.FromZap(zap.New(core)),
		"tid", kLog.Valuer(func(ctx context.Context) any {
			tid, _ := putil.GetTraceIdFromCtx(ctx)
			return putil.StrPrefixByNum(tid, 6)
		}),
	))
	t.Cleanup(func() { plogger.SetDefaultLogger(original) })
	return logs
}

func TestGormSlowQueryRequestContext(t *testing.T) {
	logs := captureGormLogs(t)
	logger := newGormLogger(dbLogger.Config{
		LogLevel:      dbLogger.Warn,
		SlowThreshold: time.Nanosecond,
	})
	db, err := gorm.Open(
		glebarez.Open(filepath.Join(t.TempDir(), "trace.db")),
		&gorm.Config{Logger: logger},
	)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	logs.TakeAll()

	const workerCount = 24
	var workers sync.WaitGroup
	for i := range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			id := fmt.Sprintf("r%05d", i)
			ctx := putil.SetTraceIdToCtx(context.Background(), id)
			query := fmt.Sprintf("SELECT %d", i)
			var result int
			queryErr := db.WithContext(ctx).Raw(query).Scan(&result).Error
			if queryErr != nil {
				t.Errorf("query %s: %v", id, queryErr)
			}
		}()
	}
	workers.Wait()
	entries := logs.TakeAll()
	if len(entries) != workerCount {
		t.Fatalf("logs = %d, want %d", len(entries), workerCount)
	}
	seen := make(map[string]bool)
	for _, entry := range entries {
		id, ok := entry.ContextMap()["tid"].(string)
		if !ok || id == "" || seen[id] {
			t.Fatalf("missing or duplicate tid: %v", entry.ContextMap())
		}
		seen[id] = true
		var i int
		_, err := fmt.Sscanf(id, "r%05d", &i)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(entry.Message, fmt.Sprintf("SELECT %d", i)) {
			t.Errorf("tid %s attached to wrong query: %s", id, entry.Message)
		}
		if !strings.HasPrefix(entry.Message, "SLOW SQL >=") {
			t.Errorf("missing slow query format: %s", entry.Message)
		}
	}

	var result int
	err = db.WithContext(context.Background()).Raw("SELECT 99").Scan(&result).Error
	if err != nil {
		t.Fatal(err)
	}
	entries = logs.TakeAll()
	if len(entries) != 1 || entries[0].ContextMap()["tid"] != "" {
		t.Fatalf("query without request inherited a tid: %v", entries)
	}
}

func TestGormCallerQueryPaths(t *testing.T) {
	logs := captureGormLogs(t)
	logger := newGormLogger(dbLogger.Config{LogLevel: dbLogger.Info})
	db, err := gorm.Open(
		glebarez.Open(filepath.Join(t.TempDir(), "caller.db")),
		&gorm.Config{Logger: logger},
	)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	ctx := putil.SetTraceIdToCtx(context.Background(), "dao123")
	db = db.WithContext(ctx)
	testList := []struct {
		name string
		run  func(*gorm.DB) (string, error)
	}{
		{"raw", func(db *gorm.DB) (string, error) {
			var result int
			_, file, line, _ := runtime.Caller(0)
			err := db.Raw("SELECT 1").Scan(&result).Error
			return fmt.Sprintf("%s:%d", file, line+1), err
		}},
		{"find", func(db *gorm.DB) (string, error) {
			var records []struct{ ID int }
			_, file, line, _ := runtime.Caller(0)
			err := db.Table("(SELECT 1 AS id) AS record").Find(&records).Error
			return fmt.Sprintf("%s:%d", file, line+1), err
		}},
		{"exec", func(db *gorm.DB) (string, error) {
			_, file, line, _ := runtime.Caller(0)
			err := db.Exec("SELECT 1").Error
			return fmt.Sprintf("%s:%d", file, line+1), err
		}},
		{"transaction", func(db *gorm.DB) (string, error) {
			var caller string
			err := db.Transaction(func(tx *gorm.DB) error {
				var result int
				_, file, line, _ := runtime.Caller(0)
				queryErr := tx.Raw("SELECT 1").Scan(&result).Error
				caller = fmt.Sprintf("%s:%d", file, line+1)
				return queryErr
			})
			return caller, err
		}},
		{"generated", func(db *gorm.DB) (string, error) {
			// 使用 SQLite 系统表执行生成查询，不创建业务表结构。
			p := query.Use(db).UserRolePermissionAssoc.Table("sqlite_master")
			dao := p.WithContext(ctx)
			_, file, line, _ := runtime.Caller(0)
			_, err := dao.Find()
			return fmt.Sprintf("%s:%d", file, line+1), err
		}},
	}
	for _, test := range testList {
		t.Run(test.name, func(t *testing.T) {
			logs.TakeAll()
			caller, err := test.run(db)
			if err != nil {
				t.Fatal(err)
			}
			entries := logs.TakeAll()
			if len(entries) != 1 {
				t.Fatalf("logs = %d, want 1", len(entries))
			}
			entry := entries[0]
			if entry.ContextMap()["caller"] != caller {
				t.Errorf("caller = %v, want %s", entry.ContextMap()["caller"], caller)
			}
			if entry.ContextMap()["tid"] != "dao123" {
				t.Errorf("lost tid: %v", entry.ContextMap())
			}
			if !strings.HasPrefix(entry.Message, "[") {
				t.Errorf("unexpected position prefix: %s", entry.Message)
			}
		})
	}
}

func TestGormMessageCaller(t *testing.T) {
	logs := captureGormLogs(t)
	ctx := putil.SetTraceIdToCtx(context.Background(), "req123")
	logger := newGormLogger(dbLogger.Config{LogLevel: dbLogger.Info})
	for _, log := range []func(context.Context, string, ...any){
		logger.Info, logger.Warn, logger.Error,
	} {
		_, file, line, _ := runtime.Caller(0)
		log(ctx, "message %d", 42)
		entries := logs.TakeAll()
		if len(entries) != 1 {
			t.Fatalf("logs = %d, want 1", len(entries))
		}
		entry := entries[0]
		caller := fmt.Sprintf("%s:%d", file, line+1)
		if entry.ContextMap()["caller"] != caller {
			t.Errorf("caller = %v, want %s", entry.ContextMap()["caller"], caller)
		}
		if entry.Message != "message 42" || entry.ContextMap()["tid"] != "req123" {
			t.Errorf("unexpected log: %s %v", entry.Message, entry.ContextMap())
		}
	}
}

func TestGormTraceFilteringAndFormat(t *testing.T) {
	logs := captureGormLogs(t)
	ctx := putil.SetTraceIdToCtx(context.Background(), "req123")
	testList := []struct {
		name   string
		level  dbLogger.LogLevel
		slow   time.Duration
		ignore bool
		err    error
		want   string
	}{
		{"silent", dbLogger.Silent, time.Second, false, errors.New("fail"), ""},
		{"error threshold", dbLogger.Error, time.Second, false, nil, ""},
		{"warn threshold", dbLogger.Warn, time.Second, false, nil, "SLOW SQL"},
		{"slow disabled", dbLogger.Warn, 0, false, nil, ""},
		{"normal info", dbLogger.Info, 0, false, nil, "["},
		{"ignored not found", dbLogger.Warn, 0, true, gorm.ErrRecordNotFound, ""},
		{"not found", dbLogger.Warn, 0, false, gorm.ErrRecordNotFound, "record not found"},
		{"wrapped not found", dbLogger.Warn, 0, true,
			fmt.Errorf("wrapped: %w", gorm.ErrRecordNotFound), ""},
		{"ignored slow not found", dbLogger.Warn, time.Second, true,
			gorm.ErrRecordNotFound, "SLOW SQL"},
		{"error before slow", dbLogger.Info, time.Second, false,
			errors.New("query failed"), "query failed"},
	}
	for _, test := range testList {
		t.Run(test.name, func(t *testing.T) {
			logger := newGormLogger(dbLogger.Config{
				LogLevel:                  test.level,
				SlowThreshold:             test.slow,
				IgnoreRecordNotFoundError: test.ignore,
			})
			calls := 0
			_, file, line, _ := runtime.Caller(0)
			logger.Trace(ctx, time.Now().Add(-3*time.Second), func() (string, int64) {
				calls++
				return "SELECT '/keep/file.go:12'", -1
			}, test.err)
			entries := logs.TakeAll()
			if test.want == "" {
				if len(entries) != 0 || calls != 0 {
					t.Fatal("suppressed query logged or evaluated SQL")
				}
				return
			}
			if len(entries) != 1 || calls != 1 {
				t.Fatal("SQL should be evaluated and logged once")
			}
			entry := entries[0]
			if !strings.HasPrefix(entry.Message, test.want) ||
				!strings.HasSuffix(entry.Message, "[rows:-] SELECT '/keep/file.go:12'") {
				t.Errorf("unexpected SQL format: %s", entry.Message)
			}
			caller := fmt.Sprintf("%s:%d", file, line+1)
			if entry.ContextMap()["caller"] != caller {
				t.Errorf("caller = %v, want %s", entry.ContextMap()["caller"], caller)
			}
			if entry.Level != zap.InfoLevel {
				t.Errorf("changed existing output level: %s", entry.Level)
			}
		})
	}
}

func TestGormLoggerModesAndFiltering(t *testing.T) {
	logs := captureGormLogs(t)
	ctx := putil.SetTraceIdToCtx(context.Background(), "req123")
	logger := newGormLogger(dbLogger.Config{
		LogLevel:                  dbLogger.Warn,
		SlowThreshold:             time.Second,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
	})
	query := func() (string, int64) { return "SELECT ?", 1 }
	logger.Trace(ctx, time.Now(), query, nil)
	logger.Trace(ctx, time.Now(), query, gorm.ErrRecordNotFound)
	logger.LogMode(dbLogger.Silent).Trace(ctx, time.Now(), query, errors.New("fail"))
	if logs.Len() != 0 {
		t.Fatalf("filtered queries emitted %d logs", logs.Len())
	}
	logger.LogMode(dbLogger.Info).Trace(ctx, time.Now(), query, nil)
	logger.Trace(ctx, time.Now(), query, nil)
	if logs.Len() != 1 {
		t.Fatal("LogMode mutated the shared logger or failed to enable SQL logs")
	}
	logger.Trace(ctx, time.Now(), query, errors.New("query failed"))
	logger.Info(ctx, "suppressed info")
	logger.Warn(ctx, "warning")
	logger.Error(ctx, "error")
	entries := logs.TakeAll()
	if len(entries) != 4 {
		t.Fatalf("logs = %d, want 4", len(entries))
	}
	for _, entry := range entries {
		if entry.ContextMap()["tid"] != "req123" {
			t.Errorf("lost request context: %v", entry.ContextMap())
		}
	}
	if !strings.Contains(entries[1].Message, "query failed") {
		t.Errorf("lost query error: %s", entries[1].Message)
	}
	filter := logger.(gorm.ParamsFilter)
	sql, params := filter.ParamsFilter(ctx, "SELECT ?", "secret")
	if sql != "SELECT ?" || params != nil {
		t.Fatal("parameterized logs exposed query arguments")
	}
	filter = newGormLogger(dbLogger.Config{}).(gorm.ParamsFilter)
	_, params = filter.ParamsFilter(ctx, "SELECT ?", "value")
	if len(params) != 1 || params[0] != "value" {
		t.Fatal("non-parameterized logs lost query arguments")
	}
}
