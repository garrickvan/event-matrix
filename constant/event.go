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

package constant

// EVENT_MODE 定义事件处理模式
type EVENT_MODE string

const (
	// QUERY_MODE 查询模式，使用高性能只读数据库
	QUERY_MODE EVENT_MODE = "Q"
	// COMMAND_MODE 命令模式，使用高性能读写数据库，事务事件必须部署在此模式
	COMMAND_MODE EVENT_MODE = "C"
)

// EXECUTOR_TYPE 定义事件执行器的类型
type EXECUTOR_TYPE uint8

const (
	// BUILD_IN_EXECUTOR 内置执行器，系统默认提供的执行器
	BUILD_IN_EXECUTOR EXECUTOR_TYPE = 0
	// CUSTOM_EXECUTOR 自定义执行器，用户自定义的执行器
	CUSTOM_EXECUTOR EXECUTOR_TYPE = 1
	// TASK_EXECUTOR 任务执行器，用于处理异步任务
	TASK_EXECUTOR EXECUTOR_TYPE = 2
)

const (
	// EVENT_SOURCE_WEB_API Web API来源的事件
	EVENT_SOURCE_WEB_API = "web_api"
	// EVENT_SOURCE_INTERNAL 内部系统产生的事件
	EVENT_SOURCE_INTERNAL = "internal"
	// EVENT_SOURCE_ANDROID Android客户端来源的事件
	EVENT_SOURCE_ANDROID = "android"
	// EVENT_SOURCE_IOS iOS客户端来源的事件
	EVENT_SOURCE_IOS = "ios"
	// EVENT_SOURCE_WINDOWS Windows客户端来源的事件
	EVENT_SOURCE_WINDOWS = "windows"
	// EVENT_SOURCE_LINUX Linux客户端来源的事件
	EVENT_SOURCE_LINUX = "linux"
	// EVENT_SOURCE_MAC macOS客户端来源的事件
	EVENT_SOURCE_MAC = "mac"
	// EVENT_SOURCE_HOS 鸿蒙系统来源的事件
	EVENT_SOURCE_HOS = "hos"
	// EVENT_SOURCE_UNKNOWN 未知来源的事件
	EVENT_SOURCE_UNKNOWN = "unknown"
)

const (
	// RULE_MSG_TYPE_EVENT 规则引擎中的事件消息类型
	RULE_MSG_TYPE_EVENT = "event"
)
