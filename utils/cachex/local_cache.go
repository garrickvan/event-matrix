// Copyright 2025 eventmatrix.cn
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cachex

import (
	"time"

	"github.com/dgraph-io/ristretto"
)

// LocalCache 基于ristretto实现的本地缓存
type LocalCache struct {
	cache      *ristretto.Cache
	defaultTTL time.Duration
}

// InitCache 初始化本地缓存
// 参数:
//
//	maxMen: 最大内存限制(字节)
//	defaultTimeout: 默认缓存过期时间(秒)
//
// 返回:
//
//	error: 初始化错误
func (lc *LocalCache) InitCache(maxMen int64, defaultTimeout int) error {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters:        maxMen / 10, // number of keys to track frequency of (10M).
		MaxCost:            maxMen,      // 50 * (1 << 20) maximum cost of cache (50 M).
		BufferItems:        64,          // number of keys per Get buffer.
		IgnoreInternalCost: false,
	})
	if err != nil {
		return err
	}
	lc.cache = cache
	lc.defaultTTL = time.Duration(defaultTimeout) * time.Second
	return nil
}

// Get 获取缓存值
// 参数:
//
//	key: 缓存键
//
// 返回:
//
//	interface{}: 缓存值
//	bool: 是否命中缓存
func (lc *LocalCache) Get(key string) (interface{}, bool) {
	if lc.cache != nil {
		return lc.cache.Get(key)
	}
	return nil, false
}

// GetOrHook 获取缓存值，若不存在则调用hook函数获取并缓存
// 参数:
//
//	key: 缓存键
//	hook: 获取数据的回调函数
//
// 返回:
//
//	interface{}: 缓存值或hook返回值
//	bool: 是否成功获取值
func (lc *LocalCache) GetOrHook(key string, hook func() interface{}) (interface{}, bool) {
	if lc.cache == nil {
		return nil, false
	}
	if data, exists := lc.cache.Get(key); exists && data != nil {
		return data, true
	}
	data := hook()
	if data == nil {
		return nil, false
	}
	lc.cache.SetWithTTL(key, data, 0, lc.defaultTTL)
	return data, true
}

// Put 设置缓存值(使用默认TTL)
// 参数:
//
//	key: 缓存键
//	value: 缓存值
//
// 返回:
//
//	bool: 是否设置成功
func (lc *LocalCache) Put(key string, value interface{}) bool {
	if lc.cache != nil {
		return lc.cache.SetWithTTL(key, value, 0, lc.defaultTTL)
	}
	return false
}

// PutWithTTL 设置缓存值(自定义TTL)
// 参数:
//
//	key: 缓存键
//	value: 缓存值
//	ttl: 过期时间
//
// 返回:
//
//	bool: 是否设置成功
func (lc *LocalCache) PutWithTTL(key string, value interface{}, ttl time.Duration) bool {
	if lc.cache != nil {
		return lc.cache.SetWithTTL(key, value, 0, ttl)
	}
	return false
}

// PutPermanent 设置永久缓存值(无过期时间)
// 参数:
//
//	key: 缓存键
//	value: 缓存值
//
// 返回:
//
//	bool: 是否设置成功
func (lc *LocalCache) PutPermanent(key string, value interface{}) bool {
	if lc.cache != nil {
		return lc.cache.Set(key, value, 0)
	}
	return false
}

// Del 删除缓存值
// 参数:
//
//	key: 缓存键
func (lc *LocalCache) Del(key string) {
	if lc.cache != nil {
		lc.cache.Del(key)
	}
}

// Flush 清空所有缓存
func (lc *LocalCache) Flush() {
	if lc.cache != nil {
		lc.cache.Clear()
	}
}

// GetCacheInstance 获取底层ristretto缓存实例
// 返回:
//
//	*ristretto.Cache: 底层缓存实例
func (lc *LocalCache) GetCacheInstance() *ristretto.Cache {
	return lc.cache
}
