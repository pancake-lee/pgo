package data

import (
	"context"
	"pgo/pkg/db"
)

func (*userDeptAssocDAO) DelByPrimaryKey(ctx context.Context,
	userID, deptID int32) error {

	q := db.GetPG().UserDeptAssoc
	_, err := q.WithContext(ctx).
		Where(q.UserID.Eq(userID), q.DeptID.Eq(deptID)).
		Delete()
	if err != nil {
		return err
	}
	return err
}
