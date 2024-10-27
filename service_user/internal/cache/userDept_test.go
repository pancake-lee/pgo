package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zhufuyi/sponge/pkg/gotest"
	"github.com/zhufuyi/sponge/pkg/utils"

	"gogogo/service_user/internal/model"
)

func newUserDeptCache() *gotest.Cache {
	record1 := &model.UserDept{}
	record1.ID = 1
	record2 := &model.UserDept{}
	record2.ID = 2
	testData := map[string]interface{}{
		utils.Uint64ToStr(record1.ID): record1,
		utils.Uint64ToStr(record2.ID): record2,
	}

	c := gotest.NewCache(testData)
	c.ICache = NewUserDeptCache(&model.CacheType{
		CType: "redis",
		Rdb:   c.RedisClient,
	})
	return c
}

func Test_userDeptCache_Set(t *testing.T) {
	c := newUserDeptCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserDept)
	err := c.ICache.(UserDeptCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// nil data
	err = c.ICache.(UserDeptCache).Set(c.Ctx, 0, nil, time.Hour)
	assert.NoError(t, err)
}

func Test_userDeptCache_Get(t *testing.T) {
	c := newUserDeptCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserDept)
	err := c.ICache.(UserDeptCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(UserDeptCache).Get(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, record, got)

	// zero key error
	_, err = c.ICache.(UserDeptCache).Get(c.Ctx, 0)
	assert.Error(t, err)
}

func Test_userDeptCache_MultiGet(t *testing.T) {
	c := newUserDeptCache()
	defer c.Close()

	var testData []*model.UserDept
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.UserDept))
	}

	err := c.ICache.(UserDeptCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(UserDeptCache).MultiGet(c.Ctx, c.GetIDs())
	if err != nil {
		t.Fatal(err)
	}

	expected := c.GetTestData()
	for k, v := range expected {
		assert.Equal(t, got[utils.StrToUint64(k)], v.(*model.UserDept))
	}
}

func Test_userDeptCache_MultiSet(t *testing.T) {
	c := newUserDeptCache()
	defer c.Close()

	var testData []*model.UserDept
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.UserDept))
	}

	err := c.ICache.(UserDeptCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_userDeptCache_Del(t *testing.T) {
	c := newUserDeptCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserDept)
	err := c.ICache.(UserDeptCache).Del(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_userDeptCache_SetCacheWithNotFound(t *testing.T) {
	c := newUserDeptCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserDept)
	err := c.ICache.(UserDeptCache).SetCacheWithNotFound(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewUserDeptCache(t *testing.T) {
	c := NewUserDeptCache(&model.CacheType{
		CType: "",
	})
	assert.Nil(t, c)
	c = NewUserDeptCache(&model.CacheType{
		CType: "memory",
	})
	assert.NotNil(t, c)
	c = NewUserDeptCache(&model.CacheType{
		CType: "redis",
	})
	assert.NotNil(t, c)
}
