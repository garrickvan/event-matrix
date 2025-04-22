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
	"strings"

	"github.com/garrickvan/event-matrix/constant"
)

// WorkerServerConfig 定义工作服务器的完整配置结构
// 包含服务器基础信息、网络设置、缓存配置、日志配置等
type WorkerServerConfig struct {
	// 基础配置
	ServerId string               `yaml:"server_id" json:"server_id"` // 工作节点唯一标识符
	WorkMode string               `yaml:"work_mode" json:"work_mode"` // 工作模式：QUERY（查询模式）或COMMAND（命令模式）
	Version  string               `yaml:"version" json:"version"`     // 服务器版本号
	Mode     constant.SERVER_MODE `yaml:"mode" json:"mode"`           // 服务器运行环境模式（开发/生产）

	// 外部服务相关配置（公网API服务）
	PublicHost       string `yaml:"public_host" json:"public_host"`               // 公网服务主机地址
	PublicPort       int    `yaml:"public_port" json:"public_port"`               // 公网服务端口
	HttpReadTimeout  int    `yaml:"http_read_timeout" json:"http_read_timeout"`   // HTTP请求读取超时时间（秒）
	HttpWriteTimeout int    `yaml:"http_write_timeout" json:"http_write_timeout"` // HTTP响应写入超时时间（秒）

	// 内部服务相关配置（内域通信服务）
	IntranetHost                      string `yaml:"intranet_host" json:"intranet_host"`                                                     // 内域服务主机地址
	IntranetPort                      int    `yaml:"intranet_port" json:"intranet_port"`                                                     // 内域服务端口
	IntranetSecret                    string `yaml:"intranet_secret" json:"intranet_secret"`                                                 // 内域通信加密密钥
	IntranetSecretAlgor               string `yaml:"intranet_secret_algor" json:"intranet_secret_algor"`                                     // 内域通信加密算法
	IntranetClientMaxIdleConnsPerHost int    `yaml:"intranet_client_max_idle_conns_per_host" json:"intranet_client_max_idle_conns_per_host"` // 内域客户端每个主机最大空闲连接数
	IntranetClientConnectionExpired   int    `yaml:"intranet_client_connection_expired" json:"intranet_client_connection_expired"`           // 内域客户端连接过期时间（秒）
	IntranetClientWriteTimeout        int    `yaml:"intranet_client_write_timeout" json:"intranet_client_write_timeout"`                     // 内域客户端写入超时时间（秒）
	IntranetCompress                  bool   `yaml:"intranet_compress" json:"intranet_compress"`                                             // 内域通信是否启用压缩

	// 日志相关配置
	LogLevel       string `yaml:"log_level" json:"log_level"`               // 日志级别（debug/info/warn/error）
	LogLocation    string `yaml:"log_location" json:"log_location"`         // 日志文件存储位置
	LogSlicePeriod int    `yaml:"log_slice_period" json:"log_slice_period"` // 日志文件切片周期（秒）

	// 默认缓存配置
	DefaultCacheMaxMen int64 `yaml:"default_cache_max_men" json:"default_cache_max_men"` // 默认缓存最大内存占用（字节）
	DefaultCacheTTL    int   `yaml:"default_cache_ttl" json:"default_cache_ttl"`         // 默认缓存项过期时间（秒）

	// 领域模型缓存配置
	DomainCacheMaxMen int64 `yaml:"domain_cache_max_men" json:"domain_cache_max_men"` // 领域缓存最大内存占用（字节）
	DomainCacheTTL    int   `yaml:"domain_cache_ttl" json:"domain_cache_ttl"`         // 领域缓存项过期时间（秒）

	// 其他配置
	GatewayIntranetEndpoint               string `yaml:"gateway_intranet_endpoint" json:"gateway_intranet_endpoint"`                                     // 网关内域服务地址
	HeartbeatReportGap                    int    `yaml:"heartbeat_report_gap" json:"heartbeat_report_gap"`                                               // 心跳上报间隔（秒）
	NotAcceptUpdateRecordEventFromGateway bool   `yaml:"not_accept_update_record_event_from_gateway" json:"not_accept_update_record_event_from_gateway"` // 是否拒绝来自网关的更新记录事件
}

// PatchWorkerServerConfig 为WorkerServerConfig补充默认配置值
// 当配置项为空或零值时，会设置合理的默认值，确保服务器可以正常启动
func PatchWorkerServerConfig(cfg *WorkerServerConfig) {
	// 补充默认配置
	if cfg.ServerId == "" {
		cfg.ServerId = "worker-1"
	}
	// 修正工作模式配置项
	if cfg.WorkMode == "" {
		cfg.WorkMode = string(constant.COMMAND_MODE)
	}
	cfg.WorkMode = strings.ToUpper(cfg.WorkMode)
	if cfg.WorkMode == "QUERY" {
		cfg.WorkMode = string(constant.QUERY_MODE)
	}
	if cfg.WorkMode == "COMMAND" {
		cfg.WorkMode = string(constant.COMMAND_MODE)
	}
	if cfg.Version == "" {
		cfg.Version = "0.0.1"
	}
	if cfg.LogSlicePeriod < 5 {
		cfg.LogSlicePeriod = 20
	}
	if cfg.Mode == "" {
		cfg.Mode = constant.DEV
	}
	if cfg.PublicPort == 0 {
		cfg.PublicPort = 8080
	}
	if cfg.HttpReadTimeout == 0 {
		cfg.HttpReadTimeout = 10
	}
	if cfg.HttpWriteTimeout == 0 {
		cfg.HttpWriteTimeout = 10
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "debug"
	}
	if cfg.LogLocation == "" {
		cfg.LogLocation = "logs"
	}
	if cfg.DefaultCacheMaxMen == 0 {
		cfg.DefaultCacheMaxMen = 100 * 1024 * 1024 // 100MB
	}
	if cfg.DefaultCacheTTL == 0 {
		cfg.DefaultCacheTTL = 5 * 60 // 5分钟
	}
	if cfg.DomainCacheMaxMen == 0 {
		cfg.DomainCacheMaxMen = 100 * 1024 * 1024 // 100MB
	}
	if cfg.DomainCacheTTL == 0 {
		cfg.DomainCacheTTL = 5 * 60 // 5分钟
	}
	if cfg.IntranetClientMaxIdleConnsPerHost == 0 {
		cfg.IntranetClientMaxIdleConnsPerHost = 200
	}
	if cfg.IntranetClientConnectionExpired == 0 {
		cfg.IntranetClientConnectionExpired = 60 * 5 // 5分钟
	}
	if cfg.IntranetClientWriteTimeout == 0 {
		cfg.IntranetClientWriteTimeout = 30 // 30秒
	}
	if cfg.HeartbeatReportGap == 0 {
		cfg.HeartbeatReportGap = 60
	}
	if cfg.IntranetSecretAlgor == "" {
		cfg.IntranetSecretAlgor = "NONE"
	}
}
