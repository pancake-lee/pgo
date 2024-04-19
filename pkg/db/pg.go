package db

import (
	"gogogo/pkg/db/dao/query"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func initPG() {
	dsn := "host=192.168.3.18 user=gogogo password=gogogo dbname=gogogo port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	_db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		panic(err)
	}
	if _db == nil {
		panic("db is nil")
	}
	db = _db
}

func GetPG() *query.Query {
	if db == nil {
		initPG()
	}
	return query.Use(db)
}
