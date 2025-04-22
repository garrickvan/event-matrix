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

// Package hertzx 提供基于Hertz框架的公共服务器实现
package hertzx

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils/logx"
)

// PublicServer 实现了面向公网的HTTP服务器
type PublicServer struct {
	port           int                 // 服务器监听端口
	serverId       string              // 服务器唯一标识
	hz             *server.Hertz       // Hertz服务器实例
	onStartHandler serverx.OnStartFunc // 服务器启动时的回调函数
	onStopHandler  serverx.OnStopFunc  // 服务器停止时的回调函数
	unHandle       serverx.HandleFunc  // 未实现的处理函数

	connCount    int64 // 当前连接数统计
	reqCounter   int64 // 请求计数器
	errorCounter int64 // 错误计数器
}

// NewPublicServer 创建一个新的公共服务器实例
//
// 参数：
//   - port: 监听端口
//   - serverId: 服务器唯一标识
//
// 返回值：
//   - *PublicServer: 新创建的服务器实例
func NewPublicServer(port int, serverId string) *PublicServer {
	hlog.SetLevel(hlog.LevelWarn)
	ps := &PublicServer{
		serverId: serverId,
		port:     port,
		hz: server.Default(
			server.WithHostPorts(fmt.Sprintf(":%d", port)),
		),
		unHandle: defaultUnHandle,
	}
	// 添加panic恢复中间件
	ps.hz.Use(func(ctx context.Context, c *app.RequestContext) {
		defer func() {
			if err := recover(); err != nil {
				if err, ok := err.(error); ok {
					ps.OnError()
					logx.Log().Error("Recovered from panic:" + err.Error())
				}
				c.String(consts.StatusInternalServerError, "Public Server Internal Error")
				c.Abort()
			}
		}()
		ps.OnRequest()
		ps.OnOpen()
		c.Next(ctx)
		ps.OnClose()
	})
	return ps
}

// ServerId 返回服务器唯一标识
func (s *PublicServer) ServerId() string {
	return s.serverId
}

// Start 启动服务器
//
// 实现细节：
//  1. 执行启动回调函数（如果有）
//  2. 检查端口可用性
//  3. 启动Hertz服务器
//
// 返回值：
//   - error: 启动过程中的错误，成功则返回nil
func (s *PublicServer) Start() error {
	var err error
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("public server panic: %v", r)
			}
		}
	}()
	if s.onStartHandler != nil {
		s.onStartHandler(s)
	}
	// 判断端口是否可用
	if err = serverx.WaitUntilPortAvailable(s.port, 5); err != nil {
		return err
	}
	logx.Info("Starting public server on port: ", s.port)
	s.hz.Spin()
	return err
}

// OnStart 设置服务器启动时的回调函数
func (s *PublicServer) OnStart(handler serverx.OnStartFunc) {
	if s.onStartHandler != nil {
		logx.Debug("OnStart handler has already been set, will be overwritten")
	}
	s.onStartHandler = handler
}

// Stop 停止服务器
func (s *PublicServer) Stop() error {
	if s.onStopHandler != nil {
		s.onStopHandler(s)
	}
	return s.hz.Close()
}

// OnStop 设置服务器停止时的回调函数
func (s *PublicServer) OnStop(handler serverx.OnStopFunc) {
	if s.onStopHandler != nil {
		logx.Debug("OnStop handler has already been set, will be overwritten")
	}
	s.onStopHandler = handler
}

// Impl 返回服务器的底层实现对象（Hertz实例）
func (s *PublicServer) Impl() interface{} {
	return s.hz
}

// UnHandle 设置未实现的处理函数
func (s *PublicServer) UnHandle(handler serverx.HandleFunc) {
	if s.unHandle != nil {
		logx.Debug("UnHandle handler has already been set, will be overwritten")
	}
	s.unHandle = handler
}

// GetUnHandler 获取未实现的处理函数
func (s *PublicServer) GetUnHandler() serverx.HandleFunc {
	if s.unHandle == nil {
		return defaultUnHandle
	}
	return s.unHandle
}

// defaultUnHandle 默认的未处理请求处理函数
func defaultUnHandle(ctx serverx.RequestContext) error {
	return ctx.SetStatus(http.StatusNotImplemented).Response([]byte("unhandle public server request event"))
}

// OnOpen 当有新连接时调用
func (s *PublicServer) OnOpen() {
	atomic.AddInt64(&s.connCount, 1)
}

// OnClose 当连接关闭时调用
func (s *PublicServer) OnClose() {
	atomic.AddInt64(&s.connCount, -1)
}

// OnRequest 当收到新请求时调用
func (s *PublicServer) OnRequest() {
	atomic.AddInt64(&s.reqCounter, 1)
}

// OnError 当发生错误时调用
func (s *PublicServer) OnError() {
	atomic.AddInt64(&s.errorCounter, 1)
}

// ConnectionCount 返回当前连接数
func (s *PublicServer) ConnectionCount() int64 {
	return atomic.LoadInt64(&s.connCount)
}

// RequestCount 返回请求计数
func (s *PublicServer) RequestCount() int64 {
	return atomic.LoadInt64(&s.reqCounter)
}

// ErrorCount 返回错误计数
func (s *PublicServer) ErrorCount() int64 {
	return atomic.LoadInt64(&s.errorCounter)
}
