package service

// MARK REPLACE IMPORT START
// MARK REPLACE IMPORT END

import (
	"context"
	"pgo/api"
	"pgo/internal/abandonCodeService/data"
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
		return nil, api.ErrorInvalidArgument("request is invalid")
	}
	newData := DTO2DO_AbandonCode(req.AbandonCode)

	err = data.AbandonCodeDAO.Add(ctx,
		newData)
	if err != nil {
		return nil, err
	}
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
		dataMap, err := data.AbandonCodeDAO.GetByIdx1List(ctx, req.Idx1List)
		if err != nil {
			return nil, err
		}
		for _, d := range dataMap {
			dataList = append(dataList, d)
		}
	} else {
		// MARK REMOVE IF NO PRIMARY KEY END 1

		dataList, err = data.AbandonCodeDAO.GetAll(ctx)
		if err != nil {
			return nil, err
		}

		// MARK REMOVE IF NO PRIMARY KEY START 2
	}
	// MARK REMOVE IF NO PRIMARY KEY END 2

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
		return nil, api.ErrorInvalidArgument("request is invalid")
	}

	do := DTO2DO_AbandonCode(req.AbandonCode)
	err = data.AbandonCodeDAO.UpdateByIdx1(ctx, do)
	if err != nil {
		return nil, err
	}

	resp = new(api.UpdateAbandonCodeResponse)
	d, err := data.AbandonCodeDAO.GetByIdx1(ctx, req.AbandonCode.Idx1)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return nil, nil
}

// MARK REMOVE IF NO PRIMARY KEY END
