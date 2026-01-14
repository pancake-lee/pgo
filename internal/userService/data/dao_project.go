package data

import (
	"errors"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/pkg/papp"
)

func (*projectDAO) UpdateMtblInfo(ctx *papp.AppCtx, id int32,
	lastEditFrom string) error {
	if id == 0 {
		return errors.New("argument invalid")
	}
	q := db.GetQuery().Project
	_, err := q.WithContext(ctx).
		Where(q.ID.Eq(id)).
		UpdateSimple(
			q.LastEditFrom.Value(lastEditFrom),
		)
	return err
}
