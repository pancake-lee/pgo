// Code generated by tools/genCURD. DO NOT EDIT.

package data

import (
	"context"
	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/internal/pkg/perr"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

type CourseSwapRequestDO = model.CourseSwapRequest

type courseSwapRequestDAO struct{}

var CourseSwapRequestDAO courseSwapRequestDAO

func (*courseSwapRequestDAO) Add(ctx context.Context, courseSwapRequest *CourseSwapRequestDO) error {
	if courseSwapRequest == nil {
		return plogger.LogErr(perr.ErrParamInvalid)
	}
	q := db.GetPG().CourseSwapRequest
	err := q.WithContext(ctx).Create(courseSwapRequest)
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

func (*courseSwapRequestDAO) GetAll(ctx context.Context,
) (courseSwapRequestList []*CourseSwapRequestDO, err error) {
	q := db.GetPG().CourseSwapRequest
	courseSwapRequestList, err = q.WithContext(ctx).Find()
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	return courseSwapRequestList, nil
}

func (*courseSwapRequestDAO) UpdateByID(ctx context.Context, do *CourseSwapRequestDO) error {
	if do.ID == 0 {
		return plogger.LogErr(perr.ErrParamInvalid)
	}
	q := db.GetPG().CourseSwapRequest
	_, err := q.WithContext(ctx).Where(q.ID.Eq(do.ID)).Updates(do)
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

func (*courseSwapRequestDAO) DelByID(ctx context.Context, iD int32) error {
	if iD == 0 {
		return plogger.LogErr(perr.ErrParamInvalid)
	}
	q := db.GetPG().CourseSwapRequest
	_, err := q.WithContext(ctx).Where(q.ID.Eq(iD)).Delete()
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

func (*courseSwapRequestDAO) DelByIDList(ctx context.Context, iDList []int32) error {
	if len(iDList) == 0 {
		return nil
	}
	q := db.GetPG().CourseSwapRequest
	_, err := q.WithContext(ctx).
		Where(q.ID.In(iDList...)).Delete()
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

func (*courseSwapRequestDAO) GetByID(ctx context.Context, iD int32,
) (courseSwapRequest *CourseSwapRequestDO, err error) {
	if iD == 0 {
		return courseSwapRequest, plogger.LogErr(perr.ErrParamInvalid)
	}

	q := db.GetPG().CourseSwapRequest
	courseSwapRequest, err = q.WithContext(ctx).
		Where(q.ID.Eq(iD)).First()
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	return courseSwapRequest, nil
}

func (*courseSwapRequestDAO) GetByIDList(ctx context.Context, iDList []int32,
) (courseSwapRequestMap map[int32]*CourseSwapRequestDO, err error) {
	if len(iDList) == 0 {
		return nil, nil
	}

	q := db.GetPG().CourseSwapRequest
	l, err := q.WithContext(ctx).
		Where(q.ID.In(iDList...)).Find()
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	courseSwapRequestMap = make(map[int32]*CourseSwapRequestDO)
	for _, i := range l {
		courseSwapRequestMap[i.ID] = i
	}
	return courseSwapRequestMap, nil
}

