package papitable

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pmq"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// 使用SyncHelper的两个方案：
// 方案1：实现MtblDAO接口，然后配置一个BaseDataProvider即可
// 		本文件主要提供的是BaseMtblHandler
// 		封装的是通用的DataProvider，并且扩展事件回调的处理方法
//      从【实现DataProvider】简化到【实现MtblDAO接口】

// 方案2：直接实现DataProvider接口，完全自定义逻辑
// 		这份封装完全不是必要的，有任何特殊需求，都可以直接实现DataProvider接口

type FieldConfig struct {
	Col     *AddField
	DOField string
}

type TableConfig struct {
	TableName  string
	FirstCol   *AddField    // 多维表格的第一列，无法隐藏无法移动
	PrimaryCol *FieldConfig // 对应本地唯一标识字段，一般是主键字段
	ColList    []*FieldConfig
	NewDO      func() any
}

// 定义常规DAO应该提供的CURD方法
type MtblDAO interface {
	Add(ctx context.Context, do any) error
	UpdateByID(ctx context.Context, do any) error

	GetAll(ctx context.Context) ([]any, error)
	GetByID(ctx context.Context, id int32) (any, error)

	// 删除只能依靠mtblRecordId，因为回调中mtbl已经没有这个数据，无法二次查询到LocalId了
	DeleteByID(ctx context.Context, id int32) error
	UpdateMtblInfo(_ctx context.Context, localId string, lastEditFrom string) error
}

type BaseDataProvider struct {
	Ctx context.Context
	log *plogger.PLogWarper

	// 该字段需要初始化设置好，mtbl事件处理中用于过滤属于当前表的事件
	DatasheetID string
	// 初始化总是nil，由 SyncHelper 根据mtbl事件自动加载
	Doc *MultiTableDoc

	TableConfig *TableConfig
	DAO         MtblDAO
}

func (h *BaseDataProvider) WithLogger(logger *plogger.PLogWarper) *BaseDataProvider {
	newLog := log.With(logger.GetLogger(),
		"mtbl", h.GetTableName(),
	)
	plogger.SetPrefixKeys("mtbl")

	h.log = plogger.NewPLogWarper(newLog)
	return h
}

// --------------------------------------------------
// 通用的处理mtbl变更事件方法

type ApiTableEvent struct {
	DatasheetId string `json:"datasheetId,omitempty"` // 表格ID
	RecordId    string `json:"recordId,omitempty"`    // 记录ID
	Event       string `json:"event,omitempty"`       // 事件类型，insert/update/delete
}

func (h *BaseDataProvider) HandleMtblEvent() error {
	if h.Doc == nil {
		spaceId, err := pconfig.GetStringE("APITable.spaceId")
		if err != nil {
			h.log.Errorf("getTaskDoc: APITable.spaceId not found in config: %v", err)
			return err
		}
		h.SetDoc(NewMultiTableDoc(spaceId, h.DatasheetID))
	}

	pmqCtx := pmq.GetPMQContext(h.Ctx)
	var event ApiTableEvent
	err := json.Unmarshal([]byte(pmqCtx.Req), &event)
	if err != nil {
		h.log.Errorf("OnMtblUpdate unmarshal event err: %v", err)
		return err
	}
	if event.DatasheetId != h.DatasheetID {
		h.log.Infof("OnMtblUpdate skip other db update")
		return nil
	}
	h.log.Infof("OnMtblUpdate received event: %+v", event)

	switch event.Event {
	case "insert", "update":
		return h.updateM2L(event.RecordId)
	case "delete":
		return h.deleteM2L(event.RecordId)
	default:
		h.log.Warnf("skip unknown event type: %v", event.Event)
		return nil
	}
}

// mtbl中配置删除按钮，触发事件，再删除双方数据，而不是直接删除mtbl数据，没有提供删除回调
func (h *BaseDataProvider) deleteM2L(recordId string) error {
	mtblRecord, err := h.getMtblRecordByRecordId(recordId)
	if err != nil {
		return err
	}
	localTmp := h.M2L(mtblRecord, nil)
	localId := h.GetPrimaryVal(localTmp)
	localIdInt, err := putil.StrToInt32(localId)
	if err != nil {
		return h.log.LogErr(err)
	}

	LastEditFromMarker_LTBL.Set(localId)

	err = h.DAO.DeleteByID(h.Ctx, localIdInt)
	if err != nil {
		return h.log.LogErr(err)
	}
	return nil
}

func (h *BaseDataProvider) updateM2L(recordId string) error {
	row, err := h.getMtblRecordByRecordId(recordId)
	if err != nil {
		return err
	}

	syncHelper := NewSyncHelper(h, h.Doc).
		WithLogger(h.log)
	err = syncHelper.UpdateToLTBL(row)
	if err != nil {
		return h.log.LogErr(err)
	}

	return nil
}

func (h *BaseDataProvider) getMtblRecordByRecordId(recordId string,
) (record *CommonRecord, err error) {
	resp, err := h.Doc.GetRow(&GetRecordRequest{
		PageSize:  1,
		RecordIds: []string{recordId},
	})
	if err != nil {
		h.log.Errorf("GetRow failed for record %s, err: %v", recordId, err)
		return nil, err
	}
	if len(resp.Data.Records) == 0 {
		return nil, h.log.LogErrfMsg("mtbl data %s not found", recordId)
	} else if len(resp.Data.Records) > 1 {
		h.log.LogErrfMsg("mtbl data %s returned multiple records", recordId)
	}

	row := resp.Data.Records[0]
	return row, nil
}

// --------------------------------------------------
// impl DataProvider
func (h *BaseDataProvider) SetDoc(doc *MultiTableDoc) {
	h.Doc = doc
}

// impl DataProvider
func (h *BaseDataProvider) GetTableName() string {
	return h.TableConfig.TableName
}

// impl DataProvider
func (h *BaseDataProvider) GetFirstCol() *AddField {
	return h.TableConfig.FirstCol
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
func (h *BaseDataProvider) GetPrimaryCol() *FieldConfig {
	return h.TableConfig.PrimaryCol
}

// impl DataProvider
func (h *BaseDataProvider) GetPrimaryVal(record any) string {
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

	field := val.FieldByName(h.TableConfig.PrimaryCol.DOField)
	if !field.IsValid() {
		return ""
	}
	if field.Kind() == reflect.String {
		return field.String()
	} else if field.Kind() >= reflect.Int && field.Kind() <= reflect.Int64 {
		return fmt.Sprintf("%d", field.Int())
	} else if field.Kind() >= reflect.Uint && field.Kind() <= reflect.Uint64 {
		return fmt.Sprintf("%d", field.Uint())
	} else {
		h.log.Errorf("primary val type error : %v", field.Kind())
		return ""
	}
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
func (h *BaseDataProvider) UpdateLastEditToLTBL(localId string, lastEditFrom string) error {
	return h.DAO.UpdateMtblInfo(h.Ctx, localId, lastEditFrom)
}

// impl DataProvider
func (h *BaseDataProvider) GetLocalRecordByMtbl(mtblRecord *CommonRecord) (any, error) {
	localTmp := h.M2L(mtblRecord, nil)
	localId := h.GetPrimaryVal(localTmp)
	localIdInt, err := putil.StrToInt32(localId)
	if err != nil {
		return nil, h.log.LogErr(err)
	}
	return h.DAO.GetByID(h.Ctx, localIdInt)
}

// impl DataProvider
func (h *BaseDataProvider) CreateOrUpdateLocalRecord(
	localRecord any, mtblRecord *CommonRecord,
) (newRecord any, stop bool, err error) {
	newDbData := h.M2L(mtblRecord, localRecord)
	h.log.Debugf("MTBL recordId %s, parsed data: %v", mtblRecord.RecordId, newDbData)

	if localRecord == nil {
		err = h.DAO.Add(h.Ctx, newDbData)
		if err != nil {
			return nil, false, err
		}
		h.log.Debugf("MTBL add recordId[%s] to LTBL", mtblRecord.RecordId)
	} else {
		err = h.DAO.UpdateByID(h.Ctx, newDbData)
		if err != nil {
			return nil, false, err
		}
		h.log.Debugf("MTBL update recordId[%s] to LTBL", mtblRecord.RecordId)
	}
	return newDbData, false, nil
}

// impl DataProvider
func (h *BaseDataProvider) L2M(record any, oldMtblRecord *CommonRecord) map[string]any {
	if record == nil {
		return nil
	}

	val := reflect.ValueOf(record)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	valMap := make(map[string]any)
	for k, v := range oldMtblRecord.Fields {
		valMap[k] = v // mtbl数据可能有更多的字段，不要丢弃了
	}
	for _, fieldCfg := range h.TableConfig.ColList {
		if fieldCfg.DOField == "" {
			continue
		}
		fieldVal := val.FieldByName(fieldCfg.DOField)
		if !fieldVal.IsValid() {
			h.log.Warnf("Field %s not found in record", fieldCfg.DOField)
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
	values := mtblRecord.Fields
	if values == nil {
		h.log.Warnf("mtbl record %s is nil", mtblRecord.RecordId)
		return nil
	}

	var recordVal reflect.Value
	if putil.AnyIsNil(localRecord) {
		if h.TableConfig.NewDO == nil {
			h.log.Error("NewDO factory is nil")
			return nil
		}
		localRecord = h.TableConfig.NewDO()
	}

	recordVal = reflect.ValueOf(localRecord)
	if recordVal.Kind() == reflect.Ptr {
		recordVal = recordVal.Elem()
	}
	h.log.Debugf("M2L recordVal: %v", recordVal)

	// 这里报错未必是这里的问题，是localRecord在传进来之前就有问题
	if !recordVal.IsValid() || recordVal.Kind() != reflect.Struct {
		h.log.Errorf("M2L: recordVal is invalid or not a struct. IsValid: %v, Kind: %v", recordVal.IsValid(), recordVal.Kind())
		return nil
	}

	for _, fieldCfg := range h.TableConfig.ColList {
		if fieldCfg.DOField == "" {
			h.log.Warnf("mtbl field %s not found do field", fieldCfg.Col.Name)
			continue
		}

		v, ok := values[fieldCfg.Col.Name]
		if !ok {
			h.log.Warnf("mtbl field %s not found value", fieldCfg.Col.Name)
			continue
		}

		fieldVal := recordVal.FieldByName(fieldCfg.DOField)
		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			h.log.Warnf("ltbl field %s not found or cannot set", fieldCfg.DOField)
			continue
		}

		switch fieldCfg.Col.Type {
		case FIELD_TYPE_TEXT:
			strVal, _ := ParseTextValue(v)
			if fieldVal.Kind() == reflect.String {
				fieldVal.SetString(strVal)
				h.log.Debugf("Set field %s to value %v", fieldCfg.DOField, strVal)
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
			h.log.Debugf("Set field %s to value %v", fieldCfg.DOField, numVal)
		}
	}

	return localRecord
}
