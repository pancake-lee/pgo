package data

import (
	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/pkg/papp"
)

func (*userDeptAssocDAO) DelByPrimaryKey(ctx *papp.AppCtx,
	userID, deptID int32) error {

	q := db.GetQuery().UserDeptAssoc
	_, err := q.WithContext(ctx).
		Where(q.UserID.Eq(userID), q.DeptID.Eq(deptID)).
		Delete()
	if err != nil {
		return err
	}
	return err
}
