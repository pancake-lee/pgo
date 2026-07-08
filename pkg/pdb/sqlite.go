package pdb

import (
	"path/filepath"

	glebarez "github.com/glebarez/sqlite"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// InitSqlite 使用 gorm.io/driver/sqlite 驱动（需要 CGO）初始化 SQLite 数据库连接，
// 并默认开启 WAL 模式以提升并发读写性能。
func InitSqlite(dbPath string) (err error) {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return err
	}

	gConf = &SqlConfig{
		Addr:   absPath,
		DbName: filepath.Base(absPath),
	}

	gDB, err = gorm.Open(sqlite.Open(absPath), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return err
	}

	if err = gDB.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		plogger.Warnf("failed to enable WAL mode: %v", err)
	}

	plogger.Infof("sqlite connected: %s", absPath)
	return nil
}

// InitSqlitePureGo 使用 github.com/glebarez/sqlite 驱动（纯 Go，无需 CGO）初始化 SQLite
// 数据库连接，并默认开启 WAL 模式。适用于交叉编译或无法启用 CGO 的环境。
func InitSqlitePureGo(dbPath string) (err error) {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return err
	}

	gConf = &SqlConfig{
		Addr:   absPath,
		DbName: filepath.Base(absPath),
	}

	gDB, err = gorm.Open(glebarez.Open(absPath), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return err
	}

	if err = gDB.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		plogger.Warnf("failed to enable WAL mode: %v", err)
	}

	plogger.Infof("sqlite (pure go) connected: %s", absPath)
	return nil
}
