package redisstore

import (
	"errors"
	"github.com/owngoals/go-redis/serializer"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
)

// https://github.com/gin-contrib/cache/blob/master/persistence/redis.go
// https://raw.githubusercontent.com/gin-contrib/cache/master/LICENSE

const (
	DEFAULT = time.Duration(0)
	FOREVER = time.Duration(-1)
)

var (
	ErrCacheMiss  = errors.New("cache: key not found")
	ErrNotStored  = errors.New("cache: not stored")
	ErrNotSupport = errors.New("cache: not support")
)

// RedisStore represents the cache with redis persistence
type RedisStore struct {
	pool              *redis.Pool
	defaultExpiration time.Duration
}

// NewRedisCache returns a RedisStore
// until redigo supports sharding/clustering, only one host will be in hostList
func NewRedisCache(host string, port int, password string, database int, defaultExpiration time.Duration) *RedisStore {
	var pool = &redis.Pool{
		MaxIdle:     5,
		MaxActive:   1000,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			// the redis protocol should probably be made sett-able
			c, err := redis.Dial("tcp", host+":"+strconv.Itoa(port))
			if err != nil {
				return nil, err
			}
			if len(password) > 0 {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			if _, err := c.Do("SELECT", database); err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
		// TestOnBorrow is an optional application supplied function for checking
		// the health of an idle connection before the connection is used again by
		// the application. Argument t is the time that the connection was returned
		// to the pool. If the function returns an error, then the connection is
		// closed.
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if _, err := c.Do("PING"); err != nil {
				return err
			}
			return nil
		},
	}
	return &RedisStore{pool, defaultExpiration}
}

// NewRedisCacheWithPool returns a RedisStore using the provided pool
// until redigo supports sharding/clustering, only one host will be in hostList
func NewRedisCacheWithPool(pool *redis.Pool, defaultExpiration time.Duration) *RedisStore {
	return &RedisStore{pool, defaultExpiration}
}

// Set (see CacheStore interface)
func (c *RedisStore) Set(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()
	return c.invoke(conn.Do, key, value, expires)
}

// Add (see CacheStore interface)
func (c *RedisStore) Add(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()
	if exists(conn, key) {
		return ErrNotStored
	}
	return c.invoke(conn.Do, key, value, expires)
}

// Replace (see CacheStore interface)
func (c *RedisStore) Replace(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()
	if !exists(conn, key) {
		return ErrNotStored
	}
	err := c.invoke(conn.Do, key, value, expires)
	if value == nil {
		return ErrNotStored
	}

	return err

}

// Get (see CacheStore interface)
func (c *RedisStore) Get(key string, ptrValue interface{}) error {
	conn := c.pool.Get()
	defer conn.Close()
	raw, err := conn.Do("GET", key)
	if raw == nil {
		return ErrCacheMiss
	}
	item, err := redis.Bytes(raw, err)
	if err != nil {
		return err
	}
	return serializer.Deserialize(item, ptrValue)
}

func exists(conn redis.Conn, key string) bool {
	retval, _ := redis.Bool(conn.Do("EXISTS", key))
	return retval
}

func (c *RedisStore) Exists(key string) bool {
	conn := c.pool.Get()
	defer conn.Close()
	b, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return b
}

func (c *RedisStore) SetExpire(key string, expires time.Duration) bool {
	conn := c.pool.Get()
	defer conn.Close()
	b, err := redis.Bool(conn.Do("EXPIRE", key, int32(expires/time.Second)))
	if err != nil {
		return false
	}
	return b
}

// Delete (see CacheStore interface)
func (c *RedisStore) Delete(key string) error {
	conn := c.pool.Get()
	defer conn.Close()
	if !exists(conn, key) {
		return ErrCacheMiss
	}
	_, err := conn.Do("DEL", key)
	return err
}

// Increment (see CacheStore interface)
func (c *RedisStore) Increment(key string, delta uint64) (uint64, error) {
	conn := c.pool.Get()
	defer conn.Close()
	// Check for existance *before* increment as per the cache contract.
	// redis will auto create the key, and we don't want that. Since we need to do increment
	// ourselves instead of natively via INCRBY (redis doesn't support wrapping), we get the value
	// and do the exists check this way to minimize calls to Redis
	val, err := conn.Do("GET", key)
	if val == nil {
		return 0, ErrCacheMiss
	}
	if err == nil {
		currentVal, err := redis.Int64(val, nil)
		if err != nil {
			return 0, err
		}
		sum := currentVal + int64(delta)
		_, err = conn.Do("SET", key, sum)
		if err != nil {
			return 0, err
		}
		return uint64(sum), nil
	}

	return 0, err
}

// Decrement (see CacheStore interface)
func (c *RedisStore) Decrement(key string, delta uint64) (newValue uint64, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	// Check for existance *before* increment as per the cache contract.
	// redis will auto create the key, and we don't want that, hence the exists call
	if !exists(conn, key) {
		return 0, ErrCacheMiss
	}
	// Decrement contract says you can only go to 0
	// so we go fetch the value and if the delta is greater than the amount,
	// 0 out the value
	currentVal, err := redis.Int64(conn.Do("GET", key))
	if err == nil && delta > uint64(currentVal) {
		tempint, err := redis.Int64(conn.Do("DECRBY", key, currentVal))
		return uint64(tempint), err
	}
	tempint, err := redis.Int64(conn.Do("DECRBY", key, delta))
	return uint64(tempint), err
}

// Flush (see CacheStore interface)
func (c *RedisStore) Flush() error {
	conn := c.pool.Get()
	defer conn.Close()
	// 這裏修改為 flushdb
	_, err := conn.Do("FLUSHDB")
	return err
}

func (c *RedisStore) invoke(f func(string, ...interface{}) (interface{}, error),
	key string, value interface{}, expires time.Duration) error {

	switch expires {
	case DEFAULT:
		expires = c.defaultExpiration
	case FOREVER:
		expires = time.Duration(0)
	}

	b, err := serializer.Serialize(value)
	if err != nil {
		return err
	}

	if expires > 0 {
		_, err := f("SETEX", key, int32(expires/time.Second), b)
		return err
	}

	_, err = f("SET", key, b)
	return err

}
