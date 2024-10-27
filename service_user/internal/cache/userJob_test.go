package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zhufuyi/sponge/pkg/gotest"
	"github.com/zhufuyi/sponge/pkg/utils"

	"gogogo/service_user/internal/model"
)

func newUserJobCache() *gotest.Cache {
	record1 := &model.UserJob{}
	record1.ID = 1
	record2 := &model.UserJob{}
	record2.ID = 2
	testData := map[string]interface{}{
		utils.Uint64ToStr(record1.ID): record1,
		utils.Uint64ToStr(record2.ID): record2,
	}

	c := gotest.NewCache(testData)
	c.ICache = NewUserJobCache(&model.CacheType{
		CType: "redis",
		Rdb:   c.RedisClient,
	})
	return c
}

func Test_userJobCache_Set(t *testing.T) {
	c := newUserJobCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserJob)
	err := c.ICache.(UserJobCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// nil data
	err = c.ICache.(UserJobCache).Set(c.Ctx, 0, nil, time.Hour)
	assert.NoError(t, err)
}

func Test_userJobCache_Get(t *testing.T) {
	c := newUserJobCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserJob)
	err := c.ICache.(UserJobCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(UserJobCache).Get(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, record, got)

	// zero key error
	_, err = c.ICache.(UserJobCache).Get(c.Ctx, 0)
	assert.Error(t, err)
}

func Test_userJobCache_MultiGet(t *testing.T) {
	c := newUserJobCache()
	defer c.Close()

	var testData []*model.UserJob
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.UserJob))
	}

	err := c.ICache.(UserJobCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(UserJobCache).MultiGet(c.Ctx, c.GetIDs())
	if err != nil {
		t.Fatal(err)
	}

	expected := c.GetTestData()
	for k, v := range expected {
		assert.Equal(t, got[utils.StrToUint64(k)], v.(*model.UserJob))
	}
}

func Test_userJobCache_MultiSet(t *testing.T) {
	c := newUserJobCache()
	defer c.Close()

	var testData []*model.UserJob
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.UserJob))
	}

	err := c.ICache.(UserJobCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_userJobCache_Del(t *testing.T) {
	c := newUserJobCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserJob)
	err := c.ICache.(UserJobCache).Del(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_userJobCache_SetCacheWithNotFound(t *testing.T) {
	c := newUserJobCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.UserJob)
	err := c.ICache.(UserJobCache).SetCacheWithNotFound(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewUserJobCache(t *testing.T) {
	c := NewUserJobCache(&model.CacheType{
		CType: "",
	})
	assert.Nil(t, c)
	c = NewUserJobCache(&model.CacheType{
		CType: "memory",
	})
	assert.NotNil(t, c)
	c = NewUserJobCache(&model.CacheType{
		CType: "redis",
	})
	assert.NotNil(t, c)
}
