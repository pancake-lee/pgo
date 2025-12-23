package papitable

import (
	"fmt"
	"time"

	"github.com/pancake-lee/pgo/pkg/plogger"
)

// 本文件主要提供的是SyncHelper
// 封装的内容主要是LTBL和MTBL的双向同步逻辑
// 而双方表格具体的字段映射关系，数据获取等等都是DataProvider接口提供的

// 同步数据到apitable
// 1：初始化apitable，创建表结构（或对比表结构差异，更新表结构），并全量同步一次数据
// 2：本地数据变更，触发同步数据到apitable
// 3：apitable数据变更，出发本地数据更新

type ApiTableEvent struct {
	DatasheetId string `json:"datasheetId,omitempty"` // 表格ID
	RecordId    string `json:"recordId,omitempty"`    // 记录ID
	Event       string `json:"event,omitempty"`       // 事件类型，insert/update/delete
}

// 由业务方提供数据的读写方法，以及apitable的字段映射关系等
type DataProvider interface {
	// 如果内部创建新的表，需要提供到外部存储
	SetDoc(doc *MultiTableDoc)

	// GetTableName 获取表名
	GetTableName() string
	// GetPrimaryCol 获取主键列定义
	GetPrimaryCol() *AddField
	// GetColList 获取所有列定义
	GetColList() []*AddField

	// GetSyncData 获取全量同步的数据
	GetSyncData() ([]*AddRecord, error)
	// GetLastEditFrom 获取最后修改来源
	GetLastEditFrom(record any) string

	// UpdateMtblRecordID 更新本地记录关联的APITable记录ID和LastEditFrom 字段
	UpdateMtblRecordID(id any, mtblRecordId, lastEditFrom string) error

	// GetLocalRecordByMtbl 根据MTBL记录获取本地记录，隐含了M2LTBL的反向映射逻辑
	// mtblRecord: MTBL记录
	// 返回error将中断处理，可以返回nil,nil继续处理mtbl新增数据，而ltbl不存在的情况
	// 至少需要在GetLastEditFrom/UpdateLocalRecord中处理ltbl不存在的情况，并且创建新记录
	// 返回: 本地记录, error
	GetLocalRecordByMtbl(mtblRecord *CommonRecord) (any, error)

	// UpdateLocalRecord 应用本地更新（写入DB）
	// localRecord: 旧的本地记录（如果存在）
	// mtblRecord: 原始MTBL记录（用于某些特殊逻辑，如删除）
	// 返回:
	//  newRecord: 创建或更新后的新数据
	// 	stop用于中断SyncHelper后续对MTBL的处理，
	// 	通过MTBL操作，触发后端复杂逻辑后，对于当前回调的MTBL记录，可能不再需要继续处理，可能已经删除了
	UpdateLocalRecord(localRecord any, mtblRecord *CommonRecord,
	) (newRecord any, stop bool, err error)

	// L2M 将本地记录转换为APITable记录
	// record: 本地记录
	// oldMtblRecord: APITable上的旧记录（如果是更新）
	L2M(record any, oldMtblRecord *CommonRecord) map[string]any

	// M2L 将MTBL记录转换为本地记录结构
	// mtblRecord: MTBL记录
	// localRecord: 现有的本地记录（如果存在），用于合并或更新
	// 返回: 准备好用于更新的本地记录对象
	M2L(mtblRecord *CommonRecord, localRecord any) any
}

type SyncHelper struct {
	doc          *MultiTableDoc
	init         bool
	dataProvider DataProvider
	log          *plogger.PLogWarper
}

// 特别注意，删除LTBL的方法，99%都是业务代码，没有封装
// 为了防止回调循环，需要在业务代码里调用 LastEditFromMarker_LTBL.Set(recordId) 标记一下

func NewSyncHelper(data DataProvider, doc *MultiTableDoc) *SyncHelper {
	return &SyncHelper{
		dataProvider: data,
		doc:          doc,
		log:          plogger.GetDefaultLogWarper(),
	}
}

// init 的意义是，创建表格，全量写入数据时，MTBL的自动化还没有建立
// 所以双向同步的逻辑有点不一样，主要体现在 LastEditFrom 字段将被设置为LTBL，而不是TEMP
func (s *SyncHelper) SetInit(init bool) *SyncHelper {
	s.init = init
	return s
}
func (s *SyncHelper) WithLogger(logger *plogger.PLogWarper) *SyncHelper {
	s.log = logger
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
		doc, err = CreateMultiTable(spaceId, s.dataProvider.GetTableName(), s.dataProvider.GetPrimaryCol())
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
func (s *SyncHelper) UpdateToMTBL(
	localId any, localRecord any,
	oldRecord *CommonRecord,
	updateTime time.Time,
) (err error) {
	if s.doc == nil {
		return fmt.Errorf("doc is nil")
	}

	if localRecord == nil {
		return fmt.Errorf("localRecord is nil")
	}

	var mtblRecordId string
	if oldRecord != nil {
		mtblRecordId = oldRecord.RecordId
	}
	// 检查循环标记 (防止本地更新触发的死循环)
	if LastEditFromMarker_LTBL.SkipAndClean(mtblRecordId, updateTime) {
		s.log.Debugf("Local recordId %s is recently edited for [LastEditFrom], skip update", mtblRecordId)
		return nil
	}

	localLastEditFrom := s.dataProvider.GetLastEditFrom(localRecord)

	// 修改对方 (MTBL)
	if localLastEditFrom != LastEditFrom_TEMP {
		if oldRecord == nil {
			// 新增
			fieldList := s.dataProvider.L2M(localRecord, nil)
			if fieldList == nil {
				// 不一定是错误，也许localRecord参数不足，或者不需要同步，中止
				s.log.Infof("local L2M [%d] is nil, skip", localId)
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
				mtblRecordId = recordList[0].RecordId
			}
			s.log.Debugf("Local[%v] create recordId[%s] to MTBL", localId, mtblRecordId)

		} else {
			// 更新
			fieldList := s.dataProvider.L2M(localRecord, oldRecord)
			if fieldList == nil {
				s.log.Errorf("local L2M [%d] is nil, skip", localId)
				return nil
			}
			fieldList["LastEdit"] = lastEditOptHandler.GetCellOptionById_S(LastEditFrom_TEMP)

			err = s.doc.EditRow([]*UpdateRecord{{RecordId: mtblRecordId, Fields: fieldList}})
			if err != nil {
				s.log.Errorf("Local[%v] update recordId[%s] to MTBL failed: %v", localId, mtblRecordId, err)
				return err
			}
			s.log.Debugf("Local[%v] update recordId[%s] to MTBL", localId, mtblRecordId)
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

	err = s.dataProvider.UpdateMtblRecordID(localId, mtblRecordId, newLastEditFrom)
	if err != nil {
		s.log.Errorf("Local[%v] update recordId[%s] to LTBL failed: %v", localId, mtblRecordId, err)
		return err
	}

	// 设置标记，防止本次 UpdateMtblRecordID 再次触发 UpdateToMTBL
	LastEditFromMarker_LTBL.Set(mtblRecordId)
	s.log.Debugf("Local rewrite recordId[%s] LastEditFrom[%v]", mtblRecordId, newLastEditFrom)

	return nil
}

func (s *SyncHelper) DeleteMTBL(mtblRecordId string, deleteTime time.Time) error {
	if s.doc == nil {
		return nil
	}
	if mtblRecordId == "" {
		return nil
	}

	if LastEditFromMarker_LTBL.SkipAndClean(mtblRecordId, deleteTime) {
		s.log.Debugf("Local recordId %s is recently delete by MTBL, skip", mtblRecordId)
		return nil
	}

	err := s.doc.DelRow([]string{mtblRecordId})
	if err != nil {
		s.log.Errorf("DelRow failed for recordId %s, err: %v", mtblRecordId, err)
		return err
	}
	s.log.Debugf("MTBL delete recordId[%s]", mtblRecordId)
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

	s.log.Debugf("MTBL recordId %s edited, now sync to LTBL", mtblRecordId)

	// Get Local Record
	localRecord, err := s.dataProvider.GetLocalRecordByMtbl(mtblRecord)
	if err != nil {
		return err
	}

	// 基于本地数据，用MTBL的值覆盖对应字段，然后调用dataProvider的“更新方法”
	s.dataProvider.M2L(mtblRecord, localRecord)

	lastEditFrom := s.dataProvider.GetLastEditFrom(localRecord)
	if lastEditFrom != LastEditFrom_TEMP {
		// Manual edit in MTBL, apply to Local
		newRecord, stop, err := s.dataProvider.UpdateLocalRecord(localRecord, mtblRecord)
		if err != nil {
			return err
		}
		if stop {
			s.log.Debugf("recordId %s is stopped by UpdateLocalRecord, skip update", mtblRecordId)
			return nil
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

// --------------------------------------------------
var LastEditFrom_TEMP string = "TEMP"
var LastEditFrom_LTBL string = "LTBL"
var LastEditFrom_MTBL string = "MTBL"
var lastEditOptHandler SelectFieldOptionHandler

func init() {
	lastEditOptHandler.RegOptionS(LastEditFrom_TEMP, LastEditFrom_TEMP, ColorRed)
	lastEditOptHandler.RegOptionS(LastEditFrom_LTBL, LastEditFrom_LTBL, ColorPurple)
	lastEditOptHandler.RegOptionS(LastEditFrom_MTBL, LastEditFrom_MTBL, ColorBlue)
}

// --------------------------------------------------
type LastEditFromMarker struct {
	recordToTimeMap map[string]time.Time
}

// local table
var LastEditFromMarker_LTBL = &LastEditFromMarker{
	recordToTimeMap: make(map[string]time.Time)}

// multi table
var LastEditFromMarker_MTBL = &LastEditFromMarker{
	recordToTimeMap: make(map[string]time.Time)}

func init() {
	// 定期清理过期的记录
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			LastEditFromMarker_LTBL.CleanTimeoutRecord()
			LastEditFromMarker_MTBL.CleanTimeoutRecord()
		}
	}()
}

func (m *LastEditFromMarker) CleanTimeoutRecord() {
	for k, v := range m.recordToTimeMap {
		if time.Since(v) > 5*time.Minute {
			delete(m.recordToTimeMap, k)
		}
	}
}

func (m *LastEditFromMarker) Set(recordId string) {
	m.recordToTimeMap[recordId] = time.Now()
}

func (m *LastEditFromMarker) SkipAndClean(recordId string, updateTime time.Time) bool {
	t, ok := m.recordToTimeMap[recordId]
	if ok && updateTime.Sub(t).Abs() < 1*time.Second {
		// 其实updateTime一定是晚于t的，abs纯粹保险
		delete(m.recordToTimeMap, recordId)
		return true
	}
	return false
}
