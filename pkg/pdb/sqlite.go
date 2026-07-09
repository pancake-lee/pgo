package pdb

import (
	"path/filepath"
	"time"

	glebarez "github.com/glebarez/sqlite"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"gorm.io/gorm"
	dbLogger "gorm.io/gorm/logger"
)

const DefaultSqliteConfigGroup = "Sqlite"

type SqliteConfig struct {
	Path string
}

func MustInitSqliteByConfig() {
	err := InitSqliteByConfig()
	if err != nil {
		panic(err)
	}
}

func InitSqliteByConfig() error {
	var conf SqliteConfig
	err := pconfig.Scan(&conf)
	if err != nil {
		return err
	}
	plogger.Infof("load sqlite config: %+v", conf)
	return InitSqlite(conf.Path)
}

// InitSqlite 使用 github.com/glebarez/sqlite 驱动（纯 Go）初始化 SQLite
// 数据库连接，并默认开启 WAL 模式。适用于交叉编译或无法启用 CGO 的环境。
// PS: gorm.io/driver/sqlite 驱动需要 CGO
func InitSqlite(dbPath string) (err error) {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return err
	}

	gDriverType = DriverSQLite
	gSavedFilePath = absPath

	gConf = &SqlConfig{
		Addr:   absPath,
		DbName: filepath.Base(absPath),
	}

	gDB, err = gorm.Open(glebarez.Open(absPath), &gorm.Config{
		Logger: dbLogger.New(
			Writer{},
			dbLogger.Config{
				SlowThreshold:             200 * time.Millisecond, // Slow SQL threshold
				LogLevel:                  dbLogger.Warn,          // Log level
				IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
				Colorful:                  false,                  // Disable color
			},
		),
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
