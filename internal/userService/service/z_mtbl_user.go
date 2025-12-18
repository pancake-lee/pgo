package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/papitable"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

var _ papitable.DataProvider = (*t_MtblUser)(nil)

type t_MtblUser struct {
	ctx context.Context
	doc *papitable.MultiTableDoc
}

func (dao *t_MtblUser) SetDoc(doc *papitable.MultiTableDoc) {
	dao.doc = doc
}

func (dao *t_MtblUser) GetTableName() string {
	return "人员表"
}

func (dao *t_MtblUser) GetPrimaryCol() *papitable.AddField {
	return papitable.NewTextCol("姓名")
}

func (dao *t_MtblUser) GetColList() []*papitable.AddField {
	return []*papitable.AddField{
		papitable.NewTextCol("姓名"),
		papitable.NewSimpleNumCol("UserID"),
	}
}

func (dao *t_MtblUser) L2MTBL(record any, oldMtblRecord *papitable.CommonRecord) map[string]any {
	if record == nil {
		return nil
	}
	u, ok := record.(*data.UserDO)
	if !ok {
		return nil
	}

	valMap := make(map[string]any)
	valMap["姓名"] = papitable.NewTextValue(u.UserName)
	valMap["UserID"] = papitable.NewNumValue(float64(u.ID))
	return valMap
}

func (dao *t_MtblUser) GetSyncData() ([]*papitable.AddRecord, error) {
	uList, err := data.UserDAO.GetAll(dao.ctx)
	if err != nil {
		return nil, err
	}

	sort.Slice(uList, func(i, j int) bool {
		return uList[i].UserName < uList[j].UserName
	})

	var ret []*papitable.AddRecord
	for _, u := range uList {
		row := dao.L2MTBL(u, nil)
		if row == nil {
			plogger.Errorf("user [%d] conv to mtbl data is nil, skip", u.ID)
			continue
		}
		ret = append(ret, &papitable.AddRecord{Values: row})
	}
	return ret, nil
}

func (dao *t_MtblUser) GetLastEditFrom(record any) string {
	u, ok := record.(*data.UserDO)
	if !ok {
		return ""
	}
	if u == nil {
		return ""
	}
	return u.LastEditFrom
}

func (dao *t_MtblUser) UpdateMtblRecordID(id any, mtblRecordId, lastEditFrom string) error {
	userID, ok := id.(int32)
	if !ok {
		return fmt.Errorf("invalid userID : %v", id)
	}

	err := data.UserDAO.UpdateMtblInfo(
		dao.ctx, userID, mtblRecordId, lastEditFrom)
	if err != nil {
		return fmt.Errorf("update user[%v] mtbl col err : %v", userID, err)
	}
	return nil
}

func (dao *t_MtblUser) M2L(mtblRecord *papitable.CommonRecord,
	localRecord any) any {
	return dao.m2l(mtblRecord, localRecord.(*data.UserDO))
}
func (dao *t_MtblUser) m2l(mtblRecord *papitable.CommonRecord,
	localRecord *data.UserDO) *data.UserDO {
	values := mtblRecord.Fields
	if values == nil {
		return nil
	}
	if localRecord == nil {
		localRecord = &data.UserDO{}
	}
	if v, ok := values["姓名"]; ok {
		localRecord.UserName, _ = papitable.ParseTextValue(v)
	}
	if v, ok := values["UserID"]; ok {
		numVal, _ := papitable.ParseNumValue(v)
		localRecord.ID = int32(numVal)
	}
	return localRecord
}

func (dao *t_MtblUser) GetLocalRecordByMtbl(mtblRecord *papitable.CommonRecord) (any, error) {
	var err error
	dbTask, _ := data.UserDAO.SelectByRecordId(dao.ctx, mtblRecord.RecordId)
	if dbTask == nil {
		tmpData := dao.m2l(mtblRecord, nil)
		dbTask, err = data.UserDAO.GetByID(dao.ctx, tmpData.ID)
		if err == nil {
			dbTask.MtblRecordID = mtblRecord.RecordId
		}
	}
	return dbTask, nil
}
func (dao *t_MtblUser) UpdateLocalRecord(localRecord any, mtblRecord *papitable.CommonRecord,
) (newRecord any, stop bool, err error) {

	var dbData *data.UserDO
	if localRecord != nil {
		var ok bool
		dbData, ok = localRecord.(*data.UserDO)
		if !ok {
			return nil, false, fmt.Errorf("invalid localRecord type")
		}
	}

	var newDbData data.UserDO
	newDbData.MtblRecordID = mtblRecord.RecordId
	if dbData != nil {
		newDbData = *dbData
	}
	dao.m2l(mtblRecord, &newDbData)
	plogger.Debugf("MTBL recordId %s, parsed user: %v", mtblRecord.RecordId, newDbData)

	if dbData == nil {
		// 新增记录
		err := data.UserDAO.Add(dao.ctx, &newDbData)
		if err != nil {
			return nil, false, err
		}
		plogger.Debugf("MTBL add recordId[%s] to LTBL", newDbData.MtblRecordID)

	} else {
		// 当前只允许修改UserName字段
		err := data.UserDAO.EditUserName(dao.ctx, newDbData.ID, newDbData.UserName)
		if err != nil {
			return nil, false, err
		}
		plogger.Debugf("MTBL update recordId[%s] to LTBL", newDbData.MtblRecordID)
	}
	return &newDbData, false, nil
}
