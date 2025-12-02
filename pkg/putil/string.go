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

func StrFirstToUpper(str string) string {
	return string(unicode.ToUpper(rune(str[0]))) + str[1:]
}
func StrFirstToLower(str string) string {
	return string(unicode.ToLower(rune(str[0]))) + str[1:]
}
func StrPrefixByNum(str string, num int) string {
	if num <= 0 || num > len(str) {
		return str
	}
	return str[:num]
}
