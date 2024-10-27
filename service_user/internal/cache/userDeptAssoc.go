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
	userDeptAssocCachePrefixKey = "userDeptAssoc:"
	// UserDeptAssocExpireTime expire time
	UserDeptAssocExpireTime = 5 * time.Minute
)

var _ UserDeptAssocCache = (*userDeptAssocCache)(nil)

// UserDeptAssocCache cache interface
type UserDeptAssocCache interface {
	Set(ctx context.Context, id uint64, data *model.UserDeptAssoc, duration time.Duration) error
	Get(ctx context.Context, id uint64) (*model.UserDeptAssoc, error)
	MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.UserDeptAssoc, error)
	MultiSet(ctx context.Context, data []*model.UserDeptAssoc, duration time.Duration) error
	Del(ctx context.Context, id uint64) error
	SetCacheWithNotFound(ctx context.Context, id uint64) error
}

// userDeptAssocCache define a cache struct
type userDeptAssocCache struct {
	cache cache.Cache
}

// NewUserDeptAssocCache new a cache
func NewUserDeptAssocCache(cacheType *model.CacheType) UserDeptAssocCache {
	jsonEncoding := encoding.JSONEncoding{}
	cachePrefix := ""

	cType := strings.ToLower(cacheType.CType)
	switch cType {
	case "redis":
		c := cache.NewRedisCache(cacheType.Rdb, cachePrefix, jsonEncoding, func() interface{} {
			return &model.UserDeptAssoc{}
		})
		return &userDeptAssocCache{cache: c}
	case "memory":
		c := cache.NewMemoryCache(cachePrefix, jsonEncoding, func() interface{} {
			return &model.UserDeptAssoc{}
		})
		return &userDeptAssocCache{cache: c}
	}

	return nil // no cache
}

// GetUserDeptAssocCacheKey cache key
func (c *userDeptAssocCache) GetUserDeptAssocCacheKey(id uint64) string {
	return userDeptAssocCachePrefixKey + utils.Uint64ToStr(id)
}

// Set write to cache
func (c *userDeptAssocCache) Set(ctx context.Context, id uint64, data *model.UserDeptAssoc, duration time.Duration) error {
	if data == nil || id == 0 {
		return nil
	}
	cacheKey := c.GetUserDeptAssocCacheKey(id)
	err := c.cache.Set(ctx, cacheKey, data, duration)
	if err != nil {
		return err
	}
	return nil
}

// Get cache value
func (c *userDeptAssocCache) Get(ctx context.Context, id uint64) (*model.UserDeptAssoc, error) {
	var data *model.UserDeptAssoc
	cacheKey := c.GetUserDeptAssocCacheKey(id)
	err := c.cache.Get(ctx, cacheKey, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MultiSet multiple set cache
func (c *userDeptAssocCache) MultiSet(ctx context.Context, data []*model.UserDeptAssoc, duration time.Duration) error {
	valMap := make(map[string]interface{})
	for _, v := range data {
		cacheKey := c.GetUserDeptAssocCacheKey(v.ID)
		valMap[cacheKey] = v
	}

	err := c.cache.MultiSet(ctx, valMap, duration)
	if err != nil {
		return err
	}

	return nil
}

// MultiGet multiple get cache, return key in map is id value
func (c *userDeptAssocCache) MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.UserDeptAssoc, error) {
	var keys []string
	for _, v := range ids {
		cacheKey := c.GetUserDeptAssocCacheKey(v)
		keys = append(keys, cacheKey)
	}

	itemMap := make(map[string]*model.UserDeptAssoc)
	err := c.cache.MultiGet(ctx, keys, itemMap)
	if err != nil {
		return nil, err
	}

	retMap := make(map[uint64]*model.UserDeptAssoc)
	for _, id := range ids {
		val, ok := itemMap[c.GetUserDeptAssocCacheKey(id)]
		if ok {
			retMap[id] = val
		}
	}

	return retMap, nil
}

// Del delete cache
func (c *userDeptAssocCache) Del(ctx context.Context, id uint64) error {
	cacheKey := c.GetUserDeptAssocCacheKey(id)
	err := c.cache.Del(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// SetCacheWithNotFound set empty cache
func (c *userDeptAssocCache) SetCacheWithNotFound(ctx context.Context, id uint64) error {
	cacheKey := c.GetUserDeptAssocCacheKey(id)
	err := c.cache.SetCacheWithNotFound(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}
