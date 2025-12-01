package data

import (
	"context"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
)

func (*userRoleAssocDAO) GetByUserID(ctx context.Context, userID int32) ([]*model.UserRoleAssoc, error) {
	q := db.GetPG()
	return q.UserRoleAssoc.WithContext(ctx).Where(q.UserRoleAssoc.UserID.Eq(userID)).Find()
}
