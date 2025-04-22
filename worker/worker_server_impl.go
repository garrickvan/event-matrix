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

package worker

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/common/controller"
	"github.com/garrickvan/event-matrix/worker/intranet/dispatcher"
	"github.com/garrickvan/event-matrix/worker/types"
)

// ServerId 返回服务器ID
func (ws *TwoWayWorkerServer) ServerId() string {
	return ws.cfg.ServerId
}

// IntranetSecret 返回内域密钥
func (ws *TwoWayWorkerServer) IntranetSecret() string {
	return ws.cfg.IntranetSecret
}

// IntranetSecretAlgor 返回内域密钥算法
func (ws *TwoWayWorkerServer) IntranetSecretAlgor() string {
	return ws.cfg.IntranetSecretAlgor
}

// GatewayIntranetEndpoint 返回网关内域端点
func (ws *TwoWayWorkerServer) GatewayIntranetEndpoint() string {
	return ws.cfg.GatewayIntranetEndpoint
}

// SharedConfigure 获取共享配置
func (ws *TwoWayWorkerServer) SharedConfigure(sid string) *core.SharedConfigure {
	if conf, has := ws.sharedConfigures.Load(sid); has {
		return conf.(*core.SharedConfigure)
	}
	cfgs := dispatcher.LoadSharedCfgFromGateway([]string{sid})
	if len(cfgs) == 0 {
		return nil
	}
	ws.sharedConfigures.Store(sid, cfgs[sid])
	return cfgs[sid]
}

// GetSharedConfigureChangeHandler 获取共享配置变更处理器
func (ws *TwoWayWorkerServer) GetSharedConfigureChangeHandler() types.OnSharedConfigureChangeFunc {
	return ws.onSharedConfigureChange
}

// Repo 返回存储库实例
func (ws *TwoWayWorkerServer) Repo() types.Repository {
	return ws.repo
}

// Cache 返回默认缓存实例
func (ws *TwoWayWorkerServer) Cache() types.DefaultCache {
	return ws.cache
}

// DomainCache 返回领域缓存实例
func (ws *TwoWayWorkerServer) DomainCache() types.DomainCache {
	return ws.domainCache
}

// RegisterPlugin 注册插件
func (ws *TwoWayWorkerServer) RegisterPlugin(plugin types.PluginWorker) {
	pluginCodes := plugin.ReceiveCodes()
	for _, code := range pluginCodes {
		ws.plugins[code] = plugin
	}
}

// RuleEngineMgr 返回规则引擎管理器
func (ws *TwoWayWorkerServer) RuleEngineMgr() types.RuleEngineManager {
	return ws.ruleEngineMgr
}

// GetWorkerByEvent 根据路径获取工作者
func (ws *TwoWayWorkerServer) GetWorkerByEvent(p types.PathToEntity) *types.Worker {
	w := types.Worker{
		Project:      p.Project,
		VersionLabel: p.Version,
		Context:      p.Context,
		Entity:       p.Entity,
	}
	worker, has := ws.entityMapToWorkers[w.GetVersionEntityLabel()]
	if !has {
		return nil
	}
	return worker
}

// RegisterWorker 注册工作者
func (ws *TwoWayWorkerServer) RegisterWorker(w *types.Worker) error {
	if w.CfgKey != "" {
		err := ws.repo.AddDBFromSharedConfig(w.CfgKey)
		if err != nil {
			return errors.New("初始化数据库失败: " + err.Error())
		}
	}
	resp, err := ws.rigsterWorkerToGateway(w)
	if err != nil {
		ws.addFailedWorker(w)
		return errors.New("添加到网关失败: " + err.Error())
	}
	if strings.TrimSpace(resp) == string(constant.SUCCESS) {
		if ws.hasWorkerId(w.ID) {
			logx.Log().Warn("重复注册（一般由重复调用导致）: " + w.ID + "  " + utils.StructToLogStr(w))
		}
		ws.addWorker(w)
		if w.SyncSchema {
			ws.repo.SyncSchema(w)
		}
		ws.setupRouter(w)
		ws.ruleEngineMgr.AddRuleEngine(w)
		ws.remvoeFailedWorker(w.ID)
		dispatcher.ReportConfigUsedBy(w.CfgKey, w.ID)
		dispatcher.ReportConfigUsedBy(ws.cfgKey, w.ID)
	} else {
		ws.addFailedWorker(w)
		return errors.New(resp)
	}
	return nil
}

// hasWorkerId 判断是否已存在该工作者ID
func (ws *TwoWayWorkerServer) hasWorkerId(workerId string) bool {
	_, has := ws.workerIds[workerId]
	return has
}

// addFailedWorker 添加失败的工作者
func (ws *TwoWayWorkerServer) addFailedWorker(w *types.Worker) {
	if _, has := ws.failedWorkers[w.ID]; has {
		return
	}
	ws.failedWorkers[w.ID] = w
}

// remvoeFailedWorker 移除失败的工作者
func (ws *TwoWayWorkerServer) remvoeFailedWorker(workerID string) {
	if _, has := ws.failedWorkers[workerID]; has {
		delete(ws.failedWorkers, workerID)
	}
}

// startFailedWorkersDaemon 启动失败工作者守护进程
func (ws *TwoWayWorkerServer) startFailedWorkersDaemon() {
	go func() {
		// 每隔5秒重新注册失败的worker
		for {
			time.Sleep(5 * time.Second)
			for _, w := range ws.failedWorkers {
				err := ws.RegisterWorker(w)
				if err != nil {
					logx.Log().Error(err.Error())
				}
			}
		}
	}()
}

// addWorker 添加工作者
func (ws *TwoWayWorkerServer) addWorker(worker *types.Worker) {
	ws.workerIds[worker.ID] = true
	ws.entityMapToWorkers[worker.GetVersionEntityLabel()] = worker
}

// rigsterWorkerToGateway 注册工作者到网关
func (ws *TwoWayWorkerServer) rigsterWorkerToGateway(w *types.Worker) (string, error) {
	w.ServerId = ws.ServerId()
	w.PublicEndpoint = fmt.Sprintf("%s:%d", ws.cfg.PublicHost, ws.cfg.PublicPort)
	w.IntranetEndpoint = fmt.Sprintf("%s:%d", ws.cfg.IntranetHost, ws.cfg.IntranetPort)
	w.HeartbeatGap = ws.cfg.HeartbeatReportGap
	w.UtcOffset = utils.GetCurrentTimezoneOffset()
	mode := strings.ToUpper(ws.cfg.WorkMode)
	if mode == "C" || mode == "COMMAND" {
		w.Mode = constant.COMMAND_MODE
	} else if mode == "Q" || mode == "QUERY" {
		w.Mode = constant.QUERY_MODE
	} else {
		w.Mode = constant.COMMAND_MODE
	}
	w.GenID()
	w.BuildExecutors()
	if w.HeartbeatGap < 3 {
		w.HeartbeatGap = 30 // 不设就默认30秒
	}
	resp, err := dispatcher.Event(ws.cfg.GatewayIntranetEndpoint, types.W_T_G_REGISTER, w, nil)
	if err != nil || resp.Status() != http.StatusOK {
		return "", err
	}
	return strings.Clone(resp.TemporaryData()), nil
}

// FindPlugin 查找插件
func (ws *TwoWayWorkerServer) FindPlugin(pluginType types.INTRANET_EVENT_TYPE) (types.PluginWorker, bool) {
	if plugin, has := ws.plugins[pluginType]; has {
		return plugin, true
	}
	return nil, false
}

// FindWorkerExecutor 查找工作者执行器
func (ws *TwoWayWorkerServer) FindWorkerExecutor(name string) (types.WorkerExecutor, bool) {
	if executor, has := ws.routers[name]; has {
		return executor, true
	}
	return nil, false
}

// FindWorkerTaskExecutor 查找工作者任务执行器
func (ws *TwoWayWorkerServer) FindWorkerTaskExecutor(name string) (types.WorkerTaskExecutor, bool) {
	if executor, has := ws.tasks[name]; has {
		return executor, true
	}
	return nil, false
}

// Intercepts 返回拦截器列表
func (ws *TwoWayWorkerServer) Intercepts() []types.Intercept {
	return ws.interceptors
}

// Filters 返回过滤器列表
func (ws *TwoWayWorkerServer) Filters() []types.Filter {
	return ws.filters
}

// setupRouter 设置工作者路由
func (ws *TwoWayWorkerServer) setupRouter(w *types.Worker) {
	events := ws.domainCache.EntityEvents(types.PathToEntityFromWorker(w))
	if len(events) < 1 && w.VersionLabel != constant.INITIAL_VERSION {
		// 没有找到事件，不设置路由
		logx.Debug("没有找到事件: " + w.Project + "." + w.Context + "." + w.Entity + "@" + w.VersionLabel)
		return
	}
	for _, event := range events {
		var builder strings.Builder
		builder.Write([]byte(w.Project))
		builder.Write([]byte("."))
		builder.Write([]byte(w.Context))
		builder.Write([]byte("."))
		builder.Write([]byte(w.Entity))
		builder.Write([]byte("->"))
		builder.Write([]byte(event.Code))
		builder.Write([]byte("@"))
		builder.Write([]byte(w.VersionLabel))
		url := builder.String()
		if event.ExecutorType == constant.BUILD_IN_EXECUTOR {
			switch event.Executor {
			case "query":
				ws.routers[url] = controller.QueryExecutor
			case "create":
				ws.routers[url] = controller.CreateExecutor
			case "update":
				ws.routers[url] = controller.UpdateExecutor
			case "delete":
				ws.routers[url] = controller.DeleteExecutor
			case "restore":
				ws.routers[url] = controller.RestoreExecutor
			case "sql":
				ws.routers[url] = controller.SqlExecutor
			default:
				logx.Log().Warn("没有找到内置执行器: " + event.Executor)
			}
		} else if event.ExecutorType == constant.CUSTOM_EXECUTOR {
			fnz, found := w.FindCustomExecutor(event.Executor)
			if !found {
				logx.Log().Error("没有找到自定义执行器: " + event.Executor)
				continue
			}
			ws.routers[url] = fnz
		} else if event.ExecutorType == constant.TASK_EXECUTOR {
			fnz, found := w.FindTaskExecutor(event.Executor)
			if !found {
				logx.Log().Error("没有找到自定义执行器: " + event.Executor)
				continue
			}
			ws.tasks[url] = fnz
		} else {
			logx.Log().Error("没有找到执行器: " + event.Executor)
		}
	}
}

// RegisterInterceptor 注册拦截器
func (ws *TwoWayWorkerServer) RegisterInterceptor(interceptor types.Intercept) {
	ws.interceptors = append(ws.interceptors, interceptor)
}

// RegisterFilter 注册过滤器
func (ws *TwoWayWorkerServer) RegisterFilter(filter types.Filter) {
	ws.filters = append(ws.filters, filter)
}

// HasWorker 判断是否存在指定ID的工作者
func (ws *TwoWayWorkerServer) HasWorker(workerId string) bool {
	_, exists := ws.workerIds[workerId]
	return exists
}
