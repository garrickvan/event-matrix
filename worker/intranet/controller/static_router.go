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

package controller

import (
	"net/http"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
)

// 内域事件根路由
func RootRouter(eventType types.INTRANET_EVENT_TYPE, payload string, ctx types.WorkerContext, unhandle serverx.HandleFunc) error {
	switch eventType {
	case types.G_T_W_CHECK_WORKER:
		return checkWorkerHandler(ctx, payload)
	case types.G_T_W_GET_LOADE_RATE:
		return getLoadRateHandler(ctx, payload)
	case types.G_T_W_RULE_UPDATE:
		return ctx.Server().RuleEngineMgr().HandleRuleUpdate(ctx, payload)
	case types.G_T_W_SHARED_CONFIGURE_CHANGE:
		return handleSharedConfigureChange(ctx, payload)
	case types.G_T_W_ENTITY_LIST_FOR_DATA_MGR:
		return OnEntityListForDataMgrHandler(ctx, payload)
	case types.G_T_W_UPDATE_RECORD_FOR_DATA_MGR:
		return OnUpdateRecordForDataMgrHandler(ctx, payload)
	case types.G_T_W_RESET_DOMAIN_CACHE:
		return resetDomainCacheHandler(ctx, payload)
	default:
		if eventType >= types.WORKER_INTERNAL_PLUGIN {
			// 处理插件事件
			plugin, found := ctx.Server().FindPlugin(eventType)
			if found {
				err := plugin.Handle(ctx, eventType)
				if err != nil {
					return ctx.SetStatus(http.StatusInternalServerError).ResponseString(err.Error())
				}
				return nil
			}
		}
		return unhandle(ctx)
	}
}

// handleSharedConfigureChange 处理共享配置变更
func handleSharedConfigureChange(ctx types.WorkerContext, payload string) error {
	handle := ctx.Server().GetSharedConfigureChangeHandler()
	if handle == nil {
		return ctx.SetStatus(http.StatusNotImplemented).Response([]byte(constant.NOT_IMPLEMENTED))
	}
	if payload == "" {
		return ctx.SetStatus(http.StatusInternalServerError).Response([]byte(constant.EMPTY_DATA))
	}
	cfg := core.SharedConfigure{}
	if err := jsonx.UnmarshalFromStr(payload, &cfg); err != nil {
		return ctx.SetStatus(http.StatusInternalServerError).Response([]byte(constant.INVALID_PARAM))
	}
	return handle(ctx.Server(), &cfg)
}

// resetDomainCacheHandler 重置域缓存
func resetDomainCacheHandler(ctx types.WorkerContext, payload string) error {
	logx.Debug("接收到重置缓存请求: " + ctx.Server().ServerId())
	ctx.Server().DomainCache().Impl().Flush()
	return ctx.SetStatus(http.StatusOK).Response([]byte(constant.SUCCESS))
}
