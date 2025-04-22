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

// PROJECT_STATE 定义项目状态的枚举类型
type PROJECT_STATE int

const (
	// PROJECT_ITERATING 项目处于迭代中状态
	PROJECT_ITERATING = 0
	// PROJECT_SEALED 项目已封版状态
	PROJECT_SEALED = 1
	// PROJECT_OFFLINE 项目已下线状态
	PROJECT_OFFLINE = 2
)

const (
	// INTERNAL_PROJECT 默认网关系统项目名
	INTERNAL_PROJECT = "event"
)

// Project 表示系统中的项目，用于组织和管理相关的资源和配置
type Project struct {
	// Code 项目编码，作为主键
	Code string `json:"code" gorm:"primaryKey"`
	// Name 项目名称
	Name string `json:"name"`
	// State 项目状态
	State PROJECT_STATE `json:"state"`
	// Description 项目描述
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

// NewProjectFromJson 从JSON字符串创建Project实例
// v 参数是一个符合Project结构的JSON字符串
// 返回创建的Project实例
func NewProjectFromJson(v string) *Project {
	var data Project
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &Project{}
	}
	return &data
}

// NewProjectFromMap 从map类型数据创建Project实例
// v 参数应该是一个包含Project字段值的map[string]interface{}
// 返回创建的Project实例
func NewProjectFromMap(v interface{}) *Project {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &Project{}
	}

	return &Project{
		Code:        cast.ToString(data["code"]),
		Name:        cast.ToString(data["name"]),
		State:       PROJECT_STATE(cast.ToInt(data["state"])),
		Description: cast.ToString(data["description"]),
		CreatedAt:   cast.ToInt64(data["createdAt"]),
		UpdatedAt:   cast.ToInt64(data["updatedAt"]),
		DeletedAt:   cast.ToInt64(data["deletedAt"]),
		DeletedBy:   cast.ToString(data["deletedBy"]),
		Creator:     cast.ToString(data["creator"]),
	}
}

// Clone 创建当前Project实例的深拷贝
// 如果接收者为nil，则返回空的Project对象
// 返回一个与当前实例数据相同但独立的新实例
func (p *Project) Clone() *Project {
	if p == nil {
		return &Project{}
	}
	return &Project{
		Code:        p.Code,
		Name:        p.Name,
		State:       p.State,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		DeletedAt:   p.DeletedAt,
		DeletedBy:   p.DeletedBy,
		Creator:     p.Creator,
	}
}
