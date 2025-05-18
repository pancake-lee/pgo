package db

import (
	"database/sql"
	"fmt"
	"pgo/pkg/db/dao/query"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var gDB *gorm.DB

func initPG() {
	host := "192.168.101.8"
	// host := "192.168.3.111"
	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v",
		host, "gogogo", "gogogo", "gogogo", 5432)
	dsn += " sslmode=disable TimeZone=Asia/Shanghai"

	_db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		panic(err)
	}
	if _db == nil {
		panic("db is nil")
	}
	gDB = _db
}
func GetDB() (*sql.DB, error) {
	if gDB == nil {
		initPG()
	}
	return gDB.DB()
}
func GetPG() *query.Query {
	if gDB == nil {
		initPG()
	}
	return query.Use(gDB)
}
func GetTables() (ret []string, err error) {
	db, _ := GetDB()
	rows, err := db.Query(`show tables`)
	if err != nil {
		return ret, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return ret, err
		}
		ret = append(ret, tableName)
	}
	if err := rows.Err(); err != nil {
		return ret, err
	}
	return ret, err
}
