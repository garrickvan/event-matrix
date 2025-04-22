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

package gnetimpl

import (
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/serverx/gnetx"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/worker/common"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/panjf2000/gnet/v2"
)

type WorkerIntranetRequestContext struct {
	gnetx.RequestContext

	svr         *WorkerIntranetServer
	uid         string                 // 用户ID
	attrs       []core.EntityAttribute // 实体属性列表
	eventParams []core.EventParam      // 事件参数列表
	params      map[string]interface{} // 请求参数
}

// NewWorkerIntranetRequestContext 创建一个新的WorkerIntranetRequestContext实例
func NewWorkerIntranetRequestContext(conn gnet.Conn, req serverx.RequestPacket, svr *WorkerIntranetServer) *WorkerIntranetRequestContext {
	ctx := &WorkerIntranetRequestContext{
		svr: svr,
	}
	ctx.RequestContext = *gnetx.NewRequestContext(conn, req)
	return ctx
}

// UserId 返回当前请求的用户ID
func (c *WorkerIntranetRequestContext) UserId() string {
	return c.uid
}

func (c *WorkerIntranetRequestContext) ValidatedParams() (
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

// WorkerServer 返回关联的Worker服务器实例
func (c *WorkerIntranetRequestContext) Server() types.WorkerServer {
	return c.svr.ws
}
