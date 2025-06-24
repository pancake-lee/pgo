package putil

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
// 使用起来依然太复杂，简单的代码替换难以分析是orm代码还是pb代码，干脆proto用ID命名就行了
// 本次代码提交，先保留调用，实现为空
func StrIdToLower(str string) string {
	if true {
		return str
	}
	return strings.ReplaceAll(str, "ID", "Id")
}
func StrFirstToUpper(str string) string {
	return string(unicode.ToUpper(rune(str[0]))) + str[1:]
}
func StrFirstToLower(str string) string {
	return string(unicode.ToLower(rune(str[0]))) + str[1:]
}
