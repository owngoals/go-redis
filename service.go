package goredis

import (
	"github.com/gomodule/redigo/redis"
	"github.com/owngoals/go-redis/redisstore"
	"time"
)

func NewService(pool *redis.Pool, prefix string) *Service {
	return &Service{
		prefix: prefix,
		store:  redisstore.NewRedisCacheWithPool(pool, redisstore.DEFAULT),
	}
}

type Service struct {
	prefix string
	store  *redisstore.RedisStore
}

func (s *Service) Get(key string, value interface{}) error {
	return s.store.Get(s.cacheKey(key), value)
}

func (s *Service) Set(key string, value interface{}, expire time.Duration) error {
	return s.store.Set(s.cacheKey(key), value, expire)
}

func (s *Service) Add(key string, value interface{}, expire time.Duration) error {
	return s.store.Add(s.cacheKey(key), value, expire)
}

func (s *Service) Replace(key string, data interface{}, expire time.Duration) error {
	return s.store.Replace(s.cacheKey(key), data, expire)
}

func (s *Service) Delete(key string) error {
	return s.store.Delete(s.cacheKey(key))
}

func (s *Service) Increment(key string, data uint64) (uint64, error) {
	return s.store.Increment(s.cacheKey(key), data)
}

func (s *Service) Decrement(key string, data uint64) (uint64, error) {
	return s.store.Decrement(s.cacheKey(key), data)
}

func (s *Service) Flush() error {
	return s.store.Flush()
}

func (s *Service) Exists(key string) bool {
	return s.store.Exists(s.cacheKey(key))
}

func (s *Service) SetExpire(key string, expires time.Duration) bool {
	return s.store.SetExpire(s.cacheKey(key), expires)
}

func (s *Service) cacheKey(key string) string {
	return s.prefix + ":" + key
}
