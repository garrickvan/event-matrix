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
	"sync"
	"time"

	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/loadtool"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/cache"
	"github.com/garrickvan/event-matrix/worker/intranet/dispatcher"
	"github.com/garrickvan/event-matrix/worker/intranet/gnetimpl"
	"github.com/garrickvan/event-matrix/worker/public/hertzimpl"
	"github.com/garrickvan/event-matrix/worker/repo"
	"github.com/garrickvan/event-matrix/worker/ruleengine"
	"github.com/garrickvan/event-matrix/worker/types"
)

// TwoWayWorkerServer 是一个双向工作服务器实现
// 它同时支持公网和内域通信，管理工作节点、插件、路由和任务执行
type TwoWayWorkerServer struct { // WILLDO: 检查字段的并发安全性
	public   serverx.NetworkServer // 公网服务器实例
	intranet serverx.NetworkServer // 内域服务器实例

	cfg    *types.WorkerServerConfig // 服务器配置
	cfgKey string                    // 配置键名

	sharedConfigures        sync.Map                          // 共享配置存储，线程安全
	onSharedConfigureChange types.OnSharedConfigureChangeFunc // 配置变更回调函数

	workerIds          map[string]bool          // 工作节点ID集合
	entityMapToWorkers map[string]*types.Worker // 实体到工作节点的映射
	failedWorkers      map[string]*types.Worker // 失败的工作节点

	plugins      map[types.INTRANET_EVENT_TYPE]types.PluginWorker // 插件映射
	interceptors []types.Intercept                                // 拦截器列表
	filters      []types.Filter                                   // 过滤器列表

	routers map[string]types.WorkerExecutor     // 路由执行器映射
	tasks   map[string]types.WorkerTaskExecutor // 任务执行器映射

	repo          types.Repository        // 数据仓库接口
	ruleEngineMgr types.RuleEngineManager // 规则引擎管理器

	cache       types.DefaultCache // 默认缓存
	domainCache types.DomainCache  // 领域模型缓存
}

// TwoWayWorkerServerSettings 包含创建TwoWayWorkerServer所需的基本配置
type TwoWayWorkerServerSettings struct {
	CfgKey                  string                // 配置键，用于从配置中心获取服务器配置
	PublicServer            serverx.NetworkServer // 公网服务器实例，不设置则默认使用Hertz作为HTTP协议的公网服务器
	IntranetSecret          string                // 内域通信加密密钥
	IntranetSecretAlgor     string                // 内域通信加密算法
	GatewayIntranetEndpoint string                // 内域网关服务地址
}

// NewTwoWayWorkerServer 创建并初始化一个新的TwoWayWorkerServer实例
// 它会根据提供的设置初始化日志、内域客户端，并尝试从配置中心获取完整配置
func NewTwoWayWorkerServer(s TwoWayWorkerServerSettings) *TwoWayWorkerServer {
	// 临时日志
	logx.InitRuntimeLogger("logs", "info", "", 20*time.Second)
	// 临时初始化内域服务客户端
	dispatcher.InitClient(
		1,
		time.Duration(10)*time.Second,
		time.Duration(30)*time.Second,
		s.GatewayIntranetEndpoint,
		"",
		s.IntranetSecret,
		s.IntranetSecretAlgor,
		false,
	)
	// 优先从远程配置中心获取配置
	if s.CfgKey != "" &&
		s.IntranetSecret != "" &&
		s.IntranetSecretAlgor != "" &&
		s.GatewayIntranetEndpoint != "" {
		return initByCfgKey(&s)
	}
	panic("配置项不完整，必须指定[CfgKey, IntranetSecret, IntranetSecretAlgor, GatewayIntranetEndpoint]")
}

// initByCfgKey 通过配置键从配置中心获取完整配置并初始化服务器
// 如果获取配置失败或配置不完整，将会触发panic
func initByCfgKey(s *TwoWayWorkerServerSettings) *TwoWayWorkerServer {
	perloads := dispatcher.LoadSharedCfgFromGateway([]string{s.CfgKey})
	cfg := types.WorkerServerConfig{}
	var cfgJson string
	// 检查 s.CfgKey 是否存在于 perloads 中
	if preload, exists := perloads[s.CfgKey]; exists {
		if preload != nil {
			cfgJson = preload.Value
		}
	} else {
		panic("配置中心返回的WorkerServer配置文件不存在")
	}
	if cfgJson == "" {
		panic("配置中心返回的WorkerServer配置文件为空")
	}
	err := jsonx.UnmarshalFromStr(cfgJson, &cfg)
	if err != nil {
		panic("WorkerServer配置文件解析失败: " + err.Error())
	}
	if cfg.PublicHost == "" || cfg.PublicPort == 0 {
		panic("WorkerServer配置文件中[public_host、public_port]不能为空")
	}
	types.PatchWorkerServerConfig(&cfg)
	cfg.IntranetSecret = s.IntranetSecret
	cfg.IntranetSecretAlgor = s.IntranetSecretAlgor
	cfg.GatewayIntranetEndpoint = s.GatewayIntranetEndpoint
	return newWorkerServerFromConfig(&cfg, s, perloads)
}

// newWorkerServerFromConfig 使用完整配置初始化WorkerServer的各个组件
// 包括日志系统、内域客户端、缓存、数据仓库、规则引擎和网络服务等
func newWorkerServerFromConfig(
	cfg *types.WorkerServerConfig, s *TwoWayWorkerServerSettings, perloads map[string]*core.SharedConfigure,
) *TwoWayWorkerServer {
	// 重新初始化内域服务客户端
	dispatcher.InitClient(
		cfg.IntranetClientMaxIdleConnsPerHost,
		time.Duration(cfg.IntranetClientConnectionExpired)*time.Second,
		time.Duration(cfg.IntranetClientWriteTimeout)*time.Second,
		cfg.GatewayIntranetEndpoint,
		cfg.IntranetHost,
		cfg.IntranetSecret,
		cfg.IntranetSecretAlgor,
		cfg.IntranetCompress,
	)
	// 初始化日志
	logSlicePeriod := time.Duration(cfg.LogSlicePeriod) * time.Second
	logx.InitEventLogger(cfg.LogLocation, cfg.ServerId, logSlicePeriod)
	logx.InitRuntimeLogger(cfg.LogLocation, cfg.LogLevel, cfg.ServerId, logSlicePeriod)
	loadtool.Init(6) // 6*5s = 30s 采样周期
	// 初始化两路Worker服务
	ws := TwoWayWorkerServer{
		cfgKey: s.CfgKey,
		cfg:    cfg,

		workerIds:          map[string]bool{},
		entityMapToWorkers: make(map[string]*types.Worker),
		failedWorkers:      make(map[string]*types.Worker),

		plugins: map[types.INTRANET_EVENT_TYPE]types.PluginWorker{},

		interceptors: []types.Intercept{},
		filters:      []types.Filter{},

		routers: make(map[string]types.WorkerExecutor),
		tasks:   make(map[string]types.WorkerTaskExecutor),
	}
	for _, cfg := range perloads {
		ws.sharedConfigures.Store(cfg.Key, cfg)
	}
	// 初始化 WorkerServer 内部组件
	ws.ruleEngineMgr = ruleengine.NewRuleEngineManager(&ws)
	// 初始化 WorkerServer 数据库组件
	ws.repo = repo.NewRepository(&ws)
	// 初始化默认缓存
	defaultCache, err := cache.NewDefaultCacheImpl(cfg.DefaultCacheMaxMen, cfg.DefaultCacheTTL, &ws)
	if err != nil {
		panic("初始化默认缓存失败: " + err.Error())
	}
	ws.cache = defaultCache
	// 初始化领域缓存
	domainCache, err := cache.NewDomainCacheImpl(cfg.DomainCacheMaxMen, cfg.DomainCacheTTL, &ws)
	if err != nil {
		panic("初始化领域缓存失败: " + err.Error())
	}
	ws.domainCache = domainCache
	if s.PublicServer == nil {
		// 初始化公网服务
		pSvr := hertzimpl.NewWorkerPublicServer(cfg, &ws)
		ws.public = pSvr
	} else {
		ws.public = s.PublicServer
	}
	// 初始化内域服务
	iSvr := gnetimpl.NewWorkerIntranetServer(cfg, &ws)
	ws.intranet = iSvr
	return &ws
}

// Cfg 获取服务器配置
// 如果配置为空，会创建一个带有默认值的配置对象
func (s *TwoWayWorkerServer) Cfg() *types.WorkerServerConfig {
	if s.cfg == nil {
		cfg := types.WorkerServerConfig{}
		types.PatchWorkerServerConfig(&cfg)
		s.cfg = &cfg
		return s.cfg
	}
	return s.cfg
}

// Start 启动工作服务器
// 它会先向网关注册自身端点信息，然后启动内域和公网服务
// 内域服务在单独的goroutine中启动，公网服务在主调用线程中启动
func (s *TwoWayWorkerServer) Start() error {
	endpoint := core.Endpoint{
		ServerId:     s.Cfg().ServerId,
		PublicHost:   s.Cfg().PublicHost,
		PublicPort:   s.Cfg().PublicPort,
		IntranetHost: s.Cfg().IntranetHost,
		IntranetPort: s.Cfg().IntranetPort,
		Type:         core.WORKER_ENDPOINT,
	}
	err := dispatcher.ReportEndpoint(&endpoint)
	if err != nil {
		logx.Error("上报WorkerServer信息失败: " + err.Error())
	}
	// 启动内域网络服务
	go func() {
		err := s.intranet.Start()
		if err != nil {
			logx.Error("启动内域网络服务失败: " + err.Error())
		}
	}()
	// 启动网络服务
	err = s.public.Start()
	if err != nil {
		return err
	}
	return nil
}

// Stop 停止工作服务器
// 依次停止公网和内域服务，任何一个停止失败都会返回错误
func (s *TwoWayWorkerServer) Stop() error {
	err := s.public.Stop()
	if err != nil {
		return err
	}
	err = s.intranet.Stop()
	if err != nil {
		return err
	}
	return nil
}
