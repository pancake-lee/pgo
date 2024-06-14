package main

import (
	"flag"
	"fmt"
	"gogogo/pkg/config"
	"gogogo/pkg/db"
	"log"
)

type Column struct {
	Name string
	Type string
}
type Table struct {
	ServiceName string
	Name        string
	Columns     []*Column
}

func main() {
	var confPath string
	flag.StringVar(&confPath, "conf", "configs/config.ini", "config path, eg: -conf config.yaml")
	flag.Parse()

	config.LoadConf(confPath)

	var err error

	tblToSvrMap := make(map[string]*Table)
	tblToSvrMap["user"] =
		&Table{ServiceName: "userService", Name: "user"}
	tblToSvrMap["user_dept"] =
		&Table{ServiceName: "userService", Name: "user_dept"}
	tblToSvrMap["user_dept_assoc"] =
		&Table{ServiceName: "userService", Name: "user_dept_assoc"}
	tblToSvrMap["user_job"] =
		&Table{ServiceName: "userService", Name: "user_job"}

	//1: 读取数据库表结构
	// tables, _ := db.GetTables()
	// for _, tableName := range tables {
	for _, table := range tblToSvrMap {
		table.Columns, err = getTableColumns(table.Name)
		if err != nil {
			log.Fatal("get tbl col err : ", err)
		}
		for _, col := range table.Columns {
			log.Printf("table[%v] col[%v] type[%v]\n", table.Name, col.Name, col.Type)
		}
	}

	//2: 读取模板文件

	//3: 生成代码
	//3.1: 调用 gorm-gen 生成 model query 代码
	//3.2: 生成 curd 的 dao 代码
	//3.3: 生成 curd 的 proto 定义，包括数据结构和接口定义
	//3.4: 调用 protoc 生成 go 代码
	//3.5: 生成 curd 的 service 代码，包括 DO 和 DTO 的转换代码，基础 curd 的实现
}

func getTableColumns(t string) ([]*Column, error) {
	db, _ := db.GetDB()
	rows, err := db.Query(fmt.Sprintf(
		`SELECT column_name, data_type FROM information_schema.columns 
		WHERE table_name = '%v'`, t))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []*Column
	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			return nil, err
		}
		var col Column
		col.Name = columnName
		col.Type = dataType
		cols = append(cols, &col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cols, nil
}
