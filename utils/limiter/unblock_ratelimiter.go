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

// Package limiter 实现了非阻塞式速率限制器
package limiter

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/garrickvan/event-matrix/utils/cachex"
)

// NewUnBlockRateLimiter 创建一个非阻塞式速率限制器实例
// windowSize: 时间窗口大小，在此时间范围内限制请求次数
// maxRequests: 在时间窗口内允许的最大请求数
// cooldownTime: 达到限制后的冷却时间
// 返回: 初始化后的UnBlockRateLimiter实例
func NewUnBlockRateLimiter(
	windowSize time.Duration,
	maxRequests int,
	cooldownTime time.Duration,
) *UnBlockRateLimiter {
	return &UnBlockRateLimiter{
		windowSize:   windowSize,
		maxRequests:  maxRequests,
		cooldownTime: cooldownTime,
	}
}

// UnBlockRateLimiter 实现了一个非阻塞的速率限制器
// 特点:
// - 基于滑动时间窗口算法
// - 超过限制时直接拒绝请求而不是阻塞
// - 支持冷却时间，防止频繁重试
type UnBlockRateLimiter struct {
	windowSize   time.Duration // 时间窗口大小
	maxRequests  int           // 时间窗口内的最大请求次数
	cooldownTime time.Duration // 冷却时间长度

	cooldownUntil time.Time  // 冷却状态结束的时间
	currentCount  int        // 当前时间窗口内的请求计数
	windowStart   time.Time  // 当前时间窗口的起始时间
	mu            sync.Mutex // 并发安全锁
}

// Allow 检查是否允许当前操作
// 返回: 如果允许请求则返回true，否则返回false
// 说明:
// - 在冷却时间内，直接拒绝请求
// - 超出时间窗口时，重置计数
// - 未达到限制时，允许请求并增加计数
// - 达到限制时，进入冷却状态并拒绝请求
func (r *UnBlockRateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// 如果在冷却时间内，拒绝请求
	if now.Before(r.cooldownUntil) {
		return false
	}

	// 如果超出时间窗口，重置计数和窗口起始时间
	if now.Sub(r.windowStart) >= r.windowSize {
		r.windowStart = now
		r.currentCount = 0
	}

	// 如果计数未达到限制，允许请求并递增计数
	if r.currentCount < r.maxRequests {
		r.currentCount++
		return true
	}

	// 达到限制，记录冷却开始时间，拒绝请求
	r.cooldownUntil = now.Add(r.cooldownTime)
	return false
}

// UnBlockRateLimiterMgr 管理多个非阻塞速率限制器实例
// 支持:
// - 为不同的行为键维护独立的限制器
// - 白名单机制
// - 动态配置管理
type UnBlockRateLimiterMgr struct {
	cache         *cachex.LocalCache
	windowSize    time.Duration
	maxRequests   int
	cooldownTime  time.Duration
	keyWhitelist  []string
	isInWhitelist func(Key string, KeyInWhiteList string) bool // 用于判断Key是否在白名单中
}

// NewUnBlockRateLimiterMgr 创建一个非阻塞速率限制器管理器实例
// cache: 用于存储限制器实例的缓存
// windowSize: 时间窗口大小
// maxRequests: 窗口内最大请求数
// cooldownTime: 冷却时间
// isInWhitelist: 自定义函数，用于判断行为键是否在白名单中
// 返回: 初始化后的UnBlockRateLimiterMgr实例和可能的错误
func NewUnBlockRateLimiterMgr(
	cache *cachex.LocalCache,
	windowSize time.Duration,
	maxRequests int,
	cooldownTime time.Duration,
	isInWhitelist func(Key string, KeyInWhiteList string) bool,
) (*UnBlockRateLimiterMgr, error) {
	if cache == nil {
		return nil, errors.New("BlockRateLimiterMgr must have cache")
	}
	if isInWhitelist == nil {
		isInWhitelist = func(behaveKey string, whiteList string) bool {
			return behaveKey == whiteList
		}
	}

	return &UnBlockRateLimiterMgr{
		cache:         cache,
		windowSize:    windowSize,
		maxRequests:   maxRequests,
		cooldownTime:  cooldownTime,
		keyWhitelist:  []string{},
		isInWhitelist: isInWhitelist,
	}, nil
}

// UpdateWhitelist 更新白名单列表
// behaveKeys: 以逗号分隔的白名单键列表字符串，为空时清空白名单
func (b *UnBlockRateLimiterMgr) UpdateWhitelist(behaveKeys string) {
	if behaveKeys == "" {
		b.keyWhitelist = make([]string, 0)
		return
	}
	keyArray := strings.Split(behaveKeys, ",")
	b.keyWhitelist = keyArray
}

// checkWhitelist 检查给定的行为键是否在白名单中
// behaveKey: 要检查的行为键
// 返回: 如果行为键在白名单中则返回true，否则返回false
func (b *UnBlockRateLimiterMgr) checkWhitelist(behaveKey string) bool {
	for _, key := range b.keyWhitelist {
		if b.isInWhitelist(behaveKey, key) {
			return true
		}
	}
	return false
}

// Allow 检查指定行为是否允许执行
// behaveKey: 行为标识键
// 返回: 如果允许执行则返回true，否则返回false
// 说明:
// - 白名单中的行为总是允许执行
// - 为每个行为键维护独立的限制器实例
// - 首次遇到的行为键会创建新的限制器实例
func (b *UnBlockRateLimiterMgr) Allow(behaveKey string) bool {
	var limiter *UnBlockRateLimiter

	if b.checkWhitelist(behaveKey) {
		return true
	}

	l, found := b.cache.Get(behaveKey)
	if found {
		limiter = l.(*UnBlockRateLimiter)
	} else {
		limiter = NewUnBlockRateLimiter(b.windowSize, b.maxRequests, b.cooldownTime)
		b.cache.Put(behaveKey, limiter)
	}

	return limiter.Allow()
}
