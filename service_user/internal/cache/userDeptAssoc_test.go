package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zhufuyi/sponge/pkg/gotest"
	"github.com/zhufuyi/sponge/pkg/utils"

	"gogogo/service_user/internal/model"
)

func newUserDeptAssocCache() *gotest.Cache {
	record1 := &model.UserDeptAssoc{}
	record1.ID = 1
	record2 := &model.UserDeptAssoc{}
	record2.ID = 2
	testData := map[string]interface{}{
		utils.Uint64ToStr(record1.ID): record1,
		utils.Uint64ToStr(record2.ID): record2,
	}

	c := gotest.NewCache(testData)
	c.ICache = NewUserDeptAssocCache(&model.CacheType{
		CType: "redis",
		Rdb:   c.RedisClient,
	})
	return c
}

func Test_userDeptAssocCache_Set(t *testing.T) {
	c := newUserDeptAssocCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserDeptAssoc)
	err := c.ICache.(UserDeptAssocCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// nil data
	err = c.ICache.(UserDeptAssocCache).Set(c.Ctx, 0, nil, time.Hour)
	assert.NoError(t, err)
}

func Test_userDeptAssocCache_Get(t *testing.T) {
	c := newUserDeptAssocCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserDeptAssoc)
	err := c.ICache.(UserDeptAssocCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(UserDeptAssocCache).Get(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, record, got)

	// zero key error
	_, err = c.ICache.(UserDeptAssocCache).Get(c.Ctx, 0)
	assert.Error(t, err)
}

func Test_userDeptAssocCache_MultiGet(t *testing.T) {
	c := newUserDeptAssocCache()
	defer c.Close()

	var testData []*model.UserDeptAssoc
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.UserDeptAssoc))
	}

	err := c.ICache.(UserDeptAssocCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(UserDeptAssocCache).MultiGet(c.Ctx, c.GetIDs())
	if err != nil {
		t.Fatal(err)
	}

	expected := c.GetTestData()
	for k, v := range expected {
		assert.Equal(t, got[utils.StrToUint64(k)], v.(*model.UserDeptAssoc))
	}
}

func Test_userDeptAssocCache_MultiSet(t *testing.T) {
	c := newUserDeptAssocCache()
	defer c.Close()

	var testData []*model.UserDeptAssoc
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.UserDeptAssoc))
	}

	err := c.ICache.(UserDeptAssocCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_userDeptAssocCache_Del(t *testing.T) {
	c := newUserDeptAssocCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserDeptAssoc)
	err := c.ICache.(UserDeptAssocCache).Del(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_userDeptAssocCache_SetCacheWithNotFound(t *testing.T) {
	c := newUserDeptAssocCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserDeptAssoc)
	err := c.ICache.(UserDeptAssocCache).SetCacheWithNotFound(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewUserDeptAssocCache(t *testing.T) {
	c := NewUserDeptAssocCache(&model.CacheType{
		CType: "",
	})
	assert.Nil(t, c)
	c = NewUserDeptAssocCache(&model.CacheType{
		CType: "memory",
	})
	assert.NotNil(t, c)
	c = NewUserDeptAssocCache(&model.CacheType{
		CType: "redis",
	})
	assert.NotNil(t, c)
}
