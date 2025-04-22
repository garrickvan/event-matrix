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

package core

import (
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/spf13/cast"
)

// ENDPOINT_TYPE 定义端点类型的枚举
type ENDPOINT_TYPE int

const (
	GATEWAY_ENDPOINT ENDPOINT_TYPE = 1 // 网关端点类型
	WORKER_ENDPOINT  ENDPOINT_TYPE = 2 // 工作节点端点类型
)

// Endpoint 定义系统中的服务端点信息
// 包含服务器的公网和内域访问地址、服务类型、状态等信息
type Endpoint struct {
	ServerId     string        `json:"serverId" gorm:"primaryKey"` // 服务实例唯一标识
	PublicHost   string        `json:"publicHost"`                 // 公网访问主机地址
	PublicPort   int           `json:"publicPort"`                 // 公网访问端口
	IntranetHost string        `json:"intranetHost"`               // 内域访问主机地址
	IntranetPort int           `json:"intranetPort"`               // 内域访问端口
	Type         ENDPOINT_TYPE `json:"type" gorm:"index"`          // 端点类型（网关/工作节点）
	Disabled     bool          `json:"disabled"`                   // 端点是否禁用
	RegisteredAt int64         `json:"registeredAt"`               // 注册时间戳
	UpdatedAt    int64         `json:"updatedAt"`                  // 更新时间戳
	DeletedAt    int64         `json:"deletedAt" gorm:"index"`     // 删除时间戳
}

// NewEndpointFromNewMap 从map[string]interface{}创建Endpoint实例
// 使用类型转换确保数据类型正确性
func NewEndpointFromNewMap(v interface{}) *Endpoint {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &Endpoint{}
	}

	return &Endpoint{
		ServerId:     cast.ToString(data["serverId"]),
		PublicHost:   cast.ToString(data["publicHost"]),
		PublicPort:   cast.ToInt(data["publicPort"]),
		IntranetHost: cast.ToString(data["intranetHost"]),
		IntranetPort: cast.ToInt(data["intranetPort"]),
		Type:         ENDPOINT_TYPE(cast.ToInt(data["type"])),
		Disabled:     cast.ToBool(data["disabled"]),
		RegisteredAt: cast.ToInt64(data["registeredAt"]),
		UpdatedAt:    cast.ToInt64(data["updatedAt"]),
		DeletedAt:    cast.ToInt64(data["deletedAt"]),
	}
}

// NewEndpointFromJson 从JSON字符串创建Endpoint实例
// 如果解析失败则返回空的Endpoint对象
func NewEndpointFromJson(v string) *Endpoint {
	var data Endpoint
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &Endpoint{}
	}
	return &data
}

// Clone 创建当前Endpoint的深拷贝
// 如果接收者为nil，则返回空的Endpoint对象
func (e *Endpoint) Clone() *Endpoint {
	if e == nil {
		return &Endpoint{}
	}
	return &Endpoint{
		ServerId:     e.ServerId,
		PublicHost:   e.PublicHost,
		PublicPort:   e.PublicPort,
		IntranetHost: e.IntranetHost,
		IntranetPort: e.IntranetPort,
		Type:         e.Type,
		Disabled:     e.Disabled,
		RegisteredAt: e.RegisteredAt,
		UpdatedAt:    e.UpdatedAt,
		DeletedAt:    e.DeletedAt,
	}
}
