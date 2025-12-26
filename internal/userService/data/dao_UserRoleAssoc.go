package data

import (
	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/pkg/papp"
)

func (*userRoleAssocDAO) GetByUserID(ctx *papp.AppCtx, userID int32) ([]*model.UserRoleAssoc, error) {
	q := db.GetQuery()
	return q.UserRoleAssoc.WithContext(ctx).Where(q.UserRoleAssoc.UserID.Eq(userID)).Find()
}
