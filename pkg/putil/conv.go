package putil

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// IntToStr converts int to string.
func IntToStr(value int) string {
	return strconv.Itoa(value)
}

// UintToStr converts int to string.
func UintToStr(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

// Int8ToStr converts int32 to string.
func Int8ToStr(value int8) string {
	return strconv.FormatInt(int64(value), 10)
}

// Uint8ToStr converts int32 to string.
func Uint8ToStr(value uint8) string {
	return strconv.FormatUint(uint64(value), 10)
}

// Int16ToStr converts int32 to string.
func Int16ToStr(value int16) string {
	return strconv.FormatInt(int64(value), 10)
}

// Uint16ToStr converts int32 to string.
func Uint16ToStr(value uint16) string {
	return strconv.FormatUint(uint64(value), 10)
}

// Int32ToStr converts int32 to string.
func Int32ToStr(value int32) string {
	return strconv.FormatInt(int64(value), 10)
}

// Uint32ToStr converts int32 to string.
func Uint32ToStr(value uint32) string {
	return strconv.FormatUint(uint64(value), 10)
}

// Int64ToStr converts int64 to string.
func Int64ToStr(value int64) string {
	return strconv.FormatInt(value, 10)
}

// Uint64ToStr converts uint64 to string.
func Uint64ToStr(value uint64) string {
	return strconv.FormatUint(value, 10)
}

// ByteToStr converts byte to string.
func ByteToStr(value byte) string {
	return strconv.Itoa(int(value))
}

// Float32ToStr converts float32 to string.
func Float32ToStr(value float32) string {
	return strconv.FormatFloat(float64(value), 'f', -1, 32)
}

// Float32ToStr converts float32 to string. Keep prec of 2
func Float32ToStrPrec2(value float32) string {
	return strconv.FormatFloat(float64(value), 'f', 2, 32)
}

// Float64ToStr converts float64 to string.
func Float64ToStr(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// Float64ToStr converts float64 to string. Keep prec of 2
func Float64ToStrPrec2(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
func Float64ToStrByPrec(f float64, prec int) string {
	return strconv.FormatFloat(f, 'f', prec, 64)
}

// BoolToStr converts bool to string.
func BoolToStr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// StrToInt converts string to int.
func StrToInt(value string) (int, error) {
	return strconv.Atoi(value)
}

// StrToInt converts string to int.
func StrToUint(value string) (uint, error) {
	v, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}

// StrToInt8 converts string to int8.
func StrToInt8(value string) (int8, error) {
	v, err := strconv.ParseInt(value, 10, 8)
	if err != nil {
		return 0, err
	}
	return int8(v), nil
}

// StrToUint8 converts string to uint8.
func StrToUint8(value string) (uint8, error) {
	v, err := strconv.ParseUint(value, 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(v), nil
}

// StrToInt16 converts string to int16.
func StrToInt16(value string) (int16, error) {
	v, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		return 0, err
	}
	return int16(v), nil
}

// StrToUint16 converts string to int16.
func StrToUint16(value string) (uint16, error) {
	v, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(v), nil
}

// StrToInt32 converts string to int32.
func StrToInt32(value string) (int32, error) {
	v, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

// StrToInt32 converts string to int32.
func StrToInt32WithDefault(value string, defaultVal int32) int32 {
	v, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return defaultVal
	}
	return int32(v)
}

// StrToUint32 converts string to uint32.
func StrToUint32(value string) (uint32, error) {
	v, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// StrToInt64 converts string to int64.
func StrToInt64(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}

// StrToInt64 converts string to int64.
func StrToInt64WithDefault(value string, defaultVal int64) int64 {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultVal
	}
	return v
}

// StrToUint64 converts string to uint64.
func StrToUint64(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 64)
}

// StrToByte converts string to byte.
func StrToByte(value string) (byte, error) {
	v, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return byte(v), nil
}

// StrToFloat32 converts string to float32.
func StrToFloat32(value string) (float32, error) {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0, err
	}
	return float32(v), nil
}

// StrToFloat64 converts string to float64.
func StrToFloat64(value string) (float64, error) {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// StrToFloat64 converts string to float64.
func StrToFloat64WithDefault(value string, d float64) float64 {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return d
	}
	return v
}

// StrToBool converts string to bool.
func StrToBool(value string) (bool, error) {
	return strconv.ParseBool(strings.ToLower(value))
}

// --------------------------------------------------
// EncodeBase64 converts an input string to base64 string.
func EncodeBase64(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

// DecodeBase64 decode a base64 string.
func DecodeBase64(input string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(input)
}

// DecodeBase64Str decode a base64 string.
func DecodeBase64Str(input string) string {
	s, _ := base64.StdEncoding.DecodeString(input)
	return string(s)
}

// --------------------------------------------------
func EncodeHex(input []byte) string {
	return hex.EncodeToString(input)
}

func DecodeHex(input string) ([]byte, error) {
	return hex.DecodeString(input)
}

// --------------------------------------------------
// Length2Bytes converts an int64 value to a byte array.
func Length2Bytes(len int64, buffer []byte) []byte {
	binary.BigEndian.PutUint64(buffer, uint64(len))
	return buffer
}

// Bytes2Length converts a byte array to an int64 value.
func Bytes2Length(ret []byte) int64 {
	return int64(binary.BigEndian.Uint64(ret))
}

// --------------------------------------------------
// Int32ListToStr []int32 to string
func Int32ListToStr(Int32List []int32, split string) string {
	var outString strings.Builder
	cnt := len(Int32List)
	for n, id := range Int32List {
		outString.WriteString(Int32ToStr(id))
		if n < (cnt - 1) {
			outString.WriteString(split)
		}
	}
	return outString.String()
}

// StrToInt32List string to []int32
func StrToInt32List(str string, split string) (ret []int32, err error) {
	if str == "" {
		return ret, nil
	}
	subStrList := strings.Split(str, split)
	for _, v := range subStrList {
		if v == "" {
			continue
		}
		i, err := StrToInt32(v)
		if err != nil {
			return ret, err
		}
		ret = append(ret, i)
	}
	return ret, nil
}

func StrToInt32List2(str, start, end, split string) ([]int32, error) {
	str = strings.TrimPrefix(str, start)
	str = strings.TrimSuffix(str, end)
	if split == "" {
		i, err := StrToInt32(str)
		return []int32{i}, err
	}
	return StrToInt32List(str, split)
}

// --------------------------------------------------
func Int32ListToStrWithDelimiter(Int32List []int32, split, left, right string) string {
	var outString strings.Builder
	cnt := len(Int32List)
	for n, id := range Int32List {
		outString.WriteString(fmt.Sprintf("%s%d%s", left, id, right))
		if n < (cnt - 1) {
			outString.WriteString(split)
		}
	}
	return outString.String()
}

// StrToInt32List string to []int32
func StrToInt32ListWithDelimiter(str, split, left, right string) (ret []int32, err error) {
	if str == "" {
		return ret, nil
	}
	subStrList := strings.Split(str, split)
	for _, v := range subStrList {
		if v == "" {
			continue
		}
		v = strings.TrimPrefix(v, left)
		v = strings.TrimSuffix(v, right)
		i, err := StrToInt32(v)
		if err != nil {
			return ret, err
		}
		ret = append(ret, i)
	}
	return ret, nil
}

// --------------------------------------------------
// StringListToStr []string to string
func StrListToStr(stringList []string, split string) string {
	if len(stringList) == 0 {
		return ""
	}
	var outString strings.Builder
	cnt := len(stringList)
	for n, str := range stringList {
		outString.WriteString(str)
		if n < (cnt - 1) {
			outString.WriteString(split)
		}
	}
	return outString.String()
}

// StrToStringList string to []string, ignore empty string item
func StrToStrList(str string, split string) (ret []string) {
	if str == "" {
		return ret
	}
	subStrList := strings.Split(str, split)
	for _, v := range subStrList {
		if v != "" {
			ret = append(ret, v)
		}
	}
	return ret
}

// StrToStrList基础上，分割后不删除split字符
func StrToStrListWithSplit(str string, split string) (ret []string) {
	if str == "" {
		return ret
	}

	splitLen := len(split)
	start := 0

	for i := 0; i < len(str); i++ {
		if i+splitLen <= len(str) && str[i:i+splitLen] == split {
			if start != i {
				ret = append(ret, str[start:i])
			}
			ret = append(ret, split)
			i += splitLen - 1
			start = i + 1
		}
	}

	if start < len(str) {
		ret = append(ret, str[start:])
	}

	return ret
}

// StrToStringList string to []string, include empty string item
func StrToStrListWithEmpty(str string, split string) (ret []string) {
	if str == "" {
		return ret
	}
	subStrList := strings.Split(str, split)
	for i, v := range subStrList {
		if i == 0 && v == "" { //如果第一个字符就是split，那么第一个元素就是空字符串，忽略该空字符
			continue
		}
		if i == len(subStrList)-1 && v == "" { //如果最后一个字符就是split，那么最后一个元素就是空字符串，忽略该空字符
			continue
		}
		ret = append(ret, v)
	}
	return ret
}

func StrToStrListByStartAndEnd(s string, splitStart string, splitEnd string) (ret []string) {
	if s == "" {
		return ret
	}
	startIndex := 0

	for {
		openIndex := strings.Index(s[startIndex:], splitStart)
		if openIndex == -1 {
			break
		}
		openIndex += startIndex

		closeIndex := strings.Index(s[openIndex:], splitEnd)
		if closeIndex == -1 {
			break
		}
		closeIndex += openIndex

		if openIndex > startIndex {
			ret = append(ret, s[startIndex:openIndex])
		}
		ret = append(ret, s[openIndex:closeIndex+1])

		startIndex = closeIndex + len(splitEnd)
	}

	if startIndex < len(s) {
		ret = append(ret, s[startIndex:])
	}

	return ret
}

// --------------------------------------------------

// 根据替换映射表替换字符串，要注意map的数据，不要造成循环替换
func ReplaceByStringMap(str string, replaceMap map[string]string) string {
	for oldStr, newStr := range replaceMap {
		str = strings.ReplaceAll(str, oldStr, newStr)
	}
	return str
}

// --------------------------------------------------
func AnyToInt32(val any, defaultVal int32) int32 {
	switch val := val.(type) {
	case int32:
		return val
	case int64:
		return int32(val)
	case int:
		return int32(val)
	case float32:
		return int32(val)
	case float64:
		return int32(val)
	case string:
		vInt, err := StrToInt32(val)
		if err != nil {
			return defaultVal
		}
		return vInt
	default:
		return defaultVal
	}
}
func AnyToStr(val any) string {
	switch val := val.(type) {
	case string:
		return val
	case *string:
		return *val
	case int32:
		return Int32ToStr(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// 接口变量只有在 类型（Type）和值（Value）都为 nil 时，才被认为是 nil
func AnyIsNil(val any) bool {
	if val == nil {
		return true
	}
	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Chan &&
		rv.Kind() != reflect.Func &&
		rv.Kind() != reflect.Interface &&
		rv.Kind() != reflect.Map &&
		rv.Kind() != reflect.Ptr &&
		rv.Kind() != reflect.Slice {
		return false
	}
	return rv.IsNil()
}

// --------------------------------------------------

func FillPrefixToLen(in, prefix string, length int) string {
	if len(in) >= length {
		return in
	}
	return strings.Repeat(prefix, length-len(in)) + in
}

// --------------------------------------------------
// 尝试将数据转换为UTF-8
// TODO 并不能自动识别编码，有时候用错编码，乱码，但是属于可见字符，也会被认为是UTF-8合法
func strToUTF8(str string) (string, error) {
	if utf8.ValidString(str) {
		return str, nil // 如果转换后的数据是有效的UTF-8，返回结果
	}
	// 定义可能的编码
	var (
		gbk     = simplifiedchinese.GBK
		gb2312  = simplifiedchinese.HZGB2312
		utf16LE = unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
		utf16BE = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
	)

	// 尝试转换为UTF-8
	for _, e := range []encoding.Encoding{gbk, gb2312, utf16LE, utf16BE} {
		reader := transform.NewReader(strings.NewReader(str), e.NewDecoder())
		converted, err := io.ReadAll(reader)
		if err != nil {
			continue // 如果转换失败，尝试下一个编码
		}
		newStr := string(converted)
		if utf8.ValidString(newStr) {
			return newStr, nil // 如果转换后的数据是有效的UTF-8，返回结果
		}
	}

	// 如果没有找到合适的编码，返回错误
	return str, fmt.Errorf("unknown encoding")
}

// --------------------------------------------------
// snake_case
// lowerCamelCase
// UpperCamelCase

func StrToUpperCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}
