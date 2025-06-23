package data

import (
	"context"
	"errors"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
)

func (*userDAO) GetOrAdd(ctx context.Context,
	user *UserDO) (*model.User, error) {
	if user == nil || user.UserName == "" {
		return nil, errors.New("user name is empty")
	}
	q := db.GetPG().User
	user, err := q.WithContext(ctx).
		Where(q.UserName.Eq(user.UserName)).
		Attrs(q.UserName.Value(user.UserName)).
		FirstOrCreate()
	if err != nil {
		return nil, err
	}
	return user, err
}

func (*userDAO) EditUserName(ctx context.Context, id int32, userName string) error {
	if id == 0 || userName == "" {
		return errors.New("argument invalid")
	}
	q := db.GetPG().User
	_, err := q.WithContext(ctx).
		Where(q.ID.Eq(id)).
		UpdateSimple(q.UserName.Value(userName))
	return err
}
