package putil

import (
	"reflect"
)

// 命名是copilot提供的，表示"集合"，主要用于存放slice和map的一些操作代码封装

// int32数组交集，输出排序是根据nums2的排序
func Int32ListIntersect(list1 []int32, list2 []int32) (ret []int32) {
	if len(list1) == 0 || len(list2) == 0 {
		return []int32{}
	}

	m := make(map[int32]bool)
	for _, v := range list1 {
		m[v] = true
	}
	for _, v := range list2 {
		if m[v] {
			ret = append(ret, v)
			m[v] = false // 避免重复添加
		}
	}
	return ret
}

func StrListIntersect(list1 []string, list2 []string) (ret []string) {
	if len(list1) == 0 || len(list2) == 0 {
		return []string{}
	}

	m := make(map[string]bool)
	for _, v := range list1 {
		m[v] = true
	}
	for _, v := range list2 {
		if m[v] {
			ret = append(ret, v)
			m[v] = false // 避免重复添加
		}
	}
	return ret
}

// --------------------------------------------------

// 差集：输出，baseList中有，exclude中没有，的元素
func Int32ListExcept(baseList []int32, exclude []int32) (ret []int32) {
	if len(baseList) == 0 {
		return ret
	}
	if len(exclude) == 0 {
		return baseList
	}

	m := make(map[int32]bool)
	for _, v := range exclude {
		m[v] = true
	}
	for _, v := range baseList {
		if !m[v] {
			ret = append(ret, v)
		}
	}
	return ret
}

// 差集：输出，baseList中有，exclude中没有，的元素
func StrListExcept(baseList []string, exclude []string) (ret []string) {
	if len(baseList) == 0 {
		return ret
	}
	if len(exclude) == 0 {
		return baseList
	}

	m := make(map[string]struct{})
	for _, v := range exclude {
		m[v] = struct{}{}
	}
	for _, v := range baseList {
		_, ok := m[v]
		if !ok {
			ret = append(ret, v)
		}
	}
	return ret
}

// --------------------------------------------------

// Int32ListUnion int32数组并集，去除重复数据
func Int32ListUnion(list1 []int32, list2 []int32) []int32 {
	if len(list1) == 0 {
		return list2
	} else if len(list2) == 0 {
		return list1
	}
	m := make(map[int32]bool)
	for _, v := range list1 {
		m[v] = true
	}
	for _, v := range list2 {
		m[v] = true
	}
	ret := make([]int32, 0, len(m))
	for v := range m {
		ret = append(ret, v)
	}

	return ret
}

// StrListUnion string数组并集，去除重复数据
func StrListUnion(list1 []string, list2 []string) []string {
	if len(list1) == 0 {
		return list2
	} else if len(list2) == 0 {
		return list1
	}
	//将数组A转成map
	m := make(map[string]bool)
	for _, v := range list1 {
		m[v] = true
	}
	for _, v := range list2 {
		m[v] = true
	}
	ret := make([]string, 0, len(m))
	for v := range m {
		ret = append(ret, v)
	}

	return ret
}

// --------------------------------------------------

// 取名XOR异或，实际逻辑是(A-B)U(B-A)的集合b
// https://www.lodashjs.com/docs/lodash.xor
func StrListXOR(list1 []string, list2 []string) (ret []string) {
	c := StrListExcept(list1, list2)
	d := StrListExcept(list2, list1)
	return StrListUnion(c, d)
}

// --------------------------------------------------

// Int32ListUnique int32数组去重
func Int32ListUnique(list []int32) []int32 {
	if len(list) == 0 {
		return list
	}
	m := make(map[int32]bool)
	for _, v := range list {
		m[v] = true
	}
	ret := make([]int32, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	return ret
}

// StrListUnique string数组去重
func StrListUnique(list []string) []string {
	if len(list) == 0 {
		return list
	}
	m := make(map[string]bool)
	for _, v := range list {
		m[v] = true
	}
	ret := make([]string, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	return ret
}

// --------------------------------------------------

func WalkSliceByStep(x any, step int, cb func(s, e int) error) (err error) {
	rv := reflect.ValueOf(x)
	length := rv.Len()

	var curIndex int = 0
	for {
		if curIndex >= length {
			break
		}

		endIndex := curIndex + step
		if endIndex > length {
			endIndex = length
		}

		err = cb(curIndex, endIndex)
		if err != nil {
			return err
		}
		curIndex += step
	}
	return nil
}
