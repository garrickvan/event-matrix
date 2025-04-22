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

// Package hertzx 提供基于Hertz框架的HTTP请求上下文实现
package hertzx

import (
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/network"
	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/jsonx"
)

// RequestContext 用于封装请求的上下文信息
// 实现了serverx.RequestContext接口，提供HTTP请求处理的核心功能
type RequestContext struct {
	status      int                 // HTTP响应状态码
	event       *core.Event         // 核心事件对象
	entityEvent *core.EntityEvent   // 实体事件对象
	hertzCtx    *app.RequestContext // Hertz框架的请求上下文
	tmpData     interface{}         // 临时数据存储
}

// NewRequestContext 创建一个新的RequestContext实例
//
// 参数：
//   - hertzCtx: Hertz框架的请求上下文
//
// 返回值：
//   - *RequestContext: 新创建的请求上下文
func NewRequestContext(hertzCtx *app.RequestContext) *RequestContext {
	return &RequestContext{
		status:      0,
		event:       nil,
		entityEvent: nil,
		hertzCtx:    hertzCtx,
		tmpData:     nil,
	}
}

// IP 返回请求客户端的IP地址
// 对于本地回环地址::1，转换为127.0.0.1
func (r *RequestContext) IP() string {
	if r.hertzCtx == nil {
		return ""
	}
	ip := r.hertzCtx.ClientIP()
	if ip == "::1" {
		ip = "127.0.0.1"
	}
	return ip
}

// Path 返回请求的完整路径
func (r *RequestContext) Path() string {
	return fastconv.BytesToString(r.hertzCtx.Path())
}

// Body 返回请求的体内容
func (r *RequestContext) Body() []byte {
	return r.hertzCtx.Request.Body()
}

// BodyType 返回请求体的内容类型
// 根据Content-Type头判断是否为JSON
func (r *RequestContext) BodyType() serverx.CONTENT_TYPE {
	contentType := r.hertzCtx.Request.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return serverx.CONTENT_TYPE_JSON
	}
	return serverx.CONTENT_TYPE_STRING
}

// IsJsonBody 判断请求体是否为JSON格式
func (r *RequestContext) IsJsonBody() bool {
	return r.BodyType() == serverx.CONTENT_TYPE_JSON
}

// Header 返回请求头中指定键的值
func (r *RequestContext) Header(key string) string {
	return r.hertzCtx.Request.Header.Get(key)
}

// SetHeader 设置响应头中的键值对
func (r *RequestContext) SetHeader(key, value string) {
	r.hertzCtx.Response.Header.Set(key, value)
}

// SetStatus 设置HTTP响应的状态码
// 返回当前上下文以支持链式调用
func (r *RequestContext) SetStatus(code int) serverx.RequestContext {
	r.status = code
	return r
}

// Response 发送HTTP响应，内容为字节数组
func (r *RequestContext) Response(strBytes []byte) error {
	if r.status == 0 {
		r.hertzCtx.SetStatusCode(http.StatusOK)
	} else {
		r.hertzCtx.SetStatusCode(r.status)
	}
	_, err := r.hertzCtx.Write(strBytes)
	return err
}

// ResponseString 发送HTTP响应，内容为字符串
// 内部转换为字节数组后调用Response方法
func (r *RequestContext) ResponseString(str string) error {
	return r.Response(fastconv.StringToBytes(str))
}

// ResponseJson 发送JSON格式的HTTP响应
// 自动设置Content-Type为application/json
func (r *RequestContext) ResponseJson(data interface{}) error {
	dataBytes, err := jsonx.MarshalToBytes(data)
	if err != nil {
		return err
	}
	r.hertzCtx.Response.Header.Set("Content-Type", "application/json; charset=utf-8")
	if r.status == 0 {
		r.hertzCtx.SetStatusCode(http.StatusOK)
	} else {
		r.hertzCtx.SetStatusCode(r.status)
	}
	_, err = r.hertzCtx.Write(dataBytes)
	return err
}

// ResponseBuiltinJson 发送预定义的JSON格式HTTP响应
// 使用常量定义的响应码获取对应的JSON消息
func (r *RequestContext) ResponseBuiltinJson(code constant.RESPONSE_CODE) error {
	msg := jsonx.GetStaticJsonResponseStr(code)
	r.hertzCtx.Response.Header.Set("Content-Type", "application/json; charset=utf-8")
	if r.status == 0 {
		r.hertzCtx.SetStatusCode(http.StatusOK)
	} else {
		r.hertzCtx.SetStatusCode(r.status)
	}
	_, err := r.hertzCtx.WriteString(msg)
	return err
}

// Event 返回当前的Event对象
// 如果为空则初始化一个新的Event
func (r *RequestContext) Event() *core.Event {
	if r.event == nil {
		r.event = &core.Event{}
	}
	return r.event
}

// ResetEvent 重置当前的Event为传入的Event
func (r *RequestContext) ResetEvent(event *core.Event) {
	r.event = event
}

// EntityEvent 返回当前的EntityEvent对象
// 如果为空则初始化一个新的EntityEvent
func (r *RequestContext) EntityEvent() *core.EntityEvent {
	if r.entityEvent == nil {
		r.entityEvent = &core.EntityEvent{}
	}
	return r.entityEvent
}

// ResetEntityEvent 重置当前的EntityEvent为传入的EntityEvent
func (r *RequestContext) ResetEntityEvent(entityEvent *core.EntityEvent) {
	r.entityEvent = entityEvent
}

// CallChain 返回调用链信息
// 公网端不存在调用链检查，返回空数组
func (r *RequestContext) CallChain() []string {
	return []string{}
}

// Data 返回临时存储的上下文数据
func (r *RequestContext) Data() interface{} {
	return r.tmpData
}

// SetData 设置临时存储的上下文数据
func (r *RequestContext) SetData(data interface{}) {
	r.tmpData = data
}

// OpenSSEResponse 返回一个支持SSE的响应对象
// 用于支持SSE的HTTP响应
// cross参数用于设置跨域访问控制
// 如果cross为true，则设置Access-Control-Allow-Origin为*
// 如果cross为false，则不设置Access-Control-Allow-Origin
func (r *RequestContext) OpenSSEResponse(cross bool) network.ExtWriter {
	r.SetHeader("Content-Type", "text/event-stream")
	r.SetHeader("Cache-Control", "no-cache")
	r.SetHeader("Connection", "keep-alive")
	if cross {
		r.SetHeader("Access-Control-Allow-Origin", "*")
	}
	return r.hertzCtx.GetResponse().GetHijackWriter()
}

// CtxImpl 返回底层的Hertz请求上下文
// 用于需要直接操作底层框架时的类型断言
func (r *RequestContext) CtxImpl() interface{} {
	return r.hertzCtx
}
