package util

import (
	"reflect"
)

// 命名是copilot提供的，表示“集合”，主要用于存放slice和map的一些操作代码封装

func SliceIndex(x any, f func(i int) bool) int {
	rv := reflect.ValueOf(x)
	length := rv.Len()
	for i := 0; i < length; i++ {
		if f(i) {
			return i
		}
	}
	return -1
}
