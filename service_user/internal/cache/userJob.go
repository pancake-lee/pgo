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
	userJobCachePrefixKey = "userJob:"
	// UserJobExpireTime expire time
	UserJobExpireTime = 5 * time.Minute
)

var _ UserJobCache = (*userJobCache)(nil)

// UserJobCache cache interface
type UserJobCache interface {
	Set(ctx context.Context, id uint64, data *model.UserJob, duration time.Duration) error
	Get(ctx context.Context, id uint64) (*model.UserJob, error)
	MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.UserJob, error)
	MultiSet(ctx context.Context, data []*model.UserJob, duration time.Duration) error
	Del(ctx context.Context, id uint64) error
	SetCacheWithNotFound(ctx context.Context, id uint64) error
}

// userJobCache define a cache struct
type userJobCache struct {
	cache cache.Cache
}

// NewUserJobCache new a cache
func NewUserJobCache(cacheType *model.CacheType) UserJobCache {
	jsonEncoding := encoding.JSONEncoding{}
	cachePrefix := ""

	cType := strings.ToLower(cacheType.CType)
	switch cType {
	case "redis":
		c := cache.NewRedisCache(cacheType.Rdb, cachePrefix, jsonEncoding, func() interface{} {
			return &model.UserJob{}
		})
		return &userJobCache{cache: c}
	case "memory":
		c := cache.NewMemoryCache(cachePrefix, jsonEncoding, func() interface{} {
			return &model.UserJob{}
		})
		return &userJobCache{cache: c}
	}

	return nil // no cache
}

// GetUserJobCacheKey cache key
func (c *userJobCache) GetUserJobCacheKey(id uint64) string {
	return userJobCachePrefixKey + utils.Uint64ToStr(id)
}

// Set write to cache
func (c *userJobCache) Set(ctx context.Context, id uint64, data *model.UserJob, duration time.Duration) error {
	if data == nil || id == 0 {
		return nil
	}
	cacheKey := c.GetUserJobCacheKey(id)
	err := c.cache.Set(ctx, cacheKey, data, duration)
	if err != nil {
		return err
	}
	return nil
}

// Get cache value
func (c *userJobCache) Get(ctx context.Context, id uint64) (*model.UserJob, error) {
	var data *model.UserJob
	cacheKey := c.GetUserJobCacheKey(id)
	err := c.cache.Get(ctx, cacheKey, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MultiSet multiple set cache
func (c *userJobCache) MultiSet(ctx context.Context, data []*model.UserJob, duration time.Duration) error {
	valMap := make(map[string]interface{})
	for _, v := range data {
		cacheKey := c.GetUserJobCacheKey(v.ID)
		valMap[cacheKey] = v
	}

	err := c.cache.MultiSet(ctx, valMap, duration)
	if err != nil {
		return err
	}

	return nil
}

// MultiGet multiple get cache, return key in map is id value
func (c *userJobCache) MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.UserJob, error) {
	var keys []string
	for _, v := range ids {
		cacheKey := c.GetUserJobCacheKey(v)
		keys = append(keys, cacheKey)
	}

	itemMap := make(map[string]*model.UserJob)
	err := c.cache.MultiGet(ctx, keys, itemMap)
	if err != nil {
		return nil, err
	}

	retMap := make(map[uint64]*model.UserJob)
	for _, id := range ids {
		val, ok := itemMap[c.GetUserJobCacheKey(id)]
		if ok {
			retMap[id] = val
		}
	}

	return retMap, nil
}

// Del delete cache
func (c *userJobCache) Del(ctx context.Context, id uint64) error {
	cacheKey := c.GetUserJobCacheKey(id)
	err := c.cache.Del(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// SetCacheWithNotFound set empty cache
func (c *userJobCache) SetCacheWithNotFound(ctx context.Context, id uint64) error {
	cacheKey := c.GetUserJobCacheKey(id)
	err := c.cache.SetCacheWithNotFound(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}
