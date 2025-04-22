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

package main

import (
	"github.com/garrickvan/event-matrix/utils"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker"
	"github.com/garrickvan/event-matrix/worker/plugins/aiassistant"
	"github.com/garrickvan/event-matrix/worker/plugins/logcenter"
	"github.com/garrickvan/event-matrix/worker/plugins/taskcenter"
	"github.com/garrickvan/event-matrix/worker/types"
)

func main() {
	// 获取环境变量
	secret := utils.GetEnv("IntranetSecret")
	algor := utils.GetEnv("IntranetSecretAlgor")
	gatewayEndpoint := utils.GetEnv("GatewayIntranetEndpoint")

	// 初始化配置
	svr := worker.NewTwoWayWorkerServer(worker.TwoWayWorkerServerSettings{
		CfgKey:                  "demo_worker",
		IntranetSecret:          secret,
		IntranetSecretAlgor:     algor,
		GatewayIntranetEndpoint: gatewayEndpoint,
	})
	// 初始化日志中心插件
	lc := logcenter.NewLogCenter(svr, "sqlite_runtime_log", "sqlite_event_log")
	if err := lc.Setup(); err != nil {
		logx.Log().Error(err.Error())
	}
	// 初始化任务中心插件
	tc := taskcenter.NewTaskCenter(svr, "sql_task", 500) // 最高同时调起500个异步任务
	if err := tc.Setup(); err != nil {
		logx.Log().Error(err.Error())
	}
	// 初始化AI助手插件
	ac := aiassistant.NewAiAssistantCenter(svr, "")
	if err := ac.Setup(); err != nil {
		logx.Log().Error(err.Error())
	}
	// 注册worker
	if err := svr.RegisterWorker(types.NewWorker("demo", "0.0.1", "user", "user_info", "sql_demo", 60)); err != nil {
		logx.Log().Error(err.Error())
	}
	// 启动日志提交
	cf := svr.Cfg()
	logcenter.NewLogDaemonSubmitter(cf.LogLocation).StartDaemon()
	// 启动HTTP服务
	svr.Start()
}
