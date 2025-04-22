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
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"github.com/dgraph-io/ristretto"
)

type Event struct {
	ID          string `json:"id" gorm:"primaryKey"`
	Project     string `json:"project" gorm:"index"`
	Version     string `json:"version"`
	Context     string `json:"context"`
	Entity      string `json:"entity"`
	Event       string `json:"event"`
	Source      string `json:"source"`
	Params      string `json:"params"`
	AccessToken string `json:"accessToken"`
	CreatedAt   int64  `json:"createdAt"`
	Sign        string `json:"sign"`
	raw         string `json:"-"`
}

func TestRistrettoCacheMemory(t *testing.T) {
	// 测量内存使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	// 创建 Ristretto 缓存
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters:        1e9,       // 保持较大的计数器数量以提高命中率
		MaxCost:            104857600, // 设置最大缓存容量为 100 MB
		BufferItems:        64,        // 批量处理项的队列大小
		IgnoreInternalCost: true,      // 忽略内部缓存的成本
	})
	if err != nil {
		t.Fatalf("failed to create  cache: %v", err)
	}

	before := memStats.Alloc
	// 创建 N 万个 Event 对象并添加到缓存
	numEvents := 200 * 10000
	events := make([]*Event, numEvents)
	for i := 0; i < numEvents; i++ {
		events[i] = &Event{
			ID:          fmt.Sprintf("id-%d", i),
			Project:     "project",
			Version:     "v1.0.0",
			Context:     "context",
			Entity:      "entity",
			Event:       "event",
			Source:      "source",
			Params:      "params",
			AccessToken: "accessToken",
			CreatedAt:   1234567890,
			Sign:        "sign",
		}
		cache.Set(fmt.Sprintf("key-%d", i), events[i], 1) // 成本设为 1
	}

	// 强制等待所有写入完成
	cache.Wait()
	start := time.Now()
	// 校验缓存内容
	missingCount := 0
	for i := 0; i < numEvents; i++ {
		if _, found := cache.Get(fmt.Sprintf("key-%d", i)); !found {
			missingCount++
			// t.Errorf("cache missing key-%d", i)
		}
	}

	elapsed := time.Since(start)

	fmt.Printf("Time taken to validate cache: %v\n\n", elapsed)

	runtime.ReadMemStats(&memStats)
	used := memStats.Alloc - before
	fmt.Printf("Used memory: %.2f MB\n", float64(used)/(1024*1024))
	fmt.Printf("Total allocated memory: %.2f MB\n", float64(memStats.TotalAlloc)/(1024*1024))
	fmt.Printf("Heap memory: %.2f MB\n", float64(memStats.HeapAlloc)/(1024*1024))
	fmt.Printf("Number of missing keys: %d\n", missingCount)

}

func TestBigCacheMemory(t *testing.T) {
	// 创建 BigCache 缓存
	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(24 * 3600)) // 默认缓存过期时间为24小时
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	// 创建 10 万个 Event 对象并添加到缓存
	numEvents := 00 * 10000
	events := make([]*Event, numEvents)
	for i := 0; i < numEvents; i++ {
		events[i] = &Event{
			ID:          fmt.Sprintf("id-%d", i),
			Project:     "project",
			Version:     "v1.0.0",
			Context:     "context",
			Entity:      "entity",
			Event:       "event",
			Source:      "source",
			Params:      "params",
			AccessToken: "accessToken",
			CreatedAt:   1234567890,
			Sign:        "sign",
		}
		// 使用缓存的 Set 方法添加数据
		cache.Set(fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf("%v", events[i]))) // BigCache 不支持直接存储对象，需转换为字节数组
	}

	// 校验缓存内容
	missingCount := 0
	for i := 0; i < numEvents; i++ {
		if _, err := cache.Get(fmt.Sprintf("key-%d", i)); err != nil {
			missingCount++
			// t.Errorf("cache missing key-%d", i)
		}
	}

	// 测量内存使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Printf("Allocated memory: %.2f MB\n", float64(memStats.Alloc)/(1024*1024))
	fmt.Printf("Total allocated memory: %.2f MB\n", float64(memStats.TotalAlloc)/(1024*1024))
	fmt.Printf("Heap memory: %.2f MB\n", float64(memStats.HeapAlloc)/(1024*1024))
	fmt.Printf("Number of missing keys: %d\n", missingCount)
}

func TestFreecacheMemory(t *testing.T) {
	var memStats runtime.MemStats
	// 创建 Freecache 缓存，最大缓存大小为 100 MB
	cacheSize := 100 * 1024 * 1024 // 100 MB
	cache := freecache.NewCache(cacheSize)
	// 测量内存使用情况
	runtime.ReadMemStats(&memStats)
	before := memStats.Alloc

	// 创建 10 万个 Event 对象并添加到缓存
	numEvents := 200 * 10000
	events := make([]*Event, numEvents)
	for i := 0; i < numEvents; i++ {
		events[i] = &Event{
			ID:          fmt.Sprintf("id-%d", i),
			Project:     "project",
			Version:     "v1.0.0",
			Context:     "context",
			Entity:      "entity",
			Event:       "event",
			Source:      "source",
			Params:      "params",
			AccessToken: "accessToken",
			CreatedAt:   1234567890,
			Sign:        "sign",
		}

		// 将 Event 对象序列化为字节切片，并添加到 Freecache 缓存
		eventData := []byte(fmt.Sprintf("%v", events[i]))         // 简单的序列化为字节切片，实际中可以使用 JSON 或其他方式
		cache.Set([]byte(fmt.Sprintf("key-%d", i)), eventData, 0) // 0 表示没有过期时间
	}

	// cache.Clear()

	// 校验缓存内容
	missingCount := 0
	for i := 0; i < numEvents; i++ {
		_, err := cache.Get([]byte(fmt.Sprintf("key-%d", i)))
		if err != nil {
			missingCount++
			// t.Errorf("cache missing key-%d", i)
		}
	}

	runtime.ReadMemStats(&memStats)
	usedMemory := memStats.Alloc - before
	fmt.Printf("Used memory: %.2f MB\n", float64(usedMemory)/(1024*1024))
	fmt.Printf("Total allocated memory: %.2f MB\n", float64(memStats.TotalAlloc)/(1024*1024))
	fmt.Printf("Heap memory: %.2f MB\n", float64(memStats.HeapAlloc)/(1024*1024))
	fmt.Printf("Number of missing keys: %d\n", missingCount)
}

func TestMapCacheMemory(t *testing.T) {
	// 创建 MapCache 缓存
	cache := NewCache(2, 2*time.Minute) // 设置最大缓存大小为3

	// 设置一些缓存
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// 模拟一些访问
	time.Sleep(2 * time.Second)
	cache.Get("key1") // 访问 key1

	// 设置一些新的缓存
	cache.Set("key4", "value4")
	cache.Set("key5", "value5")

	// 等待一段时间，让缓存条目被淘汰
	time.Sleep(1 * time.Second)

	// 删除指定的缓存条目
	cache.Del("key3")

	// 检查缓存的状态
	if val, found := cache.Get("key1"); found {
		fmt.Println("key1:", val)
	} else {
		fmt.Println("key1 not found")
	}

	if val, found := cache.Get("key4"); found {
		fmt.Println("key4:", val)
	} else {
		fmt.Println("key4 not found")
	}

	if val, found := cache.Get("key3"); found {
		fmt.Println("key3:", val)
	} else {
		fmt.Println("key3 not found") // key3 should be evicted or deleted
	}
}

func TestMapLRUCacheMemory(t *testing.T) {
	// 测量内存使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	// 创建 MapCache 缓存
	cache := NewCache(2000*10000, 2*time.Minute) // 设置最大缓存大小为3

	before := memStats.Alloc
	// 创建 N 万个 Event 对象并添加到缓存
	numEvents := 200 * 10000
	events := make([]*Event, numEvents)
	for i := 0; i < numEvents; i++ {
		events[i] = &Event{
			ID:          fmt.Sprintf("id-%d", i),
			Project:     "project",
			Version:     "v1.0.0",
			Context:     "context",
			Entity:      "entity",
			Event:       "event",
			Source:      "source",
			Params:      "params",
			AccessToken: "accessToken",
			CreatedAt:   1234567890,
			Sign:        "sign",
		}
		cache.Set(fmt.Sprintf("key-%d", i), events[i])
	}

	start := time.Now()

	// 校验缓存内容
	missingCount := 0
	for i := 0; i < numEvents; i++ {
		if _, found := cache.Get(fmt.Sprintf("key-%d", i)); !found {
			missingCount++
			// t.Errorf("cache missing key-%d", i)
		}
	}

	elapsed := time.Since(start)

	fmt.Printf("Time taken to validate cache: %v\n\n", elapsed)

	runtime.ReadMemStats(&memStats)
	used := memStats.Alloc - before
	fmt.Printf("Used memory: %.2f MB\n", float64(used)/(1024*1024))
	fmt.Printf("Total allocated memory: %.2f MB\n", float64(memStats.TotalAlloc)/(1024*1024))
	fmt.Printf("Heap memory: %.2f MB\n", float64(memStats.HeapAlloc)/(1024*1024))
	fmt.Printf("Number of missing keys: %d\n", missingCount)

}
