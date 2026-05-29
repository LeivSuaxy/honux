package interfaces

import "time"

type CacheRegister struct {
	Value  []byte
	Expire time.Time
}

func (c *CacheRegister) New(value []byte, expire time.Time) *CacheRegister {
	return &CacheRegister{value, expire}
}

func (c *CacheRegister) Expired() bool {
	return time.Now().After(c.Expire)
}

type CacheProvider interface {
	Set(key string, value []byte, expire time.Duration) error
	Get(key string) (*[]byte, error)
	Delete(key string)
	CheckIfExists(key string) bool
	GetKeys() *[]string
}
