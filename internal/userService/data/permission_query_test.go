package data

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/pkg/papp"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"gorm.io/gorm"
)

// TestPermissionQueriesUseGeneratedFields 验证真实 DAO 的单条 JOIN 和批量查询 SQL。
func TestPermissionQueriesUseGeneratedFields(t *testing.T) {
	err := pdb.InitSqlite(filepath.Join(t.TempDir(), "queries.db"))
	if err != nil {
		t.Fatal(err)
	}
	database := pdb.GetGormDB()
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	// DryRun 不创建表、不执行 SQL，仅检验生成接口产生的真实语句。
	database.Config.DryRun = true
	db.InitQuery()
	var sqlList []string
	var varList [][]any
	err = database.Callback().Query().After("*").Register("test:capture", func(statement *gorm.DB) {
		sqlList = append(sqlList, statement.Statement.SQL.String())
		varList = append(varList, append([]any(nil), statement.Statement.Vars...))
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := papp.NewAppCtx(context.Background())
	_, err = UserRolePermissionAssocDAO.GetByUserAndProject(ctx, 42, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(sqlList) != 1 {
		t.Fatalf("JOIN issued %d queries", len(sqlList))
	}
	query := sqlList[0]
	for _, fragment := range []string{
		"INNER JOIN `user_role`", "INNER JOIN `user_role_assoc`",
		"`user_role`.`id` = `user_role_permission_assoc`.`role_id`",
		"`user_role_assoc`.`role_id` = `user_role`.`id`",
		"`user_role_assoc`.`user_id` = ?", "`user_role`.`proj_id` = ?",
		"ORDER BY `user_role_permission_assoc`.`id`",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("missing %s in %s", fragment, query)
		}
	}
	if len(varList[0]) != 2 || varList[0][0] != int32(42) || varList[0][1] != int32(7) {
		t.Fatalf("unexpected bindings: %v", varList)
	}
	_, err = UserRolePermissionAssocDAO.GetByRoleIDs(ctx, []int32{11, 12})
	if err != nil {
		t.Fatal(err)
	}
	if len(sqlList) != 2 || !strings.Contains(sqlList[1], "`role_id` IN (?,?)") {
		t.Fatalf("unexpected batch query: %v", sqlList)
	}
	_, err = UserRolePermissionAssocDAO.GetByRoleIDs(ctx, nil)
	if err != nil || len(sqlList) != 2 {
		t.Fatal("empty role list executed a query")
	}
}
