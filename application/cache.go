package application

import (
	"fmt"
	"time"

	"github.com/allegro/bigcache"
	"github.com/eko/gocache/lib/v4/cache"
	bigcacheStore "github.com/eko/gocache/store/bigcache/v4"
)

func NewCache[T comparable](eviction time.Duration) (*cache.Cache[T], error) {
	cacheClient, err := bigcache.NewBigCache(bigcache.DefaultConfig(eviction))
	if err != nil {
		return nil, fmt.Errorf("failed to create bigcache instance: %w", err)
	}
	store := bigcacheStore.NewBigcache(cacheClient)
	return cache.New[T](store), nil
}

var Cache *cache.Cache[string]

func init() {
	var err error
	if Cache, err = NewCache[string](time.Hour); err != nil {
		panic(fmt.Errorf("failed to create cache: %w", err))
	}
}
