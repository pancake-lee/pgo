package service

// MARK REPLACE IMPORT START
// MARK REPLACE IMPORT END

import (
	"context"

	"github.com/pancake-lee/pgo/api"
	"github.com/pancake-lee/pgo/internal/abandonCodeService/data"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func DO2DTO_AbandonCode(do *data.AbandonCodeDO) *api.AbandonCodeInfo {
	if do == nil {
		return nil
	}
	return &api.AbandonCodeInfo{
		// MARK REPLACE DO2DTO START 替换内容，所有字段DO2DTO
		Idx1: do.Idx1,
		Col1: do.Col1,
		// MARK REPLACE DO2DTO END
	}
}
func DTO2DO_AbandonCode(dto *api.AbandonCodeInfo) *data.AbandonCodeDO {
	if dto == nil {
		return nil
	}
	return &data.AbandonCodeDO{
		// MARK REPLACE DTO2DO START 替换内容，所有字段DTO2DO
		Idx1: dto.Idx1,
		Col1: dto.Col1,
		// MARK REPLACE DTO2DO END
	}
}

func (s *AbandonCodeCURDServer) AddAbandonCode(
	ctx context.Context, req *api.AddAbandonCodeRequest,
) (resp *api.AddAbandonCodeResponse, err error) {
	if req.AbandonCode == nil {
		return nil, api.ErrorInvalidArgument("")
	}
	newData := DTO2DO_AbandonCode(req.AbandonCode)

	err = data.AbandonCodeDAO.Add(ctx, newData)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	// MARK REMOVE IF NO PRIMARY KEY START
	plogger.Debugf("AddAbandonCode: %v", newData.Idx1)
	// MARK REMOVE IF NO PRIMARY KEY END

	resp = new(api.AddAbandonCodeResponse)
	resp.AbandonCode = DO2DTO_AbandonCode(newData)
	return resp, nil
}

func (s *AbandonCodeCURDServer) GetAbandonCodeList(
	ctx context.Context, req *api.GetAbandonCodeListRequest,
) (resp *api.GetAbandonCodeListResponse, err error) {

	var dataList []*data.AbandonCodeDO

	// MARK REMOVE IF NO PRIMARY KEY START 1
	if len(req.Idx1List) != 0 {
		plogger.Debugf("GetAbandonCodeList: %v", req.Idx1List)

		dataMap, err := data.AbandonCodeDAO.GetByIdx1List(ctx, req.Idx1List)
		if err != nil {
			return nil, plogger.LogErr(err)
		}
		for _, d := range dataMap {
			dataList = append(dataList, d)
		}
	} else {
		// MARK REMOVE IF NO PRIMARY KEY END 1

		dataList, err = data.AbandonCodeDAO.GetAll(ctx)
		if err != nil {
			return nil, plogger.LogErr(err)
		}

		// MARK REMOVE IF NO PRIMARY KEY START 2
	}
	// MARK REMOVE IF NO PRIMARY KEY END 2

	plogger.Debugf("GetAbandonCodeList resp len %v", len(dataList))

	resp = new(api.GetAbandonCodeListResponse)
	resp.AbandonCodeList = make([]*api.AbandonCodeInfo, 0, len(dataList))
	for _, data := range dataList {
		resp.AbandonCodeList = append(resp.AbandonCodeList, DO2DTO_AbandonCode(data))
	}
	return resp, nil
}

// MARK REMOVE IF NO PRIMARY KEY START

func (s *AbandonCodeCURDServer) UpdateAbandonCode(
	ctx context.Context, req *api.UpdateAbandonCodeRequest,
) (resp *api.UpdateAbandonCodeResponse, err error) {
	if req.AbandonCode == nil {
		return nil, api.ErrorInvalidArgument("")
	}

	do := DTO2DO_AbandonCode(req.AbandonCode)
	err = data.AbandonCodeDAO.UpdateByIdx1(ctx, do)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	plogger.Debugf("UpdateAbandonCode %v", req.AbandonCode.Idx1)

	resp = new(api.UpdateAbandonCodeResponse)
	d, err := data.AbandonCodeDAO.GetByIdx1(ctx, req.AbandonCode.Idx1)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	resp.AbandonCode = DO2DTO_AbandonCode(d)
	return resp, nil
}

func (s *AbandonCodeCURDServer) DelAbandonCodeByIdx1List(
	ctx context.Context, req *api.DelAbandonCodeByIdx1ListRequest,
) (resp *api.Empty, err error) {
	if len(req.Idx1List) == 0 {
		return nil, nil
	}
	err = data.AbandonCodeDAO.DelByIdx1List(ctx, req.Idx1List)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	plogger.Debugf("DelAbandonCodeByIdx1List %v", req.Idx1List)
	return nil, nil
}

// MARK REMOVE IF NO PRIMARY KEY END

// MARK REPEAT INDEX API START
func (s *AbandonCodeCURDServer) GetAbandonCodeByIdx23(
	ctx context.Context, req *api.GetAbandonCodeByIdx23Request,
) (resp *api.GetAbandonCodeByIdx23Response, err error) {
	if len(req.Idx2List) == 0 && len(req.Idx3List) == 0 {
		return nil, api.ErrorInvalidArgument("Idx2List and Idx3List cannot be both empty")
	}
	list, err := data.AbandonCodeDAO.GetByIdx23(ctx, req.Idx2List, req.Idx3List)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	resp = new(api.GetAbandonCodeByIdx23Response)
	for _, v := range list {
		resp.Data = append(resp.Data, DO2DTO_AbandonCode(v))
	}
	return resp, nil
}

// MARK REPEAT INDEX API END
