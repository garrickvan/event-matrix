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
	"net/http"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/serverx/gnetx"
	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/common"
	"github.com/garrickvan/event-matrix/worker/intranet/controller"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/panjf2000/gnet/v2"
	"github.com/spf13/cast"
)

func routeEntrance(rp serverx.RequestPacket, con gnet.Conn, iSvr interface{}) serverx.ResponsePacket {
	var svr *WorkerIntranetServer
	if s, ok := iSvr.(*WorkerIntranetServer); !ok {
		return &gnetx.ResponsePacketImpl{
			StatusCode:  http.StatusInternalServerError,
			ContentType: serverx.CONTENT_TYPE_STRING,
			Payload:     "invalid intranet server type",
		}
	} else {
		svr = s
	}
	gc := NewWorkerIntranetRequestContext(con, rp, svr)
	eventType := types.INTRANET_EVENT_TYPE(0)
	eventTypeData := cast.ToInt32(rp.Extend())
	if types.IsIntranetEventType(eventTypeData) {
		eventType = types.INTRANET_EVENT_TYPE(eventTypeData)
	}
	// 事件处理
	if eventType == types.W_T_W_EVENT_CALL {
		event, status := common.ValidatedEvent(fastconv.StringToBytes(rp.TemporaryData()))
		if status != constant.SUCCESS {
			return &gnetx.ResponsePacketImpl{
				StatusCode:  http.StatusBadRequest,
				ContentType: serverx.CONTENT_TYPE_STRING,
				Payload:     string(status),
			}
		}
		if event == nil {
			return &gnetx.ResponsePacketImpl{
				StatusCode:  http.StatusBadRequest,
				ContentType: serverx.CONTENT_TYPE_STRING,
				Payload:     string(constant.UNSUPPORTED_EVENT),
			}
		}
		// 获取实体事件
		entityEvent := svr.ws.DomainCache().EntityEvent(types.PathToEventFromEvent(event))
		if entityEvent == nil {
			return &gnetx.ResponsePacketImpl{
				StatusCode:  http.StatusNotFound,
				ContentType: serverx.CONTENT_TYPE_STRING,
				Payload:     string(constant.UNSUPPORTED_EVENT),
			}
		}
		// 鉴权并获取用户ID
		userId, status := common.GetUserId(gc, event, entityEvent.AuthType == constant.USER_AUTH, true)
		if status != constant.SUCCESS {
			return &gnetx.ResponsePacketImpl{
				StatusCode:  http.StatusUnauthorized,
				ContentType: serverx.CONTENT_TYPE_STRING,
				Payload:     string(status),
			}
		}
		// 注入上下文
		gc.uid = userId
		gc.ResetEvent(event)
		gc.ResetEntityEvent(entityEvent)
		eventUrl := event.GetUniqueLabel()
		// 处理任务
		if entityEvent.ExecutorType == constant.TASK_EXECUTOR {
			if task, found := gc.Server().FindWorkerTaskExecutor(eventUrl); found && task != nil {
				err := common.HandleTask(task, gc)
				if err != nil {
					logx.Error("internal event task error: %v", err)
					return &gnetx.ResponsePacketImpl{
						StatusCode:  http.StatusInternalServerError,
						ContentType: serverx.CONTENT_TYPE_STRING,
						Payload:     "internal event task error",
					}
				}
				return gc.GetRespon()
			}
		}
		// 处理执行器
		if funz, found := gc.Server().FindWorkerExecutor(eventUrl); found && funz != nil {
			err := common.HandleExecutor(funz, gc)
			if err != nil {
				logx.Error("internal event exec error: %v", err)
				return &gnetx.ResponsePacketImpl{
					StatusCode:  http.StatusInternalServerError,
					ContentType: serverx.CONTENT_TYPE_STRING,
					Payload:     "internal event exec error",
				}
			}
			return gc.GetRespon()
		}
		// 未找到执行器或任务, 返回默认未处理信息
		uf := svr.GetUnHandler()
		if uf == nil {
			uf(gc)
		}
		return gc.GetRespon()
	}
	// 传递特定事件的配置数据
	if eventType == types.G_T_W_UPDATE_RECORD_FOR_DATA_MGR {
		gc.SetData(gc.svr.cfg.NotAcceptUpdateRecordEventFromGateway)
	}
	// 内置事件处理
	err := controller.RootRouter(eventType, rp.TemporaryData(), gc, svr.GetUnHandler())
	if err != nil {
		return &gnetx.ResponsePacketImpl{
			StatusCode: http.StatusInternalServerError,
		}
	}
	return gc.GetRespon()
}
