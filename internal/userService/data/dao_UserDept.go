package data

import (
	"context"
	"errors"

	"github.com/pancake-lee/pgo/internal/pkg/db"
)

func (*userDeptDAO) EditDeptName(ctx context.Context,
	id int32, deptName string) error {
	if id == 0 || deptName == "" {
		return errors.New("param is invalid")
	}
	q := db.GetPG().UserDept
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).
		Update(q.DeptName, deptName)
	if err != nil {
		return err

	}
	return err
}

func (*userDeptDAO) GetWithDeptPath(ctx context.Context, deptPath string,
) (userDept *UserDeptDO, err error) {
	if deptPath == "" {
		return userDept, errors.New("param is invalid")
	}

	q := db.GetPG().UserDept
	userDept, err = q.WithContext(ctx).
		Where(q.DeptPath.Eq(deptPath)).First()
	if err != nil {
		return nil, err
	}
	return userDept, nil
}

func (*userDeptDAO) GetWithDeptPaths(ctx context.Context, deptPaths []string,
) (userDeptList []*UserDeptDO, err error) {
	if len(deptPaths) == 0 {
		return userDeptList, nil
	}

	q := db.GetPG().UserDept
	userDeptList, err = q.WithContext(ctx).
		Where(q.DeptPath.In(deptPaths...)).Find()
	if err != nil {
		return nil, err
	}
	return userDeptList, nil
}
