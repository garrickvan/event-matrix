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

// Context 表示事件矩阵系统中的上下文环境，用于管理和组织相关的配置和资源
type Context struct {
	// ID 上下文的唯一标识符
	ID string `json:"id" gorm:"primaryKey"`
	// Name 上下文名称
	Name string `json:"name"`
	// Code 上下文编码，用于系统内部引用
	Code string `json:"code" gorm:"index"`
	// VersionID 版本ID，用于版本控制
	VersionID string `json:"versionId" gorm:"index"`
	// Description 上下文描述信息
	Description string `json:"description"`
	// CreatedAt 创建时间戳
	CreatedAt int64 `json:"createdAt"`
	// UpdatedAt 更新时间戳
	UpdatedAt int64 `json:"updatedAt"`
	// DeletedAt 删除时间戳
	DeletedAt int64 `json:"deletedAt" gorm:"index"`
	// DeletedBy 删除操作执行者
	DeletedBy string `json:"deletedBy"`
	// Creator 创建者
	Creator string `json:"creator"`
}

// NewContextFromMap 从map类型数据创建Context实例
// v 参数应该是一个包含Context字段值的map[string]interface{}
func NewContextFromMap(v interface{}) *Context {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &Context{}
	}

	return &Context{
		ID:          cast.ToString(data["id"]),
		Name:        cast.ToString(data["name"]),
		Code:        cast.ToString(data["code"]),
		VersionID:   cast.ToString(data["versionId"]),
		Description: cast.ToString(data["description"]),
		CreatedAt:   cast.ToInt64(data["createdAt"]),
		UpdatedAt:   cast.ToInt64(data["updatedAt"]),
		DeletedAt:   cast.ToInt64(data["deletedAt"]),
		DeletedBy:   cast.ToString(data["deletedBy"]),
		Creator:     cast.ToString(data["creator"]),
	}
}

// NewContextFromJson 从JSON字符串创建Context实例
// v 参数是一个符合Context结构的JSON字符串
func NewContextFromJson(v string) *Context {
	var data Context
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &Context{}
	}
	return &data
}

// Clone 创建当前Context实例的深拷贝
// 返回一个与当前实例数据相同但独立的新实例
func (c *Context) Clone() *Context {
	if c == nil {
		return &Context{}
	}
	return &Context{
		ID:          c.ID,
		Name:        c.Name,
		Code:        c.Code,
		VersionID:   c.VersionID,
		Description: c.Description,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		DeletedAt:   c.DeletedAt,
		DeletedBy:   c.DeletedBy,
		Creator:     c.Creator,
	}
}
