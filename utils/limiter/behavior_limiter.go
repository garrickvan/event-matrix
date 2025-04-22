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

// Package limiter 提供了多种限流器实现，用于控制请求频率和行为次数
package limiter

import (
	"github.com/garrickvan/event-matrix/utils/cachex"
	"github.com/spf13/cast"
)

// UserBehaviorLimiter 是一个用户行为限制器，用于限制用户在特定时间窗口内的行为次数。
// 时间窗口的长度由缓存的过期时间决定，超过限制次数后将不允许执行该行为。
// 主要用于防止用户在短时间内频繁执行某些敏感操作，如登录尝试、发送验证码等。

// UserBehaviorLimiter 结构体定义了行为限制器的核心组件
type UserBehaviorLimiter struct {
	cache *cachex.LocalCache // 用于存储行为次数的本地缓存
	max   int                // 允许的最大行为次数
}

// NewUserBehaviorLimiter 创建一个新的用户行为限制器实例
// cache: 用于存储行为次数的本地缓存，其过期时间决定了行为限制的时间窗口
// max: 在时间窗口内允许的最大行为次数
// 返回: 初始化后的UserBehaviorLimiter实例
func NewUserBehaviorLimiter(cache *cachex.LocalCache, max int) *UserBehaviorLimiter {
	return &UserBehaviorLimiter{
		cache: cache,
		max:   max,
	}
}

// CanExecute 检查指定的行为是否可以执行
// behaveKey: 行为标识键，用于唯一标识某个用户的特定行为
// 返回: 如果行为次数未超过限制则返回true，否则返回false
func (l *UserBehaviorLimiter) CanExecute(behaveKey string) bool {
	if l == nil {
		return false
	}
	behaveTimes, ok := l.cache.Get(behaveKey)
	if !ok {
		behaveTimes = 1
	}
	if cast.ToInt(behaveTimes) <= l.max {
		return true
	}
	return false
}

// ExecSuccess 标记行为执行成功，清除该行为的计数
// behaveKey: 行为标识键
func (l *UserBehaviorLimiter) ExecSuccess(behaveKey string) {
	if l == nil {
		return
	}
	l.cache.Del(behaveKey)
}

// ExecFailed 标记行为执行失败，增加失败计数
// behaveKey: 行为标识键
// 当行为执行失败时调用此方法，会增加对应行为的计数
func (l *UserBehaviorLimiter) ExecFailed(behaveKey string) {
	if l == nil {
		return
	}
	behaveTimes, ok := l.cache.Get(behaveKey)
	if !ok {
		behaveTimes = 1
	}
	behaveTimes = cast.ToInt(behaveTimes) + 1
	l.cache.Put(behaveKey, behaveTimes)
}
