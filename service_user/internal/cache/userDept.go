package cache

import (
	"context"
	"strings"
	"time"

	"github.com/zhufuyi/sponge/pkg/cache"
	"github.com/zhufuyi/sponge/pkg/encoding"
	"github.com/zhufuyi/sponge/pkg/utils"

	"gogogo/service_user/internal/model"
)

const (
	// cache prefix key, must end with a colon
	userDeptCachePrefixKey = "userDept:"
	// UserDeptExpireTime expire time
	UserDeptExpireTime = 5 * time.Minute
)

var _ UserDeptCache = (*userDeptCache)(nil)

// UserDeptCache cache interface
type UserDeptCache interface {
	Set(ctx context.Context, id uint64, data *model.UserDept, duration time.Duration) error
	Get(ctx context.Context, id uint64) (*model.UserDept, error)
	MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.UserDept, error)
	MultiSet(ctx context.Context, data []*model.UserDept, duration time.Duration) error
	Del(ctx context.Context, id uint64) error
	SetCacheWithNotFound(ctx context.Context, id uint64) error
}

// userDeptCache define a cache struct
type userDeptCache struct {
	cache cache.Cache
}

// NewUserDeptCache new a cache
func NewUserDeptCache(cacheType *model.CacheType) UserDeptCache {
	jsonEncoding := encoding.JSONEncoding{}
	cachePrefix := ""

	cType := strings.ToLower(cacheType.CType)
	switch cType {
	case "redis":
		c := cache.NewRedisCache(cacheType.Rdb, cachePrefix, jsonEncoding, func() interface{} {
			return &model.UserDept{}
		})
		return &userDeptCache{cache: c}
	case "memory":
		c := cache.NewMemoryCache(cachePrefix, jsonEncoding, func() interface{} {
			return &model.UserDept{}
		})
		return &userDeptCache{cache: c}
	}

	return nil // no cache
}

// GetUserDeptCacheKey cache key
func (c *userDeptCache) GetUserDeptCacheKey(id uint64) string {
	return userDeptCachePrefixKey + utils.Uint64ToStr(id)
}

// Set write to cache
func (c *userDeptCache) Set(ctx context.Context, id uint64, data *model.UserDept, duration time.Duration) error {
	if data == nil || id == 0 {
		return nil
	}
	cacheKey := c.GetUserDeptCacheKey(id)
	err := c.cache.Set(ctx, cacheKey, data, duration)
	if err != nil {
		return err
	}
	return nil
}

// Get cache value
func (c *userDeptCache) Get(ctx context.Context, id uint64) (*model.UserDept, error) {
	var data *model.UserDept
	cacheKey := c.GetUserDeptCacheKey(id)
	err := c.cache.Get(ctx, cacheKey, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MultiSet multiple set cache
func (c *userDeptCache) MultiSet(ctx context.Context, data []*model.UserDept, duration time.Duration) error {
	valMap := make(map[string]interface{})
	for _, v := range data {
		cacheKey := c.GetUserDeptCacheKey(v.ID)
		valMap[cacheKey] = v
	}

	err := c.cache.MultiSet(ctx, valMap, duration)
	if err != nil {
		return err
	}

	return nil
}

// MultiGet multiple get cache, return key in map is id value
func (c *userDeptCache) MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.UserDept, error) {
	var keys []string
	for _, v := range ids {
		cacheKey := c.GetUserDeptCacheKey(v)
		keys = append(keys, cacheKey)
	}

	itemMap := make(map[string]*model.UserDept)
	err := c.cache.MultiGet(ctx, keys, itemMap)
	if err != nil {
		return nil, err
	}

	retMap := make(map[uint64]*model.UserDept)
	for _, id := range ids {
		val, ok := itemMap[c.GetUserDeptCacheKey(id)]
		if ok {
			retMap[id] = val
		}
	}

	return retMap, nil
}

// Del delete cache
func (c *userDeptCache) Del(ctx context.Context, id uint64) error {
	cacheKey := c.GetUserDeptCacheKey(id)
	err := c.cache.Del(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// SetCacheWithNotFound set empty cache
func (c *userDeptCache) SetCacheWithNotFound(ctx context.Context, id uint64) error {
	cacheKey := c.GetUserDeptCacheKey(id)
	err := c.cache.SetCacheWithNotFound(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}
