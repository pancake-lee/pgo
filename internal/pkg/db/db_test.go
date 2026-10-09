package db

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/internal/pkg/db/query"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"gorm.io/gorm"
)

// initQueryTestDB 绑定临时 SQLite，使用 DryRun 验证查询而不修改表结构。
func initQueryTestDB(t testing.TB) *gorm.DB {
	t.Helper()
	err := pdb.InitSqlite(filepath.Join(t.TempDir(), "query.db"))
	if err != nil {
		t.Fatal(err)
	}
	database := pdb.GetGormDB()
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	database.Config.DryRun = true
	InitQuery()
	return database
}

// TestGetQueryReusesAndRebindsDefault 验证复用、零分配及数据库替换后重新绑定。
func TestGetQueryReusesAndRebindsDefault(t *testing.T) {
	firstDB := initQueryTestDB(t)
	firstQuery := GetQuery()
	if firstQuery != query.Q {
		t.Fatal("GetQuery did not return the generated default Query")
	}
	allocations := testing.AllocsPerRun(100, func() {
		if GetQuery() != firstQuery {
			t.Fatal("GetQuery rebuilt the Query")
		}
	})
	if allocations != 0 {
		t.Fatalf("GetQuery allocations = %v, want 0", allocations)
	}

	secondDB := initQueryTestDB(t)
	boundDB := GetQuery().UserRole.WithContext(context.Background()).UnderlyingDB()
	if boundDB.Statement.ConnPool != secondDB.Statement.ConnPool {
		t.Fatal("default Query did not bind to the replacement database")
	}
	if boundDB.Statement.ConnPool == firstDB.Statement.ConnPool {
		t.Fatal("default Query still uses the previous database")
	}
	if query.UserRole != &query.Q.UserRole {
		t.Fatal("generated table alias did not bind to the default Query")
	}
}

// TestDefaultQueryIsolatesRequests 验证并发请求的 context、条件和取消不互相污染。
func TestDefaultQueryIsolatesRequests(t *testing.T) {
	initQueryTestDB(t)
	q := GetQuery()
	type contextKey struct{}
	var workers sync.WaitGroup
	for i := int32(1); i <= 32; i++ {
		workers.Add(1)
		go func(id int32) {
			defer workers.Done()
			ctx := context.WithValue(context.Background(), contextKey{}, id)
			requestDB := q.UserRole.WithContext(ctx).
				Where(q.UserRole.ID.Eq(id)).UnderlyingDB()
			var roleList []model.UserRole
			result := requestDB.Find(&roleList)
			if result.Error != nil {
				t.Error(result.Error)
				return
			}
			statement := result.Statement
			if len(statement.Vars) != 1 || statement.Vars[0] != id {
				t.Errorf("request %d bindings = %v", id, statement.Vars)
			}
			if statement.Context.Value(contextKey{}) != id {
				t.Errorf("request %d context was overwritten", id)
			}
		}(i)
	}
	workers.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	canceledDB := q.UserRole.WithContext(ctx).UnderlyingDB()
	cancel()
	if canceledDB.Statement.Context.Err() != context.Canceled {
		t.Fatal("request cancellation did not reach its query")
	}
	cleanDB := q.UserRole.WithContext(context.Background()).UnderlyingDB()
	var roleList []model.UserRole
	result := cleanDB.Find(&roleList)
	if result.Error != nil || len(result.Statement.Vars) != 0 {
		t.Fatalf("base Query retained request state: %v, %v",
			result.Error, result.Statement.Vars)
	}
}

// TestGetQueryTxKeepsTransactionBinding 验证事务查询使用事务连接，结束后恢复默认库。
func TestGetQueryTxKeepsTransactionBinding(t *testing.T) {
	database := initQueryTestDB(t)
	const transactionID int32 = 7401
	tx, err := pdb.Begin(transactionID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	transactionDB := pdb.GetGormDB(transactionID)
	boundDB := GetQueryTx(transactionID).UserRole.
		WithContext(context.Background()).UnderlyingDB()
	if boundDB.Statement.ConnPool != transactionDB.Statement.ConnPool {
		t.Fatal("transaction Query did not bind to its transaction connection")
	}
	if boundDB.Statement.ConnPool == database.Statement.ConnPool {
		t.Fatal("transaction Query used the default pool")
	}
	defaultDB := GetQuery().UserRole.WithContext(context.Background()).UnderlyingDB()
	if defaultDB.Statement.ConnPool != database.Statement.ConnPool {
		t.Fatal("transaction changed the default Query")
	}
	err = tx.Rollback()
	if err != nil {
		t.Fatal(err)
	}
	boundDB = GetQueryTx(transactionID).UserRole.
		WithContext(context.Background()).UnderlyingDB()
	if boundDB.Statement.ConnPool != database.Statement.ConnPool {
		t.Fatal("completed transaction did not return to the default database")
	}
}

// TestTransactionQueriesIsolateRequests 验证不同事务的真实查询链与请求参数隔离。
func TestTransactionQueriesIsolateRequests(t *testing.T) {
	database := initQueryTestDB(t)
	type requestKey struct{}
	type requestInfo struct {
		id            int32
		transactionID int32
	}
	transactionDBMap := make(map[int32]*gorm.DB)
	for _, id := range []int32{7403, 7404} {
		tx, err := pdb.Begin(id)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = tx.Rollback() })
		transactionDBMap[id] = pdb.GetGormDB(id)
	}
	if transactionDBMap[7403].Statement.ConnPool ==
		transactionDBMap[7404].Statement.ConnPool {
		t.Fatal("distinct transactions share a transaction connection")
	}

	err := database.Callback().Query().After("*").Register(
		"test:transaction_query",
		func(result *gorm.DB) {
			statement := result.Statement
			info := statement.Context.Value(requestKey{}).(requestInfo)
			if statement.ConnPool !=
				transactionDBMap[info.transactionID].Statement.ConnPool {
				t.Errorf("request %d used another transaction", info.id)
			}
			if len(statement.Vars) != 1 || statement.Vars[0] != info.id {
				t.Errorf("request %d bindings = %v", info.id, statement.Vars)
			}
			if !strings.Contains(statement.SQL.String(), "FROM `user_role`") {
				t.Errorf("transaction Query lost model binding: %s",
					statement.SQL.String())
			}
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for i := int32(1); i <= 32; i++ {
		workers.Add(1)
		go func(id int32) {
			defer workers.Done()
			info := requestInfo{id: id, transactionID: 7403 + id%2}
			ctx := context.WithValue(context.Background(), requestKey{}, info)
			q := GetQueryTx(info.transactionID)
			_, err := q.UserRole.WithContext(ctx).
				Where(q.UserRole.ID.Eq(id)).Find()
			if err != nil {
				t.Error(err)
			}
		}(i)
	}
	workers.Wait()
}

var benchmarkQuery *query.Query

// BenchmarkQueryConstruction 对比默认查询复用与原先每次构造全部表对象的成本。
func BenchmarkQueryConstruction(b *testing.B) {
	database := initQueryTestDB(b)
	b.Run("Reused", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkQuery = GetQuery()
		}
	})
	b.Run("Rebuilt", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkQuery = query.Use(database)
		}
	})
}

// BenchmarkTransactionQueryConstruction 对比事务重新绑定与从头构造查询的成本。
func BenchmarkTransactionQueryConstruction(b *testing.B) {
	initQueryTestDB(b)
	const transactionID int32 = 7402
	tx, err := pdb.Begin(transactionID)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = tx.Rollback() })
	transactionDB := pdb.GetGormDB(transactionID)
	b.Run("Rebound", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkQuery = GetQueryTx(transactionID)
		}
	})
	b.Run("Rebuilt", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkQuery = query.Use(transactionDB)
		}
	})
}
