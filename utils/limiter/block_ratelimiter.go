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

package limiter

import (
	"errors"
	"strings"
	"time"

	"github.com/garrickvan/event-matrix/utils/cachex"
	"go.uber.org/ratelimit"
)

// BlockRateLimiterMgr 是一个基于令牌桶算法实现的速率限制器管理器
// 特点:
// - 限制每个用户行为的访问速率
// - 当请求速率超过限制时，会阻塞/延时请求而不是直接拒绝
// - 支持白名单功能，白名单中的请求不受限制
//
// 注意：由于会阻塞请求，在高并发场景下可能导致线程堆积，有造成服务器资源耗尽的风险
// 对于资源有限的服务器环境，建议谨慎使用或考虑使用UnBlockRateLimiter

// BlockRateLimiterMgr 结构体定义了阻塞式速率限制器管理器的核心组件
type BlockRateLimiterMgr struct {
	cache         *cachex.LocalCache                           // 用于存储各个行为对应的限制器实例
	rate          int                                          // 速率限制，表示每秒允许的请求数
	keyWhitelist  []string                                     // 白名单列表，白名单中的行为不受限制
	isInWhitelist func(Key string, KeyInWhiteList string) bool // 用于判断Key是否在白名单中的自定义函数
}

// NewBlockRateLimiterMgr 创建一个新的阻塞式速率限制器管理器
// cache: 用于存储限制器实例的缓存
// rate: 速率限制，表示每秒允许的请求数
// isInWhitelist: 自定义函数，用于判断行为键是否在白名单中
// 返回: 初始化后的BlockRateLimiterMgr实例和可能的错误
func NewBlockRateLimiterMgr(
	cache *cachex.LocalCache,
	rate int,
	isInWhitelist func(behaveKey string, whiteList string) bool,
) (*BlockRateLimiterMgr, error) {
	if cache == nil {
		return nil, errors.New("BlockRateLimiterMgr must have cache")
	}
	if isInWhitelist == nil {
		isInWhitelist = func(behaveKey string, whiteList string) bool {
			return behaveKey == whiteList
		}
	}
	return &BlockRateLimiterMgr{
		cache:         cache,
		rate:          rate,
		keyWhitelist:  make([]string, 0),
		isInWhitelist: isInWhitelist,
	}, nil
}

// UpdateRate 更新速率限制值
// rate: 新的速率限制值，表示每秒允许的请求数
func (b *BlockRateLimiterMgr) UpdateRate(rate int) {
	b.rate = rate
}

// UpdateWhitelist 更新白名单列表
// behaveKeys: 以逗号分隔的白名单键列表字符串，为空时清空白名单
func (b *BlockRateLimiterMgr) UpdateWhitelist(behaveKeys string) {
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
func (b *BlockRateLimiterMgr) checkWhitelist(behaveKey string) bool {
	for _, key := range b.keyWhitelist {
		if b.isInWhitelist(behaveKey, key) {
			return true
		}
	}
	return false
}

// Take 执行速率限制检查，可能会阻塞调用直到允许通过
// behaveKey: 行为标识键，用于唯一标识某个行为
// 返回: 请求被处理的时间点
// 注意: 此方法在超过速率限制时会阻塞调用，而不是立即返回错误
func (b *BlockRateLimiterMgr) Take(behaveKey string) time.Time {
	var limiter ratelimit.Limiter

	if b.checkWhitelist(behaveKey) {
		return time.Now()
	}

	l, found := b.cache.Get(behaveKey)
	if found {
		limiter = l.(ratelimit.Limiter)
	} else {
		limiter = ratelimit.New(b.rate)
		b.cache.Put(behaveKey, limiter)
	}

	return limiter.Take()
}

// IsInIpWhitelist 检查给定的IP地址是否匹配指定的IP通配符模式
// ip: 要检查的IP地址，格式为x.x.x.x
// ipRegular: IP通配符模式，如"192.168.*.*"
// 返回: 如果IP地址匹配通配符模式则返回true，否则返回false
// 示例: IsInIpWhitelist("192.168.1.1", "192.168.*.*") 返回 true
func IsInIpWhitelist(ip string, ipRegular string) bool {
	// Split both ip and ipRegular into segments
	ipSegments := strings.Split(ip, ".")
	regularSegments := strings.Split(ipRegular, ".")
	// Ensure both IP and pattern have 4 segments
	if len(ipSegments) != 4 || len(regularSegments) != 4 {
		return false
	}
	// Check each segment
	for i := 0; i < 4; i++ {
		// If the pattern segment is "*" it matches any IP segment
		if regularSegments[i] == "*" {
			continue
		}
		// If the pattern segment is not "*" and doesn't match the IP segment, return false
		if ipSegments[i] != regularSegments[i] {
			return false
		}
	}
	// All segments match
	return true
}
