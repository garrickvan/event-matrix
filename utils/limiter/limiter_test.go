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

// Package limiter 的测试文件，包含各种限制器的测试用例
package limiter

import (
	"fmt"
	"testing"
	"time"
)

// TestCircuitBreaker 测试电路断路器的基本功能
// 测试场景:
// - 初始化一个TwoStepCircuitBreaker实例
// - 模拟多次请求，前20次失败，后续成功
// - 验证断路器状态转换和请求处理逻辑
func TestCircuitBreaker(t *testing.T) {
	tw := NewTwoStepCircuitBreaker[string](Settings{
		Name:        "ExampleBreaker",
		MaxRequests: 3, // 半开状态下最多允许3个请求通过
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from State, to State) {
			fmt.Printf("%s transitioned from %s to %s\n", name, from, to)
		},
	})
	for i := 0; i < 500; i++ {
		// 第一步：检查是否允许请求
		done, err := tw.Allow()
		if err != nil {
			// 如果熔断器拒绝请求，则输出拒绝信息
			fmt.Println("Request denied:", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// 第二步：执行请求并报告结果
		success := false
		if i >= 20 {
			success = true
		}
		done(success)

		// 你也可以根据业务需求使用 success 来决定是否执行其他逻辑
		if success {
			fmt.Println("Request succeeded")
		} else {
			fmt.Println("Request failed")
		}
	}
	println("TotalFailures: ", tw.Counts().TotalFailures, " TotalSuccesses: ", tw.Counts().TotalSuccesses)
}

// TestIsInIpWhitelist 测试IP白名单匹配功能
// 测试场景:
// - 多种IP地址与通配符模式的匹配情况
// - 包括完全匹配、部分通配符匹配和不匹配的情况
func TestIsInIpWhitelist(t *testing.T) {
	tests := []struct {
		ip      string
		pattern string
		expect  bool
	}{
		// Test cases
		{"192.168.1.1", "192.*.*.*", true},    // Match first part
		{"192.168.1.1", "192.168.*.*", true},  // Match second part
		{"192.168.1.1", "10.0.0.*", false},    // No match, different first part
		{"192.168.1.1", "192.168.1.*", true},  // Match last part
		{"192.168.1.1", "192.169.*.*", false}, // No match, second part differs
		{"192.168.1.1", "192.*.1.1", true},    // Match first and third parts
		{"192.168.1.1", "192.168.1.1", true},  // Exact match
		{"192.168.1.1", "192.168.1.2", false}, // No match, last part differs
		{"192.168.1.1", "*.*.*.*", true},      // Match all parts
		{"192.168.1.1", "*.*.*.1", true},      // Match first 3 parts, last is fixed
	}

	for _, test := range tests {
		t.Run(test.ip+"-"+test.pattern, func(t *testing.T) {
			got := IsInIpWhitelist(test.ip, test.pattern)
			if got != test.expect {
				t.Errorf("IsInIpWhitelist(%s, %s) = %v; want %v", test.ip, test.pattern, got, test.expect)
			}
		})
	}
}

// TestUnblockRateLimiter 测试非阻塞速率限制器的基本功能
// 测试场景:
// - 创建一个限制器，设置时间窗口为300ms，每窗口最大请求数为1
// - 连续发送多个请求，观察限制器的行为
// - 等待冷却后继续发送请求，验证限制器重置
func TestUnblockRateLimiter(t *testing.T) {
	limiter := NewUnBlockRateLimiter(
		300*time.Millisecond, // 时间窗口大小
		1,                    // 每窗口最大允许
		200*time.Microsecond, // 冷却时间
	)

	// 模拟请求
	for i := 1; i <= 10; i++ {
		fmt.Printf("Request %d: Allowed? %v\n", i, limiter.Allow())
		time.Sleep(100 * time.Millisecond)
	}

	// 等待冷却后继续请求
	time.Sleep(1 * time.Second)

	for i := 11; i <= 15; i++ {
		fmt.Printf("Request %d: Allowed? %v\n", i, limiter.Allow())
		time.Sleep(1 * time.Second)
	}
}
