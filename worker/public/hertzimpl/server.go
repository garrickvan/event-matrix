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
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/serverx/hertzx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/spf13/cast"
)

type WorkerPublicServer struct {
	*hertzx.PublicServer

	cfg *types.WorkerServerConfig
	ws  types.WorkerServer
}

func NewWorkerPublicServer(cfg *types.WorkerServerConfig, ws types.WorkerServer) *WorkerPublicServer {
	hlog.SetLevel(hlog.LevelWarn)
	wps := &WorkerPublicServer{
		ws:  ws,
		cfg: cfg,
	}
	wps.PublicServer = hertzx.NewPublicServer(cfg.PublicPort, cfg.ServerId)
	wps.setMiddleware()
	return wps
}

func (s *WorkerPublicServer) setMiddleware() {
	var hertzSvr *server.Hertz
	if hz, ok := s.Impl().(*server.Hertz); !ok || hz == nil {
		logx.Error("Failed to set middleware, hertz is not initialized")
		return
	} else {
		hertzSvr = hz
	}
	// 开发模式日志
	if s.cfg.Mode == constant.DEV {
		hertzSvr.Use(debugMiddleware())
	}
	// 接管所有路由
	hertzSvr.Any("/*path",
		func(c context.Context, ctx *app.RequestContext) {
			// 插件请求
			pluginType := ctx.Request.Header.Get(types.PLUGIN_HEADER)
			if pluginType != "" {
				t := cast.ToUint16(pluginType)
				if t == uint16(types.UNKNOWN_EVENT) {
					logx.Error("Plugin type is invalid")
					ctx.AbortWithStatus(consts.StatusNotFound)
				} else {
					handlePluginRequest(s, types.INTRANET_EVENT_TYPE(t))(c, ctx)
				}
				return
			}
			// 正常请求
			switch string(ctx.Request.Method()) {
			case consts.MethodPost:
				postEntrance(s)(c, ctx)
			default:
				unHandle(s)(c, ctx)
			}
		})
}

// 开发模式日志中间件
func debugMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 调试日志中间件
		method := string(c.Method())
		url := string(c.Request.URI().Path())
		currentTime := time.Now().Format("01-02 15:04:05")
		// 打印请求日志
		if method == consts.MethodPost {
			event, _ := core.NewEventFromBytes(c.Request.Body())
			if !event.IsEmpty() {
				logx.Debug("----------------------------------------")
				logx.Debug(fmt.Sprintf("%s URL:【%s】  Method:【%s】  Event:【%s】", currentTime, url, method, event.GetUniqueLabel()))
			} else {
				logx.Debug("----------------------------------------")
				logx.Debug(fmt.Sprintf("%s URL:【%s】  Method:【%s】", currentTime, url, method))
			}
		} else if method == consts.MethodOptions {

		} else {
			logx.Debug("----------------------------------------")
			logx.Debug(fmt.Sprintf("%s URL:【%s】  Method:【%s】", currentTime, url, method))
		}
		// 开启CORS
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		if method == consts.MethodOptions {
			c.AbortWithStatus(consts.StatusNoContent)
			return
		}
		c.Next(ctx)
	}
}

func handlePluginRequest(s *WorkerPublicServer, ptype types.INTRANET_EVENT_TYPE) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		plugin, found := s.ws.FindPlugin(ptype)
		if !found {
			logx.Error("Plugin not found")
			c.AbortWithStatus(consts.StatusNotFound)
			return
		}
		reqCtx := NewWorkerPublicRequestContext(c, s.ws)
		plugin.Handle(reqCtx, ptype)
	}
}
