package pdb

import (
	"path/filepath"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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

	plogger.Infof("sqlite connected: %s", absPath)
	return nil
}
