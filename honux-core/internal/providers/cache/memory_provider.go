package cache

import (
	"errors"
	"honux-core/internal/interfaces"
	"sync"
	"time"
)

type MemoryCache struct {
	cache map[string]*interfaces.CacheRegister
	mu    sync.RWMutex
}

var (
	once        sync.Once
	memoryCache *MemoryCache
)

func GetCache() *MemoryCache {
	once.Do(func() {
		memoryCache = loadMemoryCache()
	})
	return memoryCache
}

func loadMemoryCache() *MemoryCache {
	return &MemoryCache{
		cache: make(map[string]*interfaces.CacheRegister),
	}
}

func (c *MemoryCache) Set(key string, value []byte, expire time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	expireAt := time.Now().Add(expire)
	c.cache[key] = &interfaces.CacheRegister{Value: value, Expire: expireAt}
	keys := make([]string, 0, len(c.cache))
	for k := range c.cache {
		keys = append(keys, k)
	}
	return nil
}

func (c *MemoryCache) Get(key string) (*[]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.cache))
	for k := range c.cache {
		keys = append(keys, k)
	}
	v, ok := c.cache[key]
	if !ok {
		return nil, errors.New("key not found")
	}
	if v.Expired() {
		c.mu.RUnlock()
		c.Delete(key)
		c.mu.RLock()
		return nil, errors.New("key expired")
	}
	return new(v.Value), nil
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
}

func (c *MemoryCache) CheckIfExists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.cache[key]
	return ok
}

func (c *MemoryCache) GetKeys() *[]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var keys = make([]string, 0)

	for k := range c.cache {
		keys = append(keys, k)
	}

	return &keys
}
