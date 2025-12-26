package data

import (
	"errors"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/pkg/papp"
)

func (*userJobDAO) EditJobName(ctx *papp.AppCtx,
	id int32, jobName string) error {
	if id == 0 || jobName == "" {
		return errors.New("param is invalid")
	}
	q := db.GetQuery().UserJob
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).
		Update(q.JobName, jobName)
	if err != nil {
		return err
	}
	return err
}
