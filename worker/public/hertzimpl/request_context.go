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

package hertzimpl

import (
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/serverx/hertzx"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/worker/common"
	"github.com/garrickvan/event-matrix/worker/types"
)

// WorkerPublicRequestContext 实现了 serverx.RequestContext 接口，用于在 Hertz 框架中处理请求上下文
type WorkerPublicRequestContext struct {
	hertzx.RequestContext

	uid         string                 // 用户ID
	ws          types.WorkerServer     // 工作服务器实例
	attrs       []core.EntityAttribute // 实体属性列表
	params      map[string]interface{} // 请求参数
	eventParams []core.EventParam      // 事件参数列表
}

// NewWorkerPublicRequestContext 创建并返回一个新的 WorkerPublicRequestContext 实例
func NewWorkerPublicRequestContext(hertzCtx *app.RequestContext, svr types.WorkerServer) *WorkerPublicRequestContext {
	hc := &WorkerPublicRequestContext{
		ws: svr,
	}
	hc.RequestContext = *hertzx.NewRequestContext(hertzCtx)
	return hc
}

// UserId 返回当前请求的用户ID
func (c *WorkerPublicRequestContext) UserId() string {
	return c.uid
}

// ValidatedParams 解析并验证请求参数，返回实体属性列表、事件参数列表、请求参数、解析结果
func (c *WorkerPublicRequestContext) ValidatedParams() (
	entityAttrs []core.EntityAttribute,
	entityEventParams []core.EventParam,
	params map[string]interface{},
	result *jsonx.JsonResponse,
) {
	// 缓存已解析的参数
	if c.attrs != nil && c.eventParams != nil && c.params != nil {
		return c.attrs, c.eventParams, c.params, nil
	}
	c.attrs, c.eventParams, c.params, result = common.ParseAndValidateParams(c)
	return c.attrs, c.eventParams, c.params, result
}

// Server 返回关联的Worker服务器实例
func (c *WorkerPublicRequestContext) Server() types.WorkerServer {
	return c.ws
}
