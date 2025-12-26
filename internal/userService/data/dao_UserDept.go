package data

import (
	"errors"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/pkg/papp"
)

func (*userDeptDAO) EditDeptName(ctx *papp.AppCtx,
	id int32, deptName string) error {
	if id == 0 || deptName == "" {
		return errors.New("param is invalid")
	}
	q := db.GetQuery().UserDept
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).
		Update(q.DeptName, deptName)
	if err != nil {
		return err

	}
	return err
}

func (*userDeptDAO) GetWithDeptPath(ctx *papp.AppCtx, deptPath string,
) (userDept *UserDeptDO, err error) {
	if deptPath == "" {
		return userDept, errors.New("param is invalid")
	}

	q := db.GetQuery().UserDept
	userDept, err = q.WithContext(ctx).
		Where(q.DeptPath.Eq(deptPath)).First()
	if err != nil {
		return nil, err
	}
	return userDept, nil
}

func (*userDeptDAO) GetWithDeptPaths(ctx *papp.AppCtx, deptPaths []string,
) (userDeptList []*UserDeptDO, err error) {
	if len(deptPaths) == 0 {
		return userDeptList, nil
	}

	q := db.GetQuery().UserDept
	userDeptList, err = q.WithContext(ctx).
		Where(q.DeptPath.In(deptPaths...)).Find()
	if err != nil {
		return nil, err
	}
	return userDeptList, nil
}
