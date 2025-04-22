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

// Entity 表示事件矩阵系统中的实体对象，用于定义业务模型
type Entity struct {
	// ID 实体的唯一标识符
	ID string `json:"id" gorm:"primaryKey"`
	// ContextID 关联的上下文ID
	ContextID string `json:"contextId" gorm:"index"`
	// Name 实体名称
	Name string `json:"name"`
	// Code 实体编码，用于系统内部引用
	Code string `json:"code" gorm:"index"`
	// Description 实体描述信息
	Description string `json:"description"`
	// BusinessRules 业务规则定义，JSON格式字符串
	BusinessRules string `json:"businessRules"`
	// DeletedAt 删除时间戳
	DeletedAt int64 `json:"deletedAt" gorm:"index"`
	// DeletedBy 删除操作执行者
	DeletedBy string `json:"deletedBy"`
	// CreatedAt 创建时间戳
	CreatedAt int64 `json:"createdAt"`
	// UpdatedAt 更新时间戳
	UpdatedAt int64 `json:"updatedAt"`
	// Creator 创建者
	Creator string `json:"creator"`
}

// BusinessRules 定义实体的业务规则结构
type BusinessRules struct {
	// ID 规则唯一标识符
	ID string `json:"id"`
	// Name 规则名称
	Name string `json:"name"`
	// Context 规则上下文，包含规则的具体定义
	Context string `json:"context"`
}

// NewEntityFromMap 从map类型数据创建Entity实例
// v 参数应该是一个包含Entity字段值的map[string]interface{}
func NewEntityFromMap(v interface{}) *Entity {
	if v == nil {
		return &Entity{}
	}

	data, ok := v.(map[string]interface{})
	if !ok {
		return &Entity{}
	}

	return &Entity{
		ID:            cast.ToString(data["id"]),
		ContextID:     cast.ToString(data["contextId"]),
		Name:          cast.ToString(data["name"]),
		Code:          cast.ToString(data["code"]),
		Description:   cast.ToString(data["description"]),
		BusinessRules: cast.ToString(data["businessRules"]),
		DeletedAt:     cast.ToInt64(data["deletedAt"]),
		DeletedBy:     cast.ToString(data["deletedBy"]),
		CreatedAt:     cast.ToInt64(data["createdAt"]),
		UpdatedAt:     cast.ToInt64(data["updatedAt"]),
		Creator:       cast.ToString(data["creator"]),
	}
}

// NewEntityFromJson 从JSON字符串创建Entity实例
// v 参数是一个符合Entity结构的JSON字符串
func NewEntityFromJson(v string) *Entity {
	var data Entity
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &Entity{}
	}
	return &data
}

// Clone 创建当前Entity实例的深拷贝
// 如果接收者为nil，则返回空的Entity对象
func (e *Entity) Clone() *Entity {
	if e == nil {
		return &Entity{}
	}
	return &Entity{
		ID:            e.ID,
		ContextID:     e.ContextID,
		Name:          e.Name,
		Code:          e.Code,
		Description:   e.Description,
		BusinessRules: e.BusinessRules,
		DeletedAt:     e.DeletedAt,
		DeletedBy:     e.DeletedBy,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
		Creator:       e.Creator,
	}
}
