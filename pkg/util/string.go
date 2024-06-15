package util

import (
	"strings"
	"unicode"
)

func StrToCamelCase(str string) string {
	split := strings.Split(str, "_")
	for i := 0; i < len(split); i++ {
		split[i] = StrFirstToUpper(split[i])
	}
	camel := strings.Join(split, "")
	return camel
}

// gorm-gen 工具生成的字段采用ID命名
// protoc   工具生成的字段采用Id命名
func StrIdToLower(str string) string {
	return strings.ReplaceAll(str, "ID", "Id")
}
func StrFirstToUpper(str string) string {
	return string(unicode.ToUpper(rune(str[0]))) + str[1:]
}
func StrFirstToLower(str string) string {
	return string(unicode.ToLower(rune(str[0]))) + str[1:]
}
