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

// Package serverx 提供服务器抽象层和网络通信基础设施
// 该包定义了服务器和客户端的核心接口，以及请求上下文等基础类型
package serverx

/**
 * 所有的服务器实现都需要实现 Server 接口
 * 所有的客户端实现都需要实现 SyncClient 接口
 * 用于定义服务器的基本操作，如启动、停止、设置事件处理函数等
 * 确保底层可控，以及以便将来可以扩展至其他服务器实现或适配更多的传输协议
 **/

import (
	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
)

// Server 定义了所有服务器实现必须提供的基本功能接口
type Server interface {
	// ServerId 返回全局服务器的唯一标识
	ServerId() string
	// Start 启动服务器，完成必要的初始化工作
	Start() error
	// Stop 停止服务器，执行清理工作
	Stop() error
}

// HandleFunc 定义事件处理函数类型
// 接收一个 RequestContext 参数，返回处理过程中的错误（如果有）
type HandleFunc func(RequestContext) error

// OnStartFunc 定义启动事件回调函数类型
// 接收 NetworkServer 参数，返回是否已完成处理
type OnStartFunc func(NetworkServer) bool

// OnStopFunc 定义停止事件回调函数类型
// 接收 NetworkServer 参数，返回是否已完成处理
type OnStopFunc func(NetworkServer) bool

// NetworkServer 扩展了 Server 接口，提供网络服务器特有的功能
type NetworkServer interface {
	Server
	// OnStart 注册启动事件回调函数
	// 用于在启动服务器时执行一些对特定服务器实现的初始化操作，比如注册中间件等
	OnStart(OnStartFunc)
	// OnStop 注册停止事件回调函数
	// 用于在停止服务器时执行一些对特定服务器实现的清理操作，比如注销中间件等
	OnStop(OnStopFunc)
	// UnHandle 设置未处理的网络事件的处理函数
	UnHandle(HandleFunc)
	// Impl 返回服务器的底层实现对象
	// 返回值是一个接口，需要通过类型断言转换为具体的实现类型
	Impl() interface{}
}

// CONTENT_TYPE 定义了请求内容的类型
type CONTENT_TYPE uint8

const (
	// CONTENT_TYPE_PING 表示心跳检测类型
	CONTENT_TYPE_PING CONTENT_TYPE = 0
	// CONTENT_TYPE_JSON 表示JSON格式内容
	CONTENT_TYPE_JSON CONTENT_TYPE = 1
	// CONTENT_TYPE_STRING 表示普通字符串内容
	CONTENT_TYPE_STRING CONTENT_TYPE = 2
)

// RequestContext 定义了请求上下文接口，封装了请求和响应的处理方法
type RequestContext interface {
	// IP 获取客户端IP地址
	IP() string

	// Path 获取请求路径
	Path() string

	// Body 获取请求体原始字节数据，需要注意的是，如果底层用了零拷贝机制，要注意带来的副作用，比如strings.Split()可能会导致越界判断
	Body() []byte

	// SetStatus 设置响应状态码，默认为200
	// 返回当前上下文以支持链式调用
	SetStatus(int) RequestContext

	// BodyType 获取请求体类型（JSON或字符串）
	BodyType() CONTENT_TYPE

	// IsJsonBody 判断请求体是否为JSON格式
	IsJsonBody() bool

	// Header 获取指定请求头的值
	Header(key string) string

	// SetHeader 设置响应头
	SetHeader(key, value string)

	// Response 发送原始字节响应
	Response(bytes []byte) error

	// ResponseString 发送字符串响应
	ResponseString(string) error

	// ResponseJson 发送JSON格式响应，自动序列化
	ResponseJson(interface{}) error

	// ResponseBuiltinJson 发送内置JSON响应
	ResponseBuiltinJson(code constant.RESPONSE_CODE) error

	// Event 获取核心事件对象
	Event() *core.Event

	// EntityEvent 获取实体事件对象
	EntityEvent() *core.EntityEvent

	// CallChain 获取调用链信息，用于防止循环调用
	CallChain() []string

	// Data 获取临时存储的上下文数据
	// 注意：建议使用类型断言转换为具体结构体，而不是直接使用map
	Data() interface{}

	// SetData 设置临时存储的上下文数据
	SetData(interface{})

	// CtxImpl 获取底层实现对象，用于类型断言获取具体实现
	CtxImpl() interface{}
}

// RequestPacket 定义处理请求包的接口
type RequestPacket interface {
	// Pack 将请求包序列化为二进制数据
	Pack(compressed bool) []byte

	// Unmarshal 将二进制数据反序列化为请求包
	Unmarshal(data []byte) error

	// Type 获取请求包的内容类型
	Type() CONTENT_TYPE

	// Extend 获取请求包的扩展数据
	Extend() string

	// TemporaryData 获取请求包的临时数据，当前请求结束即回收
	TemporaryData() string

	// IP 获取请求包的来源IP
	IP() string

	// CallChains 获取请求包的调用链信息
	CallChains() string

	// CreateTime 获取请求包的时间戳
	CreateTime() int64
}

// ResponsePacket 定义处理响应包的接口
type ResponsePacket interface {
	// Pack 将响应包序列化为二进制数据
	Pack(compressed bool) []byte

	// Unmarshal 将二进制数据反序列化为响应包
	Unmarshal(data []byte) error

	// Status 获取响应包的状态码
	Status() int

	// Type 获取请求包的内容类型
	Type() CONTENT_TYPE

	// TemporaryData 获取请求包的临时数据，当前请求结束即回收
	TemporaryData() string

	// CreateTime 获取请求包的时间戳
	CreateTime() int64
}
