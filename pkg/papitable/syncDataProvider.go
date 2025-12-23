package papitable

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pmq"
)

// 本文件主要提供的是BaseMtblHandler，实现了通用的DataProvider
// 比如最基本的同步模式，只要提供MtblDAO接口实现即可
// 字段的赋值通过MtblTableConfig定义映射关系，然后通过反射实现
// 这份封装完全不是必要的，有任何特殊需求，都可以直接实现DataProvider接口

type FieldConfig struct {
	Col     *AddField
	DOField string
}

type TableConfig struct {
	TableName  string
	PrimaryCol *AddField
	ColList    []*FieldConfig
	NewDO      func() any
}

type MtblDAO interface {
	Add(ctx context.Context, do any) error
	UpdateByID(ctx context.Context, do any) error
	GetAll(ctx context.Context) ([]any, error)
	GetByID(ctx context.Context, id int32) (any, error)
	GetByMtblRecordID(ctx context.Context, recordId string) (any, error)
	DeleteByMtblRecordID(ctx context.Context, recordId string) error
}

type BaseDataProvider struct {
	Ctx context.Context
	// 该字段需要初始化设置好，mtbl事件中用于过滤属于当前表的事件
	DatasheetID string
	// 初始化总是nil，由 SyncHelper 根据mtbl事件自动加载
	// 经过DatasheetID字段的过滤，后续逻辑中的Doc其实也都是同一个表
	Doc *MultiTableDoc

	TableConfig *TableConfig
	DAO         MtblDAO

	L2MFunc   func(record any) map[string]any
	M2LFunc   func(mtblRecord *CommonRecord, localRecord any) any
	GetIDByDO func(record any) int32
}

// --------------------------------------------------
// 通用的处理mtbl变更事件方法

func (h *BaseDataProvider) HandleMtblEvent() error {
	pmqCtx := pmq.GetPMQContext(h.Ctx)
	var event ApiTableEvent
	err := json.Unmarshal([]byte(pmqCtx.Req), &event)
	if err != nil {
		plogger.Errorf("OnMtblUpdate unmarshal event err: %v", err)
		return err
	}
	if event.DatasheetId != h.DatasheetID {
		return nil
	}
	plogger.Infof("OnMtblUpdate received event: %+v", event)

	switch event.Event {
	case "insert", "update":
		return h.updateM2L(event.DatasheetId, event.RecordId)
	case "delete":
		return h.deleteM2L(event.RecordId)
	default:
		plogger.Warnf("skip unknown event type: %v", event.Event)
		return nil
	}
}

func (h *BaseDataProvider) deleteM2L(recordId string) error {
	LastEditFromMarker_LTBL.Set(recordId)
	err := h.DAO.DeleteByMtblRecordID(h.Ctx, recordId)
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

func (h *BaseDataProvider) updateM2L(datasheetId, recordId string) error {
	spaceId, err := pconfig.GetStringE("APITable.spaceId")
	if err != nil {
		plogger.Errorf("getTaskDoc: APITable.spaceId not found in config: %v", err)
		return nil
	}
	doc := NewMultiTableDoc(spaceId, datasheetId)

	resp, err := doc.GetRow(&GetRecordRequest{
		RecordIds: []string{recordId},
	})
	if err != nil {
		plogger.Errorf("GetRow failed for recordId %s, err: %v", recordId, err)
		return err
	}
	if len(resp.Data.Records) == 0 {
		plogger.Errorf("GetRow returned no data for recordId %s", recordId)
		return fmt.Errorf("recordId %s not found in mtbl", recordId)
	} else if len(resp.Data.Records) > 1 {
		plogger.Errorf("GetRow returned multiple data for recordId %s", recordId)
		return fmt.Errorf("recordId %s returned multiple records in mtbl", recordId)
	}

	row := resp.Data.Records[0]
	h.SetDoc(doc)

	syncHelper := NewSyncHelper(h, doc).
		WithLogger(plogger.GetDefaultLogWarper())
	err = syncHelper.UpdateToLTBL(row)
	if err != nil {
		return plogger.LogErr(err)
	}

	return nil
}

// --------------------------------------------------

// impl DataProvider
func (h *BaseDataProvider) GetTableName() string {
	return h.TableConfig.TableName
}

// impl DataProvider
func (h *BaseDataProvider) GetPrimaryCol() *AddField {
	return h.TableConfig.PrimaryCol
}

// impl DataProvider
func (h *BaseDataProvider) GetColList() []*AddField {
	cols := make([]*AddField, len(h.TableConfig.ColList))
	for i, cfg := range h.TableConfig.ColList {
		cols[i] = cfg.Col
	}
	return cols
}

// impl DataProvider
func (h *BaseDataProvider) SetDoc(doc *MultiTableDoc) {
	h.Doc = doc
}

// impl DataProvider
func (h *BaseDataProvider) GetSyncData() ([]*AddRecord, error) {
	list, err := h.DAO.GetAll(h.Ctx)
	if err != nil {
		return nil, err
	}
	var ret []*AddRecord
	for _, item := range list {
		row := h.L2M(item, nil)
		if row == nil {
			continue
		}
		ret = append(ret, &AddRecord{Values: row})
	}
	return ret, nil
}

// impl DataProvider
func (h *BaseDataProvider) GetLastEditFrom(record any) string {
	if record == nil {
		return ""
	}
	val := reflect.ValueOf(record)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return ""
	}

	field := val.FieldByName("LastEditFrom")
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}

// impl DataProvider
func (h *BaseDataProvider) UpdateMtblRecordID(id any, mtblRecordId, lastEditFrom string) error {
	return nil
}

// impl DataProvider
func (h *BaseDataProvider) GetLocalRecordByMtbl(mtblRecord *CommonRecord) (any, error) {
	// mtblRecordID找到了，就直接返回
	dbData, _ := h.DAO.GetByMtblRecordID(h.Ctx, mtblRecord.RecordId)
	if dbData != nil {
		return dbData, nil
	}
	// 尝试用主键查找
	if h.GetIDByDO == nil {
		return nil, nil
	}
	tmpData := h.M2L(mtblRecord, nil)
	id := h.GetIDByDO(tmpData)
	if id != 0 {
		return h.DAO.GetByID(h.Ctx, id)
	}
	return nil, nil
}

// impl DataProvider
func (h *BaseDataProvider) UpdateLocalRecord(localRecord any, mtblRecord *CommonRecord) (newRecord any, stop bool, err error) {
	newDbData := h.M2L(mtblRecord, localRecord)
	plogger.Debugf("MTBL recordId %s, parsed data: %v", mtblRecord.RecordId, newDbData)

	if localRecord == nil {
		err = h.DAO.Add(h.Ctx, newDbData)
		if err != nil {
			return nil, false, err
		}
		plogger.Debugf("MTBL add recordId[%s] to LTBL", mtblRecord.RecordId)
	} else {
		err = h.DAO.UpdateByID(h.Ctx, newDbData)
		if err != nil {
			return nil, false, err
		}
		plogger.Debugf("MTBL update recordId[%s] to LTBL", mtblRecord.RecordId)
	}
	return newDbData, false, nil
}

// impl DataProvider
func (h *BaseDataProvider) L2M(record any, oldMtblRecord *CommonRecord) map[string]any {
	if h.L2MFunc != nil {
		return h.L2MFunc(record)
	}

	if record == nil {
		return nil
	}

	val := reflect.ValueOf(record)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	valMap := make(map[string]any)
	for _, fieldCfg := range h.TableConfig.ColList {
		if fieldCfg.DOField == "" {
			continue
		}
		fieldVal := val.FieldByName(fieldCfg.DOField)
		if !fieldVal.IsValid() {
			plogger.Warnf("Field %s not found in record", fieldCfg.DOField)
			continue
		}

		// Convert based on type
		switch fieldCfg.Col.Type {
		case FIELD_TYPE_TEXT:
			valMap[fieldCfg.Col.Name] = NewTextValue(fmt.Sprintf("%v", fieldVal.Interface()))
		case FIELD_TYPE_NUMBER:
			var num float64
			switch fieldVal.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				num = float64(fieldVal.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				num = float64(fieldVal.Uint())
			case reflect.Float32, reflect.Float64:
				num = fieldVal.Float()
			default:
				num = 0
			}
			valMap[fieldCfg.Col.Name] = NewNumValue(num)
		}
	}
	return valMap
}

// impl DataProvider
func (h *BaseDataProvider) M2L(mtblRecord *CommonRecord, localRecord any) any {
	if h.M2LFunc != nil {
		return h.M2LFunc(mtblRecord, localRecord)
	}

	values := mtblRecord.Fields
	if values == nil {
		plogger.Warnf("mtbl record %s is nil", mtblRecord.RecordId)
		return nil
	}

	var recordVal reflect.Value
	if localRecord == nil {
		if h.TableConfig.NewDO == nil {
			plogger.Error("NewDO factory is nil")
			return nil
		}
		localRecord = h.TableConfig.NewDO()
	}

	recordVal = reflect.ValueOf(localRecord)
	if recordVal.Kind() == reflect.Ptr {
		recordVal = recordVal.Elem()
	}

	for _, fieldCfg := range h.TableConfig.ColList {
		if fieldCfg.DOField == "" {
			plogger.Warnf("mtbl field %s not found do field", fieldCfg.Col.Name)
			continue
		}

		v, ok := values[fieldCfg.Col.Name]
		if !ok {
			plogger.Warnf("mtbl field %s not found value", fieldCfg.Col.Name)
			continue
		}

		fieldVal := recordVal.FieldByName(fieldCfg.DOField)
		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			plogger.Warnf("ltbl field %s not found or cannot set", fieldCfg.DOField)
			continue
		}

		switch fieldCfg.Col.Type {
		case FIELD_TYPE_TEXT:
			strVal, _ := ParseTextValue(v)
			if fieldVal.Kind() == reflect.String {
				fieldVal.SetString(strVal)
				plogger.Debugf("Set field %s to value %v", fieldCfg.DOField, strVal)
			}
		case FIELD_TYPE_NUMBER:
			numVal, _ := ParseNumValue(v)
			switch fieldVal.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fieldVal.SetInt(int64(numVal))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fieldVal.SetUint(uint64(numVal))
			case reflect.Float32, reflect.Float64:
				fieldVal.SetFloat(numVal)
			}
			plogger.Debugf("Set field %s to value %v", fieldCfg.DOField, numVal)
		}
	}

	return localRecord
}
