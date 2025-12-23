package cache

import (
	"time"

	"github.com/allegro/bigcache"
	"github.com/bytedance/sonic"
	"golang.org/x/sync/singleflight"
)

type Cache[T any] struct {
	bc *bigcache.BigCache
	sf singleflight.Group
}

const (
	shardCount = 64
)

func New[T any](expiry time.Duration) (*Cache[T], error) {
	bc, err := bigcache.NewBigCache(bigcache.Config{
		Shards:      shardCount,
		LifeWindow:  expiry,
		CleanWindow: expiry / 10,
	})
	if err != nil {
		return nil, err
	}

	return &Cache[T]{bc: bc}, nil
}

func (c *Cache[T]) Set(key string, value T) error {
	b, err := sonic.ConfigFastest.Marshal(value)
	if err != nil {
		return err
	}
	return c.bc.Set(key, b)
}

func (c *Cache[T]) Get(key string) (T, bool) {
	var zero T

	b, err := c.bc.Get(key)
	if err != nil {
		return zero, false
	}

	if err := sonic.ConfigFastest.Unmarshal(b, &zero); err != nil {
		return zero, false
	}

	return zero, true
}

func (c *Cache[T]) GetOrSet(key string, loader func() (T, error)) (T, error) {
	var zero T

	if val, ok := c.Get(key); ok {
		return val, nil
	}

	val, err := loader()
	if err != nil {
		return zero, err
	}

	if err = c.Set(key, val); err != nil {
		return zero, err
	}

	return val, nil
}

func (c *Cache[T]) Delete(key string) error {
	return c.bc.Delete(key)
}

func (c *Cache[T]) Reset() error {
	return c.bc.Reset()
}
