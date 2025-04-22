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

// 为避免循环引用，Package constant 定义了事件矩阵系统中使用的各种常量和类型
package constant

import "time"

// SPLIT_CHAR 用于内部固定结果的传输拼接字符
const SPLIT_CHAR = ","

const (
	// INITIAL_VERSION 定义初始版本号
	INITIAL_VERSION = "0.0.0"
)

// SLOW_REQUST_TIME 定义请求响应时间的阈值，超过此值视为慢请求
const SLOW_REQUST_TIME = 500 * time.Millisecond
