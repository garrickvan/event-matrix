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
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/fastconv"
)

type INTRANET_EVENT_TYPE uint16

// 内部协议，仅用于内部通信，不要修改也不要用于数据存储
// W_T_G: worker to gateway
// G_T_W: gateway to worker
// G_T_G: gateway to gateway
// W_T_W: worker to worker
// GW_T_W: gateway worker to worker
// GW_T_G: gateway worker to gateway
const (
	UNKNOWN_EVENT    INTRANET_EVENT_TYPE = 0 // 未知事件          INTRANET_EVENT_TYPE = 0 // 未知事件
	W_T_W_EVENT_CALL INTRANET_EVENT_TYPE = 1 // 工作端之间互调事件

	W_T_G_REGISTER                     INTRANET_EVENT_TYPE = 10001 // 工作端注册
	W_T_G_GET_ENTITY                   INTRANET_EVENT_TYPE = 10002 // 获取实体信息
	W_T_G_GET_ENTITY_ATTRS             INTRANET_EVENT_TYPE = 10003 // 获取实体属性
	W_T_G_GET_ENTITY_EVENTS            INTRANET_EVENT_TYPE = 10004 // 获取实体事件
	W_T_G_GET_ENDPOINT_BY_EVENT        INTRANET_EVENT_TYPE = 10005 // 根据事件获取端点
	W_T_G_VERIFY_EVENT                 INTRANET_EVENT_TYPE = 10006 // 验证事件
	W_T_G_VERIFY_EVENT_WITHOUT_EXPIRED INTRANET_EVENT_TYPE = 10007 // 验证事件（忽略过期）
	W_T_G_GET_USER_ID_BY_UCODE         INTRANET_EVENT_TYPE = 10008 // 根据用户码获取用户信息
	W_T_G_SEARCH_USER_INFO             INTRANET_EVENT_TYPE = 10009 // 搜索用户信息
	W_T_G_REPORT_CONF_USED_BY          INTRANET_EVENT_TYPE = 10010 // 报告配置使用情况
	W_T_G_GET_SHARED_CONFIGURE         INTRANET_EVENT_TYPE = 10011 // 获取共享配置
	W_T_G_GET_CONSTANTS                INTRANET_EVENT_TYPE = 10012 // 获取常量
	GW_T_G_REPORT_ENDPOINT             INTRANET_EVENT_TYPE = 10013 // 网关上报端点信息
	W_T_G_GET_USER_DETAIL              INTRANET_EVENT_TYPE = 10014 // 获取用户详情
	W_T_G_SAVE_USER_SENSITIVE_INFO     INTRANET_EVENT_TYPE = 10015 // 保存用户敏感信息
	W_T_G_GET_USER_SENSITIVE_INFO      INTRANET_EVENT_TYPE = 10016 //  获取用户敏感信息

	G_T_W_CHECK_WORKER               INTRANET_EVENT_TYPE = 20000 // 来自网关的检查工作端是否存在
	G_T_W_RULE_UPDATE                INTRANET_EVENT_TYPE = 20001 // 来自网关的规则更新
	G_T_W_SHARED_CONFIGURE_CHANGE    INTRANET_EVENT_TYPE = 20002 // 来自网关的共享配置变更
	G_T_W_ENTITY_LIST_FOR_DATA_MGR   INTRANET_EVENT_TYPE = 20003 // 来自网关的数据管理实体列表
	G_T_W_RESET_DOMAIN_CACHE         INTRANET_EVENT_TYPE = 20004 // 来自网关的域缓存重置
	G_T_W_UPDATE_RECORD_FOR_DATA_MGR INTRANET_EVENT_TYPE = 20005 // 来自网关的数据管理记录更新
	G_T_W_GET_LOADE_RATE             INTRANET_EVENT_TYPE = 20006 // 来自网关的获取负载率

	WORKER_INTERNAL_PLUGIN INTRANET_EVENT_TYPE = 30000 // 工作端内部插件，预留段号
)

// 内部事件类型是否有效, 1-40000为内部事件类型，预留部分类型
func IsIntranetEventType(typz int32) bool {
	return typz >= 1 && typz <= 40000
}

// 内部事件
type IntranetEvent struct {
	Type   INTRANET_EVENT_TYPE `json:"type"`   // 事件类型
	Params string              `json:"params"` // 事件参数
}

// 检查结果
type WorkerCheckResult struct {
	WorkerId string  `json:"wid"`
	Exist    bool    `json:"exist"`
	LoadRate float64 `json:"load_rate"`
}

// 工作端公网地址信息
type WorkerPublicEndpointInfo struct {
	Timeout        int    `json:"timeout"`        // 超时时间
	PublicEndpoint string `json:"publicEndpoint"` // 公网地址
}

// 工作端内域地址信息
type WorkerIntranetEndpointInfo struct {
	IntranetEndpoint string `json:"intranetEndpoint"` // 内域地址
}

// 常用的查询参数结构体
type SearchByFieldParam struct {
	Field   string `json:"field"`   // 查询字段
	Keyword string `json:"keyword"` // 查询关键字
	Page    int    `json:"page"`    // 页码
	Size    int    `json:"size"`    // 每页大小
}

// 内部请求参数结构体, 对参数简化, 仅包含必要的字段，降低内部通信的计算量
type PathToEntity struct {
	Project string `json:"project"` // 项目名称
	Version string `json:"version"` // 版本号
	Context string `json:"context"` // 上下文
	Entity  string `json:"entity"`  // 实体名称
}

// 从核心事件对象生成 PathToEntity 结构体
func PathToEntityFromEvent(e *core.Event) PathToEntity {
	if e == nil {
		return PathToEntity{}
	}
	p := PathToEntity{}
	p.Project = e.Project
	p.Version = e.Version
	p.Context = e.Context
	p.Entity = e.Entity
	return p
}

// 检查 PathToEntity 结构体是否不完整
func (p *PathToEntity) IsIncomplete() bool {
	return p.Project == "" || p.Version == "" || p.Context == "" || p.Entity == ""
}

// 从工作端对象生成 PathToEntity 结构体
func PathToEntityFromWorker(w *Worker) PathToEntity {
	if w == nil {
		return PathToEntity{}
	}
	p := PathToEntity{}
	p.Project = w.Project
	p.Version = w.VersionLabel
	p.Context = w.Context
	p.Entity = w.Entity
	return p
}

// 从 PathToEvent 结构体生成 PathToEntity 结构体
func PathToEntityFromPathToEvent(e PathToEvent) PathToEntity {
	p := PathToEntity{}
	p.Project = e.Project
	p.Version = e.Version
	p.Context = e.Context
	p.Entity = e.Entity
	return p
}

// 将 PathToEntity 结构体转换为字符串参数
func (p *PathToEntity) ToStrArg() string {
	return strings.Join([]string{p.Project, p.Version, p.Context, p.Entity}, constant.SPLIT_CHAR)
}

// 从字符串参数生成 PathToEntity 结构体
func PathToEntityFromStrArg(s string) PathToEntity {
	if s == "" {
		return PathToEntity{}
	}
	arr := fastconv.SafeSplit(s, constant.SPLIT_CHAR)
	if len(arr) < 4 {
		return PathToEntity{}
	}
	p := PathToEntity{}
	p.Project = arr[0]
	p.Version = arr[1]
	p.Context = arr[2]
	p.Entity = arr[3]
	return p
}

// 将字节参数 转换为 PathToEntity 结构体
func PathToEntityFromBytesArg(b []byte) PathToEntity {
	return PathToEntityFromStrArg(fastconv.BytesToString(b))
}

// 事件路径结构体
type PathToEvent struct {
	Project string `json:"project"` // 项目名称
	Version string `json:"version"` // 版本号
	Context string `json:"context"` // 上下文
	Entity  string `json:"entity"`  // 实体名称
	Event   string `json:"event"`   // 事件名称
}

// 检查 PathToEvent 结构体是否不完整
func (p *PathToEvent) IsIncomplete() bool {
	return p.Project == "" || p.Version == "" || p.Context == "" || p.Entity == "" || p.Event == ""
}

// 从核心事件对象生成 PathToEvent 结构体
func PathToEventFromEvent(e *core.Event) PathToEvent {
	if e == nil {
		return PathToEvent{}
	}
	p := PathToEvent{}
	p.Project = e.Project
	p.Version = e.Version
	p.Context = e.Context
	p.Entity = e.Entity
	p.Event = e.Event
	return p
}

// 将 PathToEvent 结构体转换为字符串参数
func (p *PathToEvent) ToStrArg() string {
	return strings.Join([]string{p.Project, p.Version, p.Context, p.Entity, p.Event}, constant.SPLIT_CHAR)
}

// 从字符串参数生成 PathToEvent 结构体
func PathToEventFromStrArg(s string) PathToEvent {
	if s == "" {
		return PathToEvent{}
	}
	arr := strings.Split(s, constant.SPLIT_CHAR)
	if len(arr) < 5 {
		return PathToEvent{}
	}
	p := PathToEvent{}
	p.Project = arr[0]
	p.Version = arr[1]
	p.Context = arr[2]
	p.Entity = arr[3]
	p.Event = arr[4]
	return p
}

// 将字节参数 转换为 PathToEvent 结构体
func PathToEventFromBytesArg(b []byte) PathToEvent {
	return PathToEventFromStrArg(fastconv.BytesToString(b))
}
