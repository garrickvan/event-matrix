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

package types

import (
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/rulego/rulego/api/types"
)

// OnSharedConfigureChangeFunc 是共享配置变更回调函数的类型定义
type OnSharedConfigureChangeFunc func(ws WorkerServer, cfg *core.SharedConfigure) error

// WorkerServer 定义了工作服务器的核心接口。
// 这个接口包含了所有与工作服务器相关的功能和方法。
type WorkerServer interface {
	serverx.Server

	// IntranetSecret 返回内部网络通信的密钥。
	IntranetSecret() string
	// IntranetSecretAlgor 返回内部网络通信密钥的算法。
	IntranetSecretAlgor() string
	// GatewayIntranetEndpoint 返回网关的内部网络端点。
	GatewayIntranetEndpoint() string

	// SharedConfigure 根据服务ID获取共享配置。
	SharedConfigure(sid string) *core.SharedConfigure
	// OnSharedConfigureChange 注册共享配置变更的回调函数。
	GetSharedConfigureChangeHandler() OnSharedConfigureChangeFunc

	// RegisterWorker 注册一个工作实例。
	RegisterWorker(w *Worker) error
	// HasWorker 判断是否存在某个已注册的工作实例。
	HasWorker(workerId string) bool
	// GetWorkerByEvent 根据事件路径获取相应的工作实例。
	GetWorkerByEvent(e PathToEntity) *Worker
	// FindWorkerExecutor 根据名称查找工作执行器。
	FindWorkerExecutor(name string) (WorkerExecutor, bool)
	// FindWorkerTaskExecutor 根据名称查找工作任务执行器。
	FindWorkerTaskExecutor(name string) (WorkerTaskExecutor, bool)

	// RegisterPlugin 注册一个插件工作实例。
	RegisterPlugin(plugin PluginWorker)
	// FindPlugin 根据事件类型查找插件工作实例。
	FindPlugin(pluginType INTRANET_EVENT_TYPE) (PluginWorker, bool)

	// Intercepts 返回所有拦截器列表。
	Intercepts() []Intercept
	// Filters 返回所有过滤器列表。
	Filters() []Filter

	// RuleEngineMgr 返回规则引擎管理器。
	RuleEngineMgr() RuleEngineManager
	// Repo 返回数据仓库实例。
	Repo() Repository
	// Cache 返回默认缓存实例。
	Cache() DefaultCache
	// DomainCache 返回域名缓存实例。
	DomainCache() DomainCache
}

// WorkerContext 定义了工作上下文的核心接口。
// 这个接口包含了所有与工作上下文相关的功能和方法。
type WorkerContext interface {
	serverx.RequestContext

	// 验证并解析参数，返回实体属性、事件参数、参数映射以及JSON响应结果。
	ValidatedParams() (
		entityAttrs []core.EntityAttribute,
		entityEventParams []core.EventParam,
		params map[string]interface{},
		result *jsonx.JsonResponse,
	)

	// UserId 返回用户ID。
	UserId() string

	// WorkerServer 返回工作服务器实例。
	Server() WorkerServer
}

/**
 * WorkerExecutor 是事件执行方法的类型定义。
 * @param wc WorkerContext 工作上下文
 * @return *jsonx.JsonResponse JSON响应结果，为空时表示自行处理返回形式
 */
type WorkerExecutor func(wc WorkerContext) (*jsonx.JsonResponse, int)

/**
 * WorkerTaskExecutor 是工作任务执行方法的类型定义。
 * @param wc WorkerContext 工作上下文
 * @return core.TaskStatus 任务状态
 */
type WorkerTaskExecutor func(wc WorkerContext) core.TaskStatus

/**
 * Intercept 是工作路由拦截器的类型定义。
 * 拦截器用于在某些事件前执行业务处理，例如特殊的权限验证或返回数据缓存等。
 * @param wc WorkerContext 工作上下文
 * @return stop 是否停止执行后续插件，true表示停止，将直接返回响应，达到拦截效果
 */
type Intercept func(wc WorkerContext) (stop bool)

/**
 * Filter 是工作路由过滤器的类型定义。
 * 过滤器用于在执行成功以后，添加额外的处理业务，例如缓存结果数据或清除缓存等。
 * @param wc WorkerContext 工作上下文
 * @param r *jsonx.JsonResponse JSON响应结果
 * @return stop 是否停止执行后续插件，true表示停止，将直接返回响应，达到过滤效果
 */
type Filter func(wc WorkerContext, r *jsonx.JsonResponse) (stop bool)

// RuleFunc 自定义规则函数
type RuleFunc func(ctx types.RuleContext, msg types.RuleMsg, ws WorkerServer)

// RuleEngine 规则引擎
type RuleEngineManager interface {
	// AddRuleEngine 添加规则引擎
	AddRuleEngine(w *Worker) error
	// 获取规则引擎, entityLabel: 带版本的实体标签, ruleId: 规则ID
	Engine(entityLabel string, ruleId string) *types.RuleEngine
	// RegisterRuleFunc 注册自定义规则函数
	RegisterRuleFunc(funcName string, ruleFunc RuleFunc)
	// SetGlobalConfig 设置全局配置
	SetGlobalConfig(config *types.Config)
	// UpdateRules 更新规则
	HandleRuleUpdate(ctx WorkerContext, paramStr string) error
}

const PLUGIN_HEADER = "X-Plugin-Worker"

// PluginWorker 定义了插件工作的核心接口。
// 该接口用于定义插件的基本行为，包括启动、停止、接收事件以及处理事件。
type PluginWorker interface {
	//  Setup 安装插件。
	// 返回一个错误，如果启动过程中发生错误，则返回非 nil 的错误。
	Setup() error

	// ReceiveCodes 返回插件处理的内部事件类型列表。
	// 该列表用于确定插件能够处理哪些类型的事件。
	ReceiveCodes() []INTRANET_EVENT_TYPE

	// Handle 处理接收到的内部事件。
	// ctx 是工作上下文，提供了与当前事件相关的上下文信息。
	// typz 是事件的类型，用于确定如何处理该事件。
	// 返回一个错误，如果处理过程中发生错误，则返回非 nil 的错误。
	Handle(ctx WorkerContext, typz INTRANET_EVENT_TYPE) error
}
