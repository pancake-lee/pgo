package putil

import (
	"slices"
	"testing"
)

func TestAnyToUrlQuery(t *testing.T) {
	var testCases struct {
		A  string
		B  int32
		C  int64
		Al []string
		Bl []int32
		Cl []int64
	}

	testCases.A = "test"
	testCases.B = 123
	testCases.C = 456789
	testCases.Al = []string{"one", "two", "three"}
	testCases.Bl = []int32{1, 2, 3}
	testCases.Cl = []int64{100, 200, 300}

	querys := GetUrlQueryString(testCases)
	if querys == nil {
		t.Fatal("GetUrlQueryString returned nil")
	}

	if querys["A"] != "test" ||
		querys["B"] != "123" ||
		querys["C"] != "456789" ||
		querys["Al"] != "one,two,three" ||
		querys["Bl"] != "1,2,3" ||
		querys["Cl"] != "100,200,300" {
		t.Fatalf("GetUrlQueryString returned unexpected values: %v", querys)
	}
	t.Logf("GetUrlQueryString returned: %v", querys)
}

func TestCollections(t *testing.T) {
	l1 := []int32{1, 2, 3, 4, 5, 5}
	l2 := []int32{3, 4, 5, 6, 7}
	sl1 := []string{"a", "b", "c", "d", "e", "e"}
	sl2 := []string{"c", "d", "e", "f", "g"}

	// 测试交集
	{
		result := Int32ListIntersect(l1, l2)
		expected := []int32{3, 4, 5}
		if !slices.Equal(result, expected) {
			t.Fatalf("Expected %v, got %v", expected, result)
		}
	}

	{
		result := StrListIntersect(sl1, sl2)
		expected := []string{"c", "d", "e"}
		if !slices.Equal(result, expected) {
			t.Fatalf("Expected %v, got %v", expected, result)
		}
	}

	// 测试差集
	{
		result := Int32ListExcept(l1, l2)
		expected := []int32{1, 2}
		if !slices.Equal(result, expected) {
			t.Fatalf("Expected %v, got %v", expected, result)
		}
	}

	{
		result := StrListExcept(sl1, sl2)
		expected := []string{"a", "b"}
		if !slices.Equal(result, expected) {
			t.Fatalf("Expected %v, got %v", expected, result)
		}
	}

	// 测试并集
	{
		result := Int32ListUnion(l1, l2)
		slices.Sort(result) // 用到map的就需要排序才能判定Equal
		expected := []int32{1, 2, 3, 4, 5, 6, 7}
		if !slices.Equal(result, expected) {
			t.Fatalf("Expected %v, got %v", expected, result)
		}
	}

	{
		result := StrListUnion(sl1, sl2)
		slices.Sort(result)
		expected := []string{"a", "b", "c", "d", "e", "f", "g"}
		if !slices.Equal(result, expected) {
			t.Fatalf("Expected %v, got %v", expected, result)
		}
	}

	// 测试异或
	{
		result := StrListXOR(sl1, sl2)
		slices.Sort(result)
		expected := []string{"a", "b", "f", "g"}
		if !slices.Equal(result, expected) {
			t.Fatalf("Expected %v, got %v", expected, result)
		}
	}

	// 测试去重
	{
		result := Int32ListUnique(l1)
		slices.Sort(result)
		expected := []int32{1, 2, 3, 4, 5}
		if !slices.Equal(result, expected) {
			t.Fatalf("Expected %v, got %v", expected, result)
		}

	}

	{
		result := StrListUnique(sl1)
		slices.Sort(result)
		expected := []string{"a", "b", "c", "d", "e"}
		if !slices.Equal(result, expected) {
			t.Fatalf("Expected %v, got %v", expected, result)
		}
	}

	// 测试分步遍历
	{
		var result []int32
		err := WalkSliceByStep(l1, 2, func(s, e int) error {
			result = append(result, l1[s:e]...)
			return nil
		})
		if err != nil {
			t.Fatalf("WalkSliceByStep failed: %v", err)
		}
		if !slices.Equal(result, l1) {
			t.Fatalf("Expected %v, got %v", l1, result)
		}
	}
}
