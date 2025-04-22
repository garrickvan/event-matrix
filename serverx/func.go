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

// Package serverx 提供服务器相关的辅助函数
package serverx

import (
	"fmt"
	"net"
	"time"
)

// CheckPortAvailable 检查指定端口是否可用
//
// 参数:
//   - port: 要检查的端口号
//
// 返回值:
//   - error: 如果端口不可用返回错误，可用则返回nil
//
// 实现细节:
//  1. 尝试在指定端口上创建TCP监听器
//  2. 如果创建成功，立即关闭监听器
//  3. 等待200ms确保端口完全释放
func CheckPortAvailable(port int) error {
	// 尝试在指定端口上监听
	listener, err := net.Listen("tcp", ":"+fmt.Sprintf("%d", port))
	if err != nil {
		// 如果端口已被占用，返回错误
		return fmt.Errorf("port %d is already in use: %v", port, err)
	}
	// 关闭监听器
	if err := listener.Close(); err != nil {
		return fmt.Errorf("failed to close listener: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	// 端口可用
	return nil
}

// WaitUntilPortAvailable 等待直到指定端口可用或超时
//
// 参数:
//   - port: 要等待的端口号
//   - times: 尝试次数，每次等待2秒
//
// 返回值:
//   - error: 如果在指定次数内端口仍然不可用返回错误，可用则返回nil
//
// 实现细节:
//  1. 每2秒检查一次端口是否可用
//  2. 如果端口可用，立即返回nil
//  3. 如果达到最大尝试次数仍不可用，返回错误
func WaitUntilPortAvailable(port int, times int) error {
	// 尝试在指定端口上监听
	for i := 0; i < times; i++ {
		if err := CheckPortAvailable(port); err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
		fmt.Printf("Waiting for port %d to be available... (%d/%d)\n", port, i+1, times)
	}
	return fmt.Errorf("port %d is still not available after %d seconds", port, times*2)
}
