package main

import (
	"context"
	"errors"
	"fmt"
	"gogogo/pkg/db/dao/model"
	"gogogo/pkg/db/dao/query"
	"gogogo/pkg/db/pgerrcode"

	"github.com/jackc/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// gentool -db postgres -dsn "host=192.168.3.18 user=gogogo password=gogogo dbname=gogogo port=5432 sslmode=disable TimeZone=Asia/Shanghai" -tables user
func pgsql() {

	dsn := "host=192.168.3.18 user=gogogo password=gogogo dbname=gogogo port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		panic(err)
	}
	{
		var u model.User
		u.UserName = "pancake"
		// 用gen生成的model以及接口
		err = query.Use(db).User.WithContext(context.Background()).Create(&u)
		// 用gen生成的model，但是用gorm的接口
		// err := db.Create(&u).Error
		if err != nil {
			// https://gorm.io/docs/error_handling.html#Dialect-Translated-Errors
			// https://github.com/go-gorm/gorm/issues/4037
			// 根据这个issues的回复，最新的处理方式是TranslateError设置为true，然后如下判断，但是我这里并没有生效
			// if errors.Is(err, gorm.ErrDuplicatedKey) {
			// 	fmt.Println("duplicated key")
			// }
			// issues里的另一个方案是可行的，先这样吧
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgErr.Code != pgerrcode.UniqueViolation {
					panic(err)
				}
			} else {
				panic(err)
			}
		}
	}
	{
		q := query.Use(db).User
		u, err := q.WithContext(context.Background()).Where(q.UserName.Eq("pancake")).First()
		if err != nil {
			panic(err)
		}
		fmt.Println("id : ", u.ID)
		fmt.Println("name : ", u.UserName)
	}
	{
		q := query.Use(db).User
		result, err := q.WithContext(context.Background()).Where(q.UserName.Eq("pancake")).Delete()
		if err != nil {
			panic(err)
		}
		fmt.Println("del cnt : ", result.RowsAffected)
	}
}
