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

// Package gnetx 提供基于gnet的内域服务器实现
package gnetx

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/panjf2000/gnet/v2"
)

// IntranetServer 是内域服务器的实现，基于gnet框架
type IntranetServer struct {
	gnet.BuiltinEventEngine // 继承gnet的事件引擎

	port           int    // 服务器监听端口
	serverId       string // 服务器唯一标识
	algorithm      string // 加密算法名称
	intranetSecret string // 内域通信的密钥

	onStartHandler serverx.OnStartFunc // 服务器启动时的回调函数
	onStopHandler  serverx.OnStopFunc  // 服务器停止时的回调函数
	unHandleFunc   serverx.HandleFunc  // 未实现的处理函数

	router     IntranetServerRouter // 请求路由函数
	routerImpl interface{}          // 工作服务器实现

	connCount    int64 // 当前连接数
	reqCounter   int64 // 请求计数器
	errorCounter int64 // 错误计数器
}

// IntranetServerRouter 是处理请求的路由函数类型
// 参数：
//   - req: 请求包
//   - c: gnet连接对象
//   - routerImpl: 路由实现对象
//
// 返回值：
//   - *ResponsePacketImpl: 响应消息
type IntranetServerRouter func(req serverx.RequestPacket, c gnet.Conn, routerImpl interface{}) serverx.ResponsePacket

// NewIntranetServer 创建一个新的内域服务器实例
//
// 参数：
//   - serverId: 服务器唯一标识
//   - port: 监听端口
//   - secret: 内域通信密钥
//   - algor: 加密算法
//   - router: 路由处理函数
//   - routerImpl: 路由实现对象
//
// 返回值：
//   - *IntranetServer: 新创建的服务器实例
func NewIntranetServer(serverId string, port int, secret, algor string, router IntranetServerRouter, routerImpl interface{}) *IntranetServer {
	return &IntranetServer{
		serverId:       serverId,   // 服务器ID
		port:           port,       // 服务器监听端口
		intranetSecret: secret,     // 内域通信密钥
		algorithm:      algor,      // 加密算法
		router:         router,     // 请求路由函数
		routerImpl:     routerImpl, // 工作服务器实现
	}
}

// ServerId 返回服务器ID
func (s *IntranetServer) ServerId() string { return s.serverId }

// Start 启动服务器
//
// 实现细节：
//  1. 执行启动回调函数（如果有）
//  2. 检查端口可用性
//  3. 启动gnet服务器，支持多核、端口复用等特性
//  4. 最多重试3次
//
// 返回值：
//   - error: 启动过程中的错误，成功则返回nil
func (s *IntranetServer) Start() error {
	defer func() {
		if r := recover(); r != nil {
			logx.Log().Error(fmt.Sprintf("Server panic on start: %v", r))
		}
	}()

	// 如果设置了启动回调函数，并且回调函数返回true，则直接返回
	if s.onStartHandler != nil && s.onStartHandler(s) {
		return nil
	}

	// 检查端口直至可用
	if err := serverx.WaitUntilPortAvailable(s.port, 5); err != nil {
		return fmt.Errorf("port check failed: %w", err)
	}

	// 启动服务器
	logx.Info("Starting intranet server on port: ", s.port)
	tryTimes := 0
	for tryTimes < 3 {
		err := gnet.Run(s, fmt.Sprintf("tcp://:%d", s.port),
			gnet.WithMulticore(true),             // 使用多核
			gnet.WithReusePort(true),             // 启用端口复用
			gnet.WithReuseAddr(true),             // 关键：允许地址重用
			gnet.WithTCPKeepAlive(time.Minute),   // 设置TCP KeepAlive
			gnet.WithTCPNoDelay(gnet.TCPNoDelay), // 启用TCP_NODELAY
			// gnet.WithTicker(true),                         // 启用定时器
			gnet.WithLoadBalancing(gnet.LeastConnections), // 使用最少连接负载均衡
		)
		if err != nil {
			tryTimes++
			logx.Error("Start server failed, try again: ", err)
			time.Sleep(time.Second * 3)
		}
	}
	if tryTimes == 3 {
		return fmt.Errorf("start server failed after 3 times")
	}
	return nil
}

// OnStart 设置服务器启动时的回调函数
func (s *IntranetServer) OnStart(handler serverx.OnStartFunc) {
	s.onStartHandler = handler
}

// Stop 停止服务器
func (s *IntranetServer) Stop() error {
	// 如果设置了停止回调函数，并且回调函数返回true，则直接返回
	if s.onStopHandler != nil && s.onStopHandler(s) {
		return nil
	}
	// UNIMPLEMENTED: 停止Gnet服务器
	return nil
}

// OnStop 设置服务器停止时的回调函数
func (s *IntranetServer) OnStop(handler serverx.OnStopFunc) {
	s.onStopHandler = handler
}

// UnHandle 设置未实现的处理函数
func (s *IntranetServer) UnHandle(handler serverx.HandleFunc) {
	s.unHandleFunc = handler
}

// GetUnHandler 获取未实现的处理函数，如果未设置则返回默认处理函数
func (s *IntranetServer) GetUnHandler() serverx.HandleFunc {
	if s.unHandleFunc == nil {
		return defaultUnHandle
	}
	return s.unHandleFunc
}

// defaultUnHandle 默认的未实现处理函数
func defaultUnHandle(ctx serverx.RequestContext) error {
	return ctx.SetStatus(http.StatusNotImplemented).Response([]byte("unimplemented intranet request event"))
}

// Impl 返回服务器实现
func (s *IntranetServer) Impl() interface{} {
	return s
}

// OnOpen 当有新连接时调用
//
// 参数：
//   - c: 新建立的连接
//
// 返回值：
//   - []byte: 要发送给客户端的数据
//   - gnet.Action: 后续动作
func (s *IntranetServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	atomic.AddInt64(&s.connCount, 1) // 增加连接数
	return nil, gnet.None
}

// OnClose 当连接关闭时调用
//
// 参数：
//   - c: 关闭的连接
//   - err: 关闭原因
//
// 返回值：
//   - gnet.Action: 后续动作
func (s *IntranetServer) OnClose(c gnet.Conn, err error) gnet.Action {
	atomic.AddInt64(&s.connCount, -1) // 减少连接数
	if err != nil && err.Error() != "read: EOF" {
		logx.Log().Error("Connection closed with error: " + err.Error())
	}
	return gnet.None
}

// ConnectionCount 返回当前连接数
func (s *IntranetServer) ConnectionCount() int64 {
	return atomic.LoadInt64(&s.connCount)
}

// RequestCount 返回请求计数器
func (s *IntranetServer) RequestCount() int64 {
	return atomic.LoadInt64(&s.reqCounter)
}

// ErrorCount 返回错误计数器
func (s *IntranetServer) ErrorCount() int64 {
	return atomic.LoadInt64(&s.errorCounter)
}
