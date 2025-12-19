package data

import (
	"context"
	"errors"

	"github.com/pancake-lee/pgo/internal/pkg/db"
)

func (*userJobDAO) EditJobName(ctx context.Context,
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
