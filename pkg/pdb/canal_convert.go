package pdb

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mysql-org/go-mysql/schema"
)

const canalTimeFormat = "2006-01-02 15:04:05"

// typeCache caches the map of column names to struct field indices
var typeCache sync.Map // map[reflect.Type]map[string]int

// getFieldMap returns a map of column name to field index for a given struct type.
// It prioritizes "gorm" tags (column:xxx), then "json" tags, then field names.
func getFieldMap(t reflect.Type) map[string]int {
	if m, ok := typeCache.Load(t); ok {
		return m.(map[string]int)
	}

	m := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// 1. Try gorm tag
		tag := field.Tag.Get("gorm")
		name := ""
		if tag != "" {
			parts := strings.Split(tag, ";")
			for _, p := range parts {
				if strings.HasPrefix(p, "column:") {
					name = strings.TrimPrefix(p, "column:")
					break
				}
			}
		}
		// 2. Try json tag
		if name == "" {
			tag := field.Tag.Get("json")
			if tag != "" {
				name = strings.Split(tag, ",")[0]
			}
		}
		// 3. Fallback to field name (snake_case conversion omitted for simplicity, usually tags are present)
		if name != "" {
			m[name] = i
		}
	}
	typeCache.Store(t, m)
	return m
}

// MapRowToStruct converts a row from canal.RowsEvent to a struct using reflection.
// This is more efficient than JSON marshaling/unmarshaling.
func MapRowToStruct(columns []schema.TableColumn, row []interface{}, dest interface{}) error {
	destPtr := reflect.ValueOf(dest)
	if destPtr.Kind() != reflect.Ptr || destPtr.IsNil() {
		return fmt.Errorf("dest must be a non-nil pointer to struct")
	}
	destVal := destPtr.Elem()
	if destVal.Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	fieldMap := getFieldMap(destVal.Type())

	for i, col := range columns {
		if i >= len(row) {
			break
		}
		val := row[i]
		if val == nil {
			continue
		}

		fieldIdx, ok := fieldMap[col.Name]
		if !ok {
			continue
		}

		field := destVal.Field(fieldIdx)
		if !field.CanSet() {
			continue
		}

		if err := setField(field, val); err != nil {
			return fmt.Errorf("failed to set field %s (col %s): %w", destVal.Type().Field(fieldIdx).Name, col.Name, err)
		}
	}
	return nil
}

func setField(field reflect.Value, val interface{}) error {
	switch field.Kind() {
	case reflect.String:
		return setString(field, val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setInt(field, val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return setUint(field, val)
	case reflect.Float32, reflect.Float64:
		return setFloat(field, val)
	case reflect.Bool:
		return setBool(field, val)
	case reflect.Struct:
		if field.Type() == reflect.TypeOf(time.Time{}) {
			return setTime(field, val)
		}
	}
	// Ignore unsupported types or implement as needed
	return nil
}

func setString(field reflect.Value, val interface{}) error {
	switch v := val.(type) {
	case string:
		field.SetString(v)
	case []byte:
		field.SetString(string(v))
	default:
		field.SetString(fmt.Sprint(val))
	}
	return nil
}

func setInt(field reflect.Value, val interface{}) error {
	i, err := toInt64(val)
	if err != nil {
		return err
	}
	field.SetInt(i)
	return nil
}

func setUint(field reflect.Value, val interface{}) error {
	i, err := toInt64(val) // Reuse toInt64 for simplicity, assuming positive values
	if err != nil {
		return err
	}
	field.SetUint(uint64(i))
	return nil
}

func setFloat(field reflect.Value, val interface{}) error {
	f, err := toFloat64(val)
	if err != nil {
		return err
	}
	field.SetFloat(f)
	return nil
}

func setBool(field reflect.Value, val interface{}) error {
	b, err := toBool(val)
	if err != nil {
		return err
	}
	field.SetBool(b)
	return nil
}

func setTime(field reflect.Value, val interface{}) error {
	switch v := val.(type) {
	case time.Time:
		field.Set(reflect.ValueOf(v))
	case string:
		// Try parsing common MySQL time formats
		// 1. DateTime: "2006-01-02 15:04:05"
		t, err := time.ParseInLocation(canalTimeFormat, v, time.Local)
		if err == nil {
			field.Set(reflect.ValueOf(t))
			return nil
		}
		// 2. Date: "2006-01-02"
		t, err = time.ParseInLocation("2006-01-02", v, time.Local)
		if err == nil {
			field.Set(reflect.ValueOf(t))
			return nil
		}
		return fmt.Errorf("cannot parse time string: %s", v)
	default:
		return fmt.Errorf("cannot convert %T to time.Time", val)
	}
	return nil
}

func toInt64(val interface{}) (int64, error) {
	switch v := val.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	}
	return 0, fmt.Errorf("cannot convert %T to int64", val)
}

func toFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	case []byte:
		return strconv.ParseFloat(string(v), 64)
	}
	return 0, fmt.Errorf("cannot convert %T to float64", val)
}

func toBool(val interface{}) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	case int, int8, int16, int32, int64:
		i, _ := toInt64(v)
		return i != 0, nil
	case string:
		return strconv.ParseBool(v)
	}
	return false, fmt.Errorf("cannot convert %T to bool", val)
}
