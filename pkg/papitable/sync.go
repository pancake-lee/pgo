package papitable

import (
	"fmt"
	"reflect"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

// 本文件主要提供的是SyncHelper
// 封装的内容主要是LTBL和MTBL的双向同步逻辑，尤其是如何避免无限回调的问题
// 同步数据的几个主要途径
// 1：初始化apitable，创建/更新表结构，并全量同步一次数据
// 		由GetSyncData实现LTBL操作，提供给InitMTBL调用
// 2：本地数据变更，触发同步数据到apitable
// 		包括UpdateToMTBL和DeleteMTBL，只要实现了双向数据转换就行，不用额外实现细节
// 3：apitable数据变更，出发本地数据更新
// 		由CreateOrUpdateLocalRecord实现LTBL操作，提供给UpdateToLTBL调用

// SyncHelper只封装同步策略和MTBL的大部分操作
// 而本地数据操作，以及数据转换等等都是DataProvider接口提供的
type DataProvider interface {
	// 如果内部创建新的表，传递到外部存储
	SetDoc(doc *MultiTableDoc)

	// --------------------------------------------------
	// 表配置
	GetTableName() string    // GetTableName 获取表名
	GetFirstCol() *AddField  // GetFirstCol 获取主键列定义
	GetColList() []*AddField // GetColList 获取所有列定义
	GetPrimaryCol() *FieldConfig
	GetPrimaryVal(localRecord any) string

	// --------------------------------------------------
	// 本地维护LastEditFrom字段用于避免循环更新，维护mtbl的recordId用于做删除逻辑

	// GetLastEditFrom 获取最后修改来源
	GetLastEditFrom(localRecord any) string
	// UpdateLastEditFromToLTBL 更新本地记录的 LastEditFrom 字段
	UpdateLastEditToLTBL(localId, mtblRecordId, lastEditFrom string) error

	// --------------------------------------------------
	GetSyncData() ([]*AddRecord, error) // 获取全量同步的数据，用于SyncHelper.InitMTBL

	// --------------------------------------------------
	// 主要提供给SyncHelper.UpdateToLTBL使用，用于把MTBL的变更应用到本地

	// GetLocalRecordByMtbl 根据MTBL记录获取本地记录，隐含了M2LTBL的反向映射逻辑
	// 返回error将中断处理，可以返回[nil,nil]继续处理mtbl新增数据，而ltbl不存在的情况
	GetLocalRecordByMtbl(mtblRecord *CommonRecord) (localRecord any, err error)

	// CreateOrUpdateLocalRecord 应用本地更新（写入DB）
	// localRecord: 旧的本地记录，可能是nil，一般意味着需要创建新本地记录
	// mtblRecord: 原始MTBL记录
	// 返回:
	//  newRecord: 创建或更新后的新数据
	// 	stop用于中断SyncHelper后续对MTBL的处理，
	// 		用户通过MTBL操作，但只是作为一个入口，触发后端复杂逻辑
	// 		而这个数据其实不需要维护，甚至被删除了，则SyncHelper.UpdateToLTBL不再需要继续处理
	CreateOrUpdateLocalRecord(localRecord any, mtblRecord *CommonRecord,
	) (newRecord any, stop bool, err error)

	// --------------------------------------------------
	// 提供本地数据和多维表格数据的双向转换方法

	// L2M 将本地记录转换为APITable记录
	// 先赋值oldRecord，再用localRecord覆盖
	L2M(localRecord any, mtblRecord *CommonRecord) map[string]any

	// M2L 将MTBL记录转换为本地记录结构
	// 如果localRecord不为nil，则直接用mtblRecord覆盖到localRecord
	M2L(mtblRecord *CommonRecord, localRecord any) any
}

// --------------------------------------------------
type SyncHelper struct {
	doc          *MultiTableDoc
	init         bool
	dataProvider DataProvider
	log          *plogger.PLogWarper
}

// 特别注意，删除LTBL的方法，99%都是业务代码，没有封装
// 为了防止回调循环，需要在业务代码里调用 LastEditFromMarker_LTBL.Set(recordId) 标记一下

func NewSyncHelper(data DataProvider, doc *MultiTableDoc) *SyncHelper {

	newLog := log.With(plogger.GetDefaultLogger(),
		"mtbl", data.GetTableName(),
	)
	plogger.SetPrefixKeys("mtbl")

	return &SyncHelper{
		dataProvider: data,
		doc:          doc,
		log:          plogger.NewPLogWarper(newLog),
	}
}

// init 的意义是，创建表格，全量写入数据时，MTBL的自动化还没有建立
// 所以双向同步的逻辑有点不一样，主要体现在 LastEditFrom 字段将被设置为LTBL，而不是TEMP
func (s *SyncHelper) SetInit(init bool) *SyncHelper {
	s.init = init
	return s
}
func (s *SyncHelper) WithLogger(logger *plogger.PLogWarper) *SyncHelper {

	newLog := log.With(logger.GetLogger(),
		"mtbl", s.dataProvider.GetTableName(),
	)
	plogger.SetPrefixKeys("mtbl")

	s.log = plogger.NewPLogWarper(newLog)
	// s.log.Debugf("SyncHelper set custom logger")
	return s
}

// InitMTBL 初始化多维表格
// spaceId: 空间ID
// datasheetId: 表格ID，如果为空则创建新表
// return: 最终的表格ID，错误信息
func (s *SyncHelper) InitMTBL(spaceId, datasheetId string) (string, error) {
	var doc *MultiTableDoc
	var err error

	// 1. 检查表格是否存在
	if datasheetId != "" {
		d := NewMultiTableDoc(spaceId, datasheetId)
		_, err = d.GetCols()
		if err != nil {
			s.log.Infof("spaceId[%s] sheet[%s] get col list fail, create new multi table", spaceId, datasheetId)
		} else {
			doc = d
			s.log.Infof("spaceId[%s] sheet[%s] exist, skip create", spaceId, datasheetId)
		}
	}

	// 2. 如果不存在则创建
	if doc == nil {
		doc, err = CreateMultiTable(spaceId, s.dataProvider.GetTableName(), s.dataProvider.GetFirstCol())
		if err != nil {
			s.log.Errorf("CreateMultiTable failed: %v", err)
			return "", err
		}
		s.log.Infof("CreateMultiTable success, datasheetId: %s", doc.DatasheetId)
		datasheetId = doc.DatasheetId
	}
	s.doc = doc                //设置给内部用
	s.dataProvider.SetDoc(doc) //设置给外部用，找到了或创建了，外部要知道目标doc

	// 3. 清空数据
	// TODO 可以触发每一行数据的update，而不是全量删除重建
	err = doc.DelAllRows()
	if err != nil {
		s.log.Errorf("DelAllRows failed: %v", err)
		return datasheetId, err
	}

	// 4. 更新列结构
	// SetColList 内部实现对比列配置，少了则补全，多了不用管
	colList := s.dataProvider.GetColList()
	colList = append(colList,
		NewSingleSelectCol("LastEdit", lastEditOptHandler.GetOptionList()),
	)

	_, err = doc.SetColList(colList, false)
	if err != nil {
		s.log.Errorf("SetColList failed: %v", err)
		return datasheetId, err
	}

	// 5. 全量写入数据
	rows, err := s.dataProvider.GetSyncData()
	if err != nil {
		s.log.Errorf("GetSyncData failed: %v", err)
		return datasheetId, err
	}

	for _, row := range rows {
		row.Values["LastEdit"] = lastEditOptHandler.GetCellOptionById_S(LastEditFrom_LTBL)
	}

	if len(rows) > 0 {
		_, err = doc.AddRow(rows)
		if err != nil {
			s.log.Errorf("AddRow failed: %v", err)
			return datasheetId, err
		}
	}

	return datasheetId, nil
}

// 从外部传入record，让业务代码复用record的获取逻辑
func (s *SyncHelper) UpdateToMTBL( // TODO 为什么是外部提供oldRecord呢？为什么让业务方自己查询？
	localRecord any,
	oldRecord *CommonRecord,
	updateTime time.Time,
) (err error) {
	if s.doc == nil {
		return fmt.Errorf("doc is nil")
	}

	if localRecord == nil {
		return fmt.Errorf("localRecord is nil")
	}

	localId := s.dataProvider.GetPrimaryVal(localRecord)
	if localId == "" {
		return fmt.Errorf("localId is empty")
	}
	// 检查循环标记 (防止本地更新触发的死循环)
	if LastEditFromMarker_LTBL.SkipAndClean(localId, updateTime) {
		s.log.Debugf("Local data [%s] is recently edited for [LastEditFrom], skip update", localId)
		return nil
	}

	localLastEditFrom := s.dataProvider.GetLastEditFrom(localRecord)

	// 修改对方 (MTBL)
	newRecordId := ""
	if oldRecord != nil {
		newRecordId = oldRecord.RecordId
	}

	if localLastEditFrom != LastEditFrom_TEMP {
		if oldRecord == nil {
			// 新增
			fieldList := s.dataProvider.L2M(localRecord, nil)
			if fieldList == nil {
				// 不一定是错误，也许localRecord参数不足，或者不需要同步，中止
				s.log.Infof("local L2M [%v] is nil, skip", localId)
				return nil
			}

			var newLastEditFrom string
			if s.init {
				newLastEditFrom = LastEditFrom_LTBL
			} else {
				newLastEditFrom = LastEditFrom_TEMP
			}
			fieldList["LastEdit"] = lastEditOptHandler.GetCellOptionById_S(newLastEditFrom)

			recordList, err := s.doc.AddRow([]*AddRecord{{Values: fieldList}})
			if err != nil {
				s.log.Errorf("Local[%v] add record to MTBL failed: %v", localId, err)
				return err
			}
			if len(recordList) > 0 {
				newRecordId = recordList[0].RecordId
				s.log.Debugf("Local[%v] create recordId[%s] to MTBL",
					localId, newRecordId)
			}

		} else {
			// 更新
			fieldList := s.dataProvider.L2M(localRecord, oldRecord)
			if fieldList == nil {
				s.log.Errorf("local L2M [%d] is nil, skip", localId)
				return nil
			}
			fieldList["LastEdit"] = lastEditOptHandler.GetCellOptionById_S(LastEditFrom_TEMP)

			err = s.doc.EditRow([]*UpdateRecord{
				{RecordId: oldRecord.RecordId, Fields: fieldList}})
			if err != nil {
				s.log.Errorf("Local[%v] update recordId[%s] to MTBL failed: %v", localId, oldRecord.RecordId, err)
				return err
			}
			s.log.Debugf("Local[%v] update recordId[%s] to MTBL", localId, oldRecord.RecordId)
		}
	}

	// 5. 修改自己 (lastEditFrom标识)
	// 如果是TEMP，则来自于apitable的回调又触发到本地的回调，则标记为MTBL，否则认为是人为在LTBL修改的
	var newLastEditFrom string
	if localLastEditFrom == LastEditFrom_TEMP {
		newLastEditFrom = LastEditFrom_MTBL
		s.log.Debugf("Local cb, LastEditFrom[MTBL], skip recycle update to MTBL")
	} else {
		newLastEditFrom = LastEditFrom_LTBL
	}

	err = s.dataProvider.UpdateLastEditToLTBL(localId, newRecordId, newLastEditFrom)
	if err != nil {
		s.log.Errorf("Local[%v] update LastEditFrom to LTBL failed: %v", localId, err)
		return err
	}

	// 设置标记，防止本次 UpdateLastEditToLTBL 再次触发 UpdateToMTBL
	LastEditFromMarker_LTBL.Set(localId)
	s.log.Debugf("Local rewrite [%v] recordId[%s] LastEditFrom[%v]",
		localId, newRecordId, newLastEditFrom)

	return nil
}

func (s *SyncHelper) DeleteMTBL(localId string, deleteTime time.Time) error {
	if s.doc == nil {
		return nil
	}
	if localId == "" {
		return nil
	}

	if LastEditFromMarker_LTBL.SkipAndClean(localId, deleteTime) {
		s.log.Debugf("Local data [%s] is recently delete by MTBL, skip", localId)
		return nil
	}

	mtblRecord, err := s.getMtblRecordByLocalId(localId)
	if err != nil {
		s.log.Warnf("getMtblRecordByLocalId failed for localId %s, skip, err: %v", localId, err)
		return err
	}

	err = s.doc.DelRow([]string{mtblRecord.RecordId})
	if err != nil {
		s.log.Errorf("DelRow failed for recordId %s, err: %v", mtblRecord.RecordId, err)
		return err
	}
	s.log.Debugf("MTBL delete recordId[%s]", mtblRecord.RecordId)
	return nil
}

func (s *SyncHelper) UpdateToLTBL(mtblRecord *CommonRecord) error {
	if s.doc == nil {
		s.log.Errorf("doc is nil")
		return nil
	}
	mtblRecordId := mtblRecord.RecordId

	// Check Loop Marker
	if LastEditFromMarker_MTBL.SkipAndClean(mtblRecordId, time.Unix(mtblRecord.UpdatedAt, 0)) {
		s.log.Debugf("MTBL recordId %s is recently edited for [LastEditFrom], skip update", mtblRecordId)
		return nil
	}

	// Get Local Record
	localRecord, err := s.dataProvider.GetLocalRecordByMtbl(mtblRecord)
	if err != nil {
		return err
	}

	s.log.Debugf("MTBL recordId %s edited, now sync to LTBL", mtblRecordId)

	// 基于本地数据，用MTBL的值覆盖对应字段，然后调用dataProvider的“更新方法”
	s.dataProvider.M2L(mtblRecord, localRecord)

	lastEditFrom := ""
	if localRecord != nil {
		lastEditFrom = s.dataProvider.GetLastEditFrom(localRecord)
	}

	if lastEditFrom != LastEditFrom_TEMP {
		// Manual edit in MTBL, apply to Local
		newRecord, stop, err := s.dataProvider.CreateOrUpdateLocalRecord(localRecord, mtblRecord)
		if err != nil {
			return err
		}
		if stop {
			s.log.Debugf("recordId %s is stopped by CreateOrUpdateLocalRecord, skip update", mtblRecordId)
			return nil
		}

		if reflect.ValueOf(newRecord).Kind() != reflect.Ptr {
			return plogger.LogErrfMsg("CreateOrUpdateLocalRecord for[%v] must return pointer type",
				s.dataProvider.GetTableName())
		}

		localRecord = newRecord
	}

	// Update Self (MTBL) LastEditFrom status
	var newLastEditFrom string
	// 如果是TEMP，则来自于本地的回调又触发到apitable的回调，则标记为LTBL，否则认为是人为在MTBL修改的
	if lastEditFrom == LastEditFrom_TEMP {
		newLastEditFrom = LastEditFrom_LTBL
		s.log.Debugf("MTBL cb, LastEditFrom[LTBL], skip recycle update to LTBL")
	} else {
		newLastEditFrom = LastEditFrom_MTBL
	}

	// 重新从LTBL转一次，这样可以通过 UpdateLocalRecord 中修改 localRecord 的值，更新到MTBL
	// 比如某些值不允许MTBL修改，或有更复杂的修改逻辑，强制覆盖回去
	fieldList := s.dataProvider.L2M(localRecord, mtblRecord)
	if fieldList == nil {
		s.log.Errorf("MTBL L2M [%s] is nil", mtblRecordId)
		return fmt.Errorf("MTBL L2M [%s] is nil", mtblRecordId)
	}
	fieldList["LastEdit"] = lastEditOptHandler.GetCellOptionById_S(newLastEditFrom)

	err = s.doc.EditRow([]*UpdateRecord{{RecordId: mtblRecordId, Fields: fieldList}})
	if err != nil {
		s.log.Errorf("UpdateRow failed for recordId %s: %v", mtblRecordId, err)
		return err
	}

	LastEditFromMarker_MTBL.Set(mtblRecordId)
	s.log.Debugf("MTBL rewrite recordId[%s] LastEditFrom[%v]", mtblRecordId, newLastEditFrom)
	return nil
}

func (s *SyncHelper) getMtblRecordByLocalId(localId string,
) (record *CommonRecord, err error) {
	resp, err := s.doc.GetRow(&GetRecordRequest{
		PageSize: 1,
		FilterByFormula: fmt.Sprintf(`AND({%v}="%v")`,
			s.dataProvider.GetPrimaryCol().Col.Name, localId),
	})
	if err != nil {
		s.log.Errorf("GetRow failed for data %s, err: %v", localId, err)
		return nil, err
	}
	if len(resp.Data.Records) == 0 {
		return nil, s.log.LogErrfMsg("local data %s not found in mtbl", localId)
	} else if len(resp.Data.Records) > 1 {
		s.log.LogErrfMsg("local data %s returned multiple records in mtbl", localId)
	}

	row := resp.Data.Records[0]
	return row, nil
}
