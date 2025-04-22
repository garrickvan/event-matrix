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
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/worker/common"
	"github.com/garrickvan/event-matrix/worker/types"
)

// 构建适配框架的上下文
func postEntrance(impl *WorkerPublicServer) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		reqCtx := NewWorkerPublicRequestContext(c, impl.ws)
		err := route(reqCtx, impl.GetUnHandler())
		if err != nil {
			c.String(consts.StatusInternalServerError, "Internal Server Error: "+err.Error())
			return
		}
	}
}

func route(ctx *WorkerPublicRequestContext, unHandle serverx.HandleFunc) error {
	// 获取请求体
	bodyBytes := ctx.Body()
	if len(bodyBytes) == 0 {
		return ctx.SetStatus(http.StatusUnauthorized).ResponseBuiltinJson(constant.UNKNOWN_DATA)
	}

	// 验证事件
	event, status := common.ValidatedEvent(bodyBytes)
	if status != constant.SUCCESS {
		return ctx.SetStatus(http.StatusUnauthorized).ResponseBuiltinJson(status)
	}

	// 获取实体事件
	entityEvent := ctx.Server().DomainCache().EntityEvent(types.PathToEventFromEvent(event))
	if entityEvent == nil {
		return ctx.SetStatus(http.StatusNotFound).ResponseBuiltinJson(constant.EVENT_NOT_EXIST)
	}

	// 如果实体事件是任务执行器或是内部认证，直接禁止调用
	if entityEvent.ExecutorType == constant.TASK_EXECUTOR || entityEvent.AuthType == constant.INTERNAL_AUTH {
		return ctx.SetStatus(http.StatusForbidden).ResponseBuiltinJson(constant.FORBIDDEN_CALL)
	}

	// 获取事件URL，根据URL获取对应的执行器
	eventUrl := event.GetUniqueLabel()
	if funz, found := ctx.Server().FindWorkerExecutor(eventUrl); found && funz != nil {
		// 获取用户ID
		userId, status := common.GetUserId(ctx, event, entityEvent.AuthType == constant.USER_AUTH, false)
		if status != constant.SUCCESS {
			return ctx.SetStatus(http.StatusOK).ResponseBuiltinJson(status)
		}
		// 注入上下文
		ctx.uid = userId
		ctx.ResetEvent(event)
		ctx.ResetEntityEvent(entityEvent)
		return common.HandleExecutor(funz, ctx)
	}
	return unHandle(ctx)
}

func unHandle(s *WorkerPublicServer) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		reqCtx := NewWorkerPublicRequestContext(c, s.ws)
		uf := s.GetUnHandler()
		if uf != nil {
			uf(reqCtx)
		} else {
			c.String(consts.StatusNotFound, "Event Not Implemented")
		}
	}
}
