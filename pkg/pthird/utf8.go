package pthird

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func Conv_gbk_utf8(s string) (string, error) {
	reader := transform.NewReader(strings.NewReader(s),
		simplifiedchinese.GBK.NewDecoder())

	// 读取转换后的内容
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(utf8Bytes), nil
}

func Conv_gb2312_utf8(s string) (string, error) {
	reader := transform.NewReader(strings.NewReader(s),
		simplifiedchinese.HZGB2312.NewDecoder())

	// 读取转换后的内容
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(utf8Bytes), nil
}

func Conv_utf16be_utf8(s string) (string, error) {
	reader := transform.NewReader(strings.NewReader(s),
		unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder())

	// 读取转换后的内容
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(utf8Bytes), nil
}

func Conv_utf16le_utf8(s string) (string, error) {
	reader := transform.NewReader(strings.NewReader(s),
		unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())

	// 读取转换后的内容
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(utf8Bytes), nil
}

// --------------------------------------------------
// 尝试将数据转换为UTF-8
// TODO 并不能自动识别编码，有时候用错编码，乱码，但是属于可见字符，也会被认为是UTF-8合法
func Conv_auto_utf8(str string) (string, error) {
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
