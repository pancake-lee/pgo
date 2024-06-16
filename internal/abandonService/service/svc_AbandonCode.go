package service

import (
	"context"
	"gogogo/internal/abandonService/data"
	"gogogo/pkg/proto/api"
)

type AbandonCodeServer struct {
	api.UnimplementedAbandonCodeServer
}

func DO2DTO_AbandonCode(do *data.AbandonCodeDO) *api.AbandonCodeInfo {
	if do == nil {
		return nil
	}
	return &api.AbandonCodeInfo{
		// MARK 6 START 替换内容，所有字段DO2DTO
		Idx1: do.Idx1,
		Col1: do.Col1,
		// MARK 6 END
	}
}
func DTO2DO_AbandonCode(dto *api.AbandonCodeInfo) *data.AbandonCodeDO {
	if dto == nil {
		return nil
	}
	return &data.AbandonCodeDO{
		// MARK 7 START 替换内容，所有字段DTO2DO
		Idx1: dto.Idx1,
		Col1: dto.Col1,
		// MARK 7 END
	}
}

func (s *AbandonCodeServer) AddAbandonCode(
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

func (s *AbandonCodeServer) GetAbandonCodeList(
	ctx context.Context, req *api.GetAbandonCodeListRequest,
) (resp *api.GetAbandonCodeListResponse, err error) {

	var dataList []*data.AbandonCodeDO

	// MARK 5 START 替换内容，没有索引的表，以替换的形式删除
	if len(req.Idx1List) != 0 {
		dataMap, err := data.AbandonCodeDAO.GetByIds(ctx, req.Idx1List)
		if err != nil {
			return nil, err
		}
		for _, d := range dataMap {
			dataList = append(dataList, d)
		}
	} else {
		// MARK 5 END

		dataList, err = data.AbandonCodeDAO.GetAll(ctx)
		if err != nil {
			return nil, err
		}

		// MARK 5 START 替换内容，没有索引的表，以替换的形式删除
	}
	// MARK 5 END

	resp = new(api.GetAbandonCodeListResponse)
	resp.AbandonCodeList = make([]*api.AbandonCodeInfo, 0, len(dataList))
	for _, data := range dataList {
		resp.AbandonCodeList = append(resp.AbandonCodeList, DO2DTO_AbandonCode(data))
	}
	return resp, nil
}

// MARK 5 START 替换内容，没有索引的表，以替换的形式删除
func (s *AbandonCodeServer) DelAbandonCodeByIds(
	ctx context.Context, req *api.DelAbandonCodeByIdsRequest,
) (resp *api.Empty, err error) {
	if len(req.Idx1List) == 0 {
		return nil, nil
	}
	err = data.AbandonCodeDAO.DelByIds(ctx, req.Idx1List)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// MARK 5 END
