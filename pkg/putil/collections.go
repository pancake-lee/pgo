package putil

// 命名是copilot提供的，表示“集合”，主要用于存放slice和map的一些操作代码封装

// int32数组交集，输出排序是根据nums2的排序
func Int32ListIntersect(nums1 []int32, nums2 []int32) []int32 {
	if len(nums1) == 0 {
		return nums1
	}
	if len(nums2) == 0 {
		return nums2
	}

	m := make(map[int32]int32, 0)
	for _, v := range nums1 {
		m[v] += 1
	}
	count := 0 //记录新数组长度
	for _, v := range nums2 {
		if m[v] > 0 {
			m[v] = 0
			nums1[count] = v
			count++
		}
	}
	return nums1[:count]
}

// 差集：输出，baseList中有，exclude中没有，的元素
func Int32ListExcept(baseList []int32, exclude []int32) (ret []int32) {
	if len(baseList) == 0 {
		return ret
	}
	if len(exclude) == 0 {
		return baseList
	}

	m := make(map[int32]struct{})
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

// Int32ListUnion int32数组并集，去除重复数据
func Int32ListUnion(nums1 []int32, nums2 []int32) []int32 {
	if len(nums1) == 0 {
		return nums2
	} else if len(nums2) == 0 {
		return nums1
	}
	//将数组A转成map
	m := make(map[int32]int32)
	for _, v := range nums1 {
		m[v] = 0
	}
	//遍历数组B
	for _, v := range nums2 {
		//判断B中的元素在A是否存在
		_, ok := m[v]
		if !ok {
			//不存在，直接插入A列表中
			nums1 = append(nums1, v)
		}
	}
	return nums1
}

// StrListUnion string数组并集，去除重复数据
func StrListUnion(list1 []string, list2 []string) []string {
	if len(list1) == 0 {
		return list2
	} else if len(list2) == 0 {
		return list1
	}
	//将数组A转成map
	m := make(map[string]int32)
	for _, v := range list1 {
		m[v] = 0
	}
	//遍历数组B
	for _, v := range list2 {
		//判断B中的元素在A是否存在
		_, ok := m[v]
		if !ok {
			//不存在，直接插入A列表中
			list1 = append(list1, v)
		}
	}
	return list1
}

// 取名XOR异或，实际逻辑是(A-B)U(B-A)的集合b
// https://www.lodashjs.com/docs/lodash.xor
func StrListXOR(a []string, b []string) (ret []string) {
	c := StrListExcept(a, b)
	d := StrListExcept(b, a)
	return StrListUnion(c, d)
}

// Int32ListUnique int32数组去重
func Int32ListUnique(list []int32) []int32 {
	if len(list) == 0 {
		return list
	}
	m := make(map[int32]bool)
	for _, v := range list {
		m[v] = true
	}
	list = list[:0]
	for k := range m {
		list = append(list, k)
	}
	return list
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
	list = list[:0]
	for k := range m {
		list = append(list, k)
	}
	return list
}
