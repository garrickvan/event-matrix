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
	"container/heap"
	"sync"
	"time"
)

// Cache 是一个LRU 缓存，使用堆来存储缓存项， 对比测试用
type Cache struct {
	data         map[string]interface{}
	accessTime   map[string]time.Time
	hp           *ItemHeap
	stopChan     chan struct{}
	maxCacheSize int
	interval     time.Duration
	mu           sync.Mutex // 新增互斥锁，用于保证并发安全
}

// 缓存条目的结构体
type Item struct {
	key      string
	accessed time.Time
	index    int
}

// ItemHeap 是一个基于访问时间的最小堆
type ItemHeap []*Item

// 为 ItemHeap 实现 heap.Interface 接口
func (h ItemHeap) Len() int           { return len(h) }
func (h ItemHeap) Less(i, j int) bool { return h[i].accessed.Before(h[j].accessed) }
func (h ItemHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// Push 向堆中插入一个元素
func (h *ItemHeap) Push(x interface{}) {
	item := x.(*Item)
	*h = append(*h, item)
}

// Pop 从堆中删除一个元素
func (h *ItemHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// NewCache 创建一个新的缓存
func NewCache(maxCacheSize int, interval time.Duration) *Cache {
	hp := &ItemHeap{}
	heap.Init(hp)

	// 最小间隔为1秒
	if interval <= 1*time.Second {
		interval = 1 * time.Second
	}

	cache := &Cache{
		data:         make(map[string]interface{}),
		accessTime:   make(map[string]time.Time),
		hp:           hp,
		stopChan:     make(chan struct{}),
		maxCacheSize: maxCacheSize,
		interval:     interval,
		mu:           sync.Mutex{},
	}
	// 启动定期清理
	go cache.startEvictionProcess()
	return cache
}

// Set 向缓存中添加条目
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 如果键已存在，更新值并更新访问时间
	if _, exists := c.data[key]; exists {
		c.data[key] = value
		c.accessTime[key] = time.Now()
		for i := 0; i < len(*c.hp); i++ {
			if (*c.hp)[i].key == key {
				(*c.hp)[i].accessed = time.Now()
				heap.Fix(c.hp, i)
				return
			}
		}
	} else {
		// 如果是新条目，插入缓存和堆
		c.data[key] = value
		c.accessTime[key] = time.Now()
		item := &Item{
			key:      key,
			accessed: time.Now(),
		}
		heap.Push(c.hp, item)
	}

	// 如果缓存超过最大大小，进行清理
	if len(c.data) > c.maxCacheSize {
		go c.evictLeastUsed()
	}
}

// Get 从缓存中获取条目
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if value, found := c.data[key]; found {
		// 更新访问时间并调整堆
		c.accessTime[key] = time.Now()
		for i := 0; i < len(*c.hp); i++ {
			if (*c.hp)[i].key == key {
				(*c.hp)[i].accessed = time.Now()
				heap.Fix(c.hp, i)
				return value, true
			}
		}
	}
	return nil, false
}

// 每分钟定期清理最少使用的缓存条目
func (c *Cache) startEvictionProcess() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.evictLeastUsed()
		case <-c.stopChan:
			return
		}
	}
}

// evictLeastUsed 移除最久未使用的缓存条目
func (c *Cache) evictLeastUsed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 弹出堆顶，即最久未使用的缓存条目
	if len(*c.hp) > 0 {
		item := heap.Pop(c.hp).(*Item)
		delete(c.data, item.key)
		delete(c.accessTime, item.key)
	}
}

// Del 删除指定key的缓存条目
func (c *Cache) Del(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 删除数据并更新堆
	if _, exists := c.data[key]; exists {
		delete(c.data, key)
		delete(c.accessTime, key)
		// 找到该条目并移除堆中的元素
		for i := 0; i < len(*c.hp); i++ {
			if (*c.hp)[i].key == key {
				heap.Remove(c.hp, i)
				break
			}
		}
	}
}

// 停止清理协程
func (c *Cache) stop() {
	c.mu.Lock()
	c.stopChan <- struct{}{}
	c.mu.Unlock()
}

func (c *Cache) GetOrHook(key string, hook func() interface{}) (interface{}, bool) {
	var data interface{}
	var ok bool
	if c != nil {
		data, ok = c.Get(key)
	} else {
		return nil, false
	}
	if !ok {
		data = hook()
		if data != nil {
			c.Set(key, data)
			return data, true
		}
	} else {
		return data, true
	}
	return nil, false
}
