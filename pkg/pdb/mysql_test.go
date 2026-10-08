package pdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	glebarez "github.com/glebarez/sqlite"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"gorm.io/gorm"
)

// TestMysqlPoolConfig 验证真实配置读取、默认值与非法输入校验。
func TestMysqlPoolConfig(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fields  string
		open    int
		idle    int
		invalid bool
	}{
		{name: "omitted", open: 64, idle: 8},
		{name: "custom", fields: "  MaxOpenConns: 32\n  MaxIdleConns: 4\n", open: 32, idle: 4},
		{name: "zero idle", fields: "  MaxIdleConns: 0\n", open: 64, idle: 0},
		{name: "only open", fields: "  MaxOpenConns: 4\n", open: 4, idle: 4},
		{name: "only idle", fields: "  MaxIdleConns: 16\n", open: 64, idle: 16},
		{name: "zero open", fields: "  MaxOpenConns: 0\n", invalid: true},
		{name: "negative open", fields: "  MaxOpenConns: -1\n", invalid: true},
		{name: "negative idle", fields: "  MaxIdleConns: -1\n", invalid: true},
		{name: "idle above open", fields: "  MaxOpenConns: 4\n  MaxIdleConns: 8\n", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "common.yaml")
			err := os.WriteFile(path, []byte("Mysql:\n  Addr: localhost:3306\n"+tc.fields), 0600)
			if err != nil {
				t.Fatal(err)
			}
			err = pconfig.InitConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = pconfig.MustGetConfig().Close() })
			var conf MysqlConfig
			err = pconfig.Scan(&conf)
			if err != nil {
				t.Fatal(err)
			}
			pool, err := resolveMysqlPoolConfig(conf.Mysql)
			if tc.invalid {
				if err == nil {
					t.Fatal("invalid pool accepted")
				}
				// 应在网络连接前拒绝非法配置。
				err = InitMysqlWithConfig(conf.Mysql)
				if err == nil {
					t.Fatal("initialization accepted invalid pool")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if *pool.MaxOpenConns != tc.open || *pool.MaxIdleConns != tc.idle {
				t.Fatalf("pool = %d/%d, want %d/%d", *pool.MaxOpenConns,
					*pool.MaxIdleConns, tc.open, tc.idle)
			}
		})
	}
}

// TestMysqlPoolLimits 验证 SQL 池达到上限时等待、归还后复用及空闲保留数量。
func TestMysqlPoolLimits(t *testing.T) {
	for _, idle := range []int{8, 0} {
		t.Run(fmt.Sprintf("idle=%d", idle), func(t *testing.T) {
			db, err := gorm.Open(glebarez.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			defer sqlDB.Close()
			conf, err := resolveMysqlPoolConfig(SqlConfig{MaxIdleConns: &idle})
			if err != nil {
				t.Fatal(err)
			}
			setMysqlPool(sqlDB, conf)
			if sqlDB.Stats().MaxOpenConnections != 64 {
				t.Fatal("maximum not applied")
			}
			var connList []*sql.Conn
			defer func() {
				for _, conn := range connList {
					_ = conn.Close()
				}
			}()
			for range 64 {
				conn, err := sqlDB.Conn(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				connList = append(connList, conn)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
			defer cancel()
			extra, err := sqlDB.Conn(ctx)
			if extra != nil {
				_ = extra.Close()
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("limit wait = %v", err)
			}
			if sqlDB.Stats().WaitCount == 0 {
				t.Fatal("waiting not recorded")
			}
			err = connList[0].Close()
			if err != nil {
				t.Fatal(err)
			}
			reused, err := sqlDB.Conn(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			_ = reused.Close()
			for _, conn := range connList {
				_ = conn.Close()
			}
			stats := sqlDB.Stats()
			if stats.Idle != idle || stats.OpenConnections != idle {
				t.Fatalf("idle pool = %+v, want %d", stats, idle)
			}
		})
	}
}
