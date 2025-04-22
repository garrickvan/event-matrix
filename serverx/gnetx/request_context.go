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

// Package gnetx 提供基于gnet的网络通信请求上下文实现
package gnetx

import (
	"net/http"
	"strings"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils"
	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/panjf2000/gnet/v2"
)

// RequestContext 结构体用于封装请求上下文信息
// 实现了serverx.RequestContext接口，提供内域通信请求处理的核心功能
type RequestContext struct {
	conn        gnet.Conn            // 网络连接对象
	ip          string               // 客户端IP地址
	status      int                  // HTTP状态码
	body        []byte               // 请求体内容
	event       *core.Event          // 核心事件对象
	entityEvent *core.EntityEvent    // 实体事件对象
	requestType serverx.CONTENT_TYPE // 请求类型
	response    *ResponsePacketImpl  // 响应包
	tmpData     interface{}          // 临时数据存储
	callChains  string               // 调用链信息，用于防止循环调用
}

// NewRequestContext 创建一个新的RequestContext实例
//
// 参数：
//   - conn: gnet网络连接对象
//   - reqPkg: 请求包数据
//
// 返回值：
//   - *RequestContext: 新创建的请求上下文
func NewRequestContext(conn gnet.Conn, req serverx.RequestPacket) *RequestContext {
	var ip string
	var body []byte

	if req != nil {
		ip = strings.Clone(req.IP())
		body = fastconv.StringToBytes(req.TemporaryData()) // 底层用了sys.Pool的buffer，临时数据仅在当前请求的生命周期内有效，长期保存需要拷贝
	}
	return &RequestContext{
		conn:        conn,
		ip:          ip,
		status:      http.StatusNotImplemented,
		body:        body,
		event:       nil,
		entityEvent: nil,
		requestType: req.Type(),
		tmpData:     nil,
		callChains:  req.CallChains(),
		response: &ResponsePacketImpl{
			StatusCode:  http.StatusNotImplemented,
			ContentType: serverx.CONTENT_TYPE_STRING,
			Payload:     "",
		},
	}
}

// IP 返回请求的客户端IP地址
// 如果未设置则从连接中提取
func (r *RequestContext) IP() string {
	if r.ip == "" && r.conn != nil {
		remoteAddr := r.conn.RemoteAddr()
		if remoteAddr != nil {
			r.ip = utils.ExtractIp(remoteAddr.String())
		} else {
			r.ip = "" // 或者返回一个默认值
		}
	}
	return r.ip
}

// Path 返回请求的路径
// 如果存在事件对象，则返回事件的实体URL
func (r *RequestContext) Path() string {
	if r.event != nil {
		return r.event.GetEntityUrl()
	}
	return ""
}

// Body 返回请求体内容的字节数组
func (r *RequestContext) Body() []byte {
	return r.body
}

// BodyType 返回请求体的内容类型
// 从请求包中获取负载类型
func (r *RequestContext) BodyType() serverx.CONTENT_TYPE {
	return r.requestType
}

// IsJsonBody 判断请求体是否为JSON格式
func (r *RequestContext) IsJsonBody() bool {
	return r.BodyType() == serverx.CONTENT_TYPE_JSON
}

// Header 返回请求头中指定键的值
// 内部通信不支持HTTP头，返回空字符串
func (r *RequestContext) Header(key string) string {
	logx.Debug("header is not supported in internal request context")
	return ""
}

// SetHeader 设置响应头中的键值对
// 内部通信不支持HTTP头，此方法不执行任何操作
func (r *RequestContext) SetHeader(key, value string) {
	logx.Debug("header is not supported in internal request context")
}

// SetStatus 设置响应的HTTP状态码
// 返回当前上下文以支持链式调用
func (r *RequestContext) SetStatus(code int) serverx.RequestContext {
	r.status = code
	return r
}

// Response 设置响应内容为字节数组
// 自动转换为字符串类型的响应
func (r *RequestContext) Response(strBytes []byte) error {
	r.response.StatusCode = r.status
	r.response.ContentType = serverx.CONTENT_TYPE_STRING
	r.response.Payload = fastconv.BytesToString(strBytes)
	return nil
}

// ResponseString 设置响应内容为字符串
// 设置响应类型为STRING
func (r *RequestContext) ResponseString(str string) error {
	r.response.StatusCode = r.status
	r.response.ContentType = serverx.CONTENT_TYPE_STRING
	r.response.Payload = str
	return nil
}

// ResponseJson 设置响应内容为JSON格式
// 将对象序列化为JSON字符串，并设置响应类型为JSON
func (r *RequestContext) ResponseJson(data interface{}) error {
	if dataStr, err := jsonx.MarshalToStr(data); err != nil {
		return err
	} else {
		r.response.StatusCode = r.status
		r.response.ContentType = serverx.CONTENT_TYPE_JSON
		r.response.Payload = dataStr
	}
	return nil
}

// ResponseBuiltinJson 设置响应内容为预定义的JSON格式
// 使用常量定义的响应码获取对应的JSON消息
func (r *RequestContext) ResponseBuiltinJson(code constant.RESPONSE_CODE) error {
	msg := jsonx.GetStaticJsonResponseStr(code)
	r.response.StatusCode = r.status
	r.response.ContentType = serverx.CONTENT_TYPE_JSON
	r.response.Payload = msg
	return nil
}

// Event 返回请求关联的事件对象
// 如果未设置则创建一个新的事件对象
func (r *RequestContext) Event() *core.Event {
	if r.event == nil {
		r.event = &core.Event{}
	}
	return r.event
}

// ResetEvent 重置请求关联的事件对象
// 用于设置特定的事件对象
func (r *RequestContext) ResetEvent(event *core.Event) {
	r.event = event
}

// EntityEvent 返回请求关联的实体事件对象
// 如果未设置则创建一个新的实体事件对象
func (r *RequestContext) EntityEvent() *core.EntityEvent {
	if r.entityEvent == nil {
		r.entityEvent = &core.EntityEvent{}
	}
	return r.entityEvent
}

// ResetEntityEvent 重置请求关联的实体事件对象
// 用于设置特定的实体事件对象
func (r *RequestContext) ResetEntityEvent(entityEvent *core.EntityEvent) {
	r.entityEvent = entityEvent
}

// CallChain 返回请求的调用链
// 用于防止内部服务之间的循环调用
func (r *RequestContext) CallChain() []string {
	if r.callChains != "" {
		return fastconv.SafeSplit(r.callChains, constant.SPLIT_CHAR)
	}
	return []string{}
}

// Data 返回请求的临时数据
func (r *RequestContext) Data() interface{} {
	return r.tmpData
}

// SetData 设置请求的临时数据
// 用于在请求处理过程中存储临时状态
func (r *RequestContext) SetData(data interface{}) {
	r.tmpData = data
}

// CtxImpl 返回请求的底层实现对象
// 在内部通信中为gnet连接对象
func (r *RequestContext) CtxImpl() interface{} {
	return r.conn
}

// GetRespon 获取响应消息对象
// 用于获取当前设置的响应内容
func (r *RequestContext) GetRespon() serverx.ResponsePacket {
	return r.response
}
