package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func CacheKey(key string, args ...interface{}) string {
	var result string = fmt.Sprintf("%v", key)
	if len(args) == 0 {
		return result
	}
	for _, arg := range args {
		result += fmt.Sprintf(":%v", arg)
	}
	return result
}

type ICacheStore interface {
	Get(key string) ([]byte, error)
	Set(key string, val []byte, exp time.Duration) error
}

func Remember[T any](store ICacheStore, key string, duration time.Duration, fn func() (T, error)) (T, error) {
	cached, err := store.Get(key)
	if err == nil && cached != nil && len(cached) > 0 {
		var result T
		if err := json.Unmarshal(cached, &result); err != nil {
			return result, err
		}
		return result, nil
	}

	result, err := fn()
	if err != nil {
		return result, err
	}

	computed, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	store.Set(key, computed, duration)
	return result, nil
}

func NewCacheStore() *CacheStore {
	return &CacheStore{
		kV:       make(map[string][]byte),
		expiries: make(map[string]time.Time),
	}
}

type CacheStore struct {
	kV       map[string][]byte
	expiries map[string]time.Time
}

func (c *CacheStore) Get(key string) ([]byte, error) {
	val, ok := c.kV[key]
	if !ok {
		return nil, errors.New("key not found")
	}

	expiry, exists := c.expiries[key]
	if exists && time.Now().After(expiry) {
		c.Delete(key)
		return nil, errors.New("key expired")
	}

	return val, nil
}

func (c *CacheStore) Set(key string, val []byte, exp time.Duration) error {
	c.kV[key] = val
	if exp > 0 {
		c.expiries[key] = time.Now().Add(exp)
	} else {
		delete(c.expiries, key)
	}
	return nil
}

func (c *CacheStore) Delete(key string) error {
	delete(c.kV, key)
	delete(c.expiries, key)
	return nil
}
