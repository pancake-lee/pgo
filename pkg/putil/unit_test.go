package putil

import "testing"

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
