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
	"strings"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/spf13/cast"
)

// EntityEvent 表示系统中的一个事件实体
// 它包含了事件的基本信息、执行配置、权限控制等属性
type EntityEvent struct {
	ID           string                 `json:"id" gorm:"primaryKey"`   // 事件唯一标识
	EntityID     string                 `json:"entityId" gorm:"index"`  // 关联的实体ID
	Name         string                 `json:"name"`                   // 事件名称
	Code         string                 `json:"code" gorm:"index"`      // 事件代码，用于标识事件类型
	ExecutorType constant.EXECUTOR_TYPE `json:"executorType"`           // 执行器类型
	Executor     string                 `json:"executor"`               // 执行器配置
	Delay        int                    `json:"delay"`                  // 延迟执行时间，单位秒
	Timeout      int                    `json:"timeout"`                // 执行超时时间，单位秒
	Params       string                 `json:"params"`                 // 事件参数，JSON格式
	Mode         constant.EVENT_MODE    `json:"mode"`                   // 事件模式
	Idempotent   bool                   `json:"idempotent"`             // WILLDO:幂等性
	Logable      bool                   `json:"logable"`                // 是否启用日志
	AuthType     constant.AUTH_TYPE     `json:"authType"`               // 认证类型
	Description  string                 `json:"description"`            // 事件描述
	CreatedAt    int64                  `json:"createdAt"`              // 创建时间戳
	UpdatedAt    int64                  `json:"updatedAt"`              // 更新时间戳
	DeletedAt    int64                  `json:"deletedAt" gorm:"index"` // 删除时间戳
	DeletedBy    string                 `json:"deletedBy"`              // 删除操作执行者
	Creator      string                 `json:"creator"`                // 创建者
}

// NewEntityEventFromJson 从JSON字符串创建EntityEvent实例
// 如果解析失败则返回空的EntityEvent对象
func NewEntityEventFromJson(v string) *EntityEvent {
	var data EntityEvent
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &EntityEvent{}
	}
	return &data
}

// NewEntityEventFromMap 从map[string]interface{}创建EntityEvent实例
// 使用类型转换确保数据类型正确性，如果转换失败则返回空的EntityEvent对象
func NewEntityEventFromMap(v interface{}) *EntityEvent {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &EntityEvent{}
	}

	return &EntityEvent{
		ID:           cast.ToString(data["id"]),
		EntityID:     cast.ToString(data["entityId"]),
		Name:         cast.ToString(data["name"]),
		Code:         cast.ToString(data["code"]),
		ExecutorType: constant.EXECUTOR_TYPE(cast.ToUint8(data["executorType"])),
		Executor:     cast.ToString(data["executor"]),
		Delay:        cast.ToInt(data["delay"]),
		Timeout:      cast.ToInt(data["timeout"]),
		Params:       cast.ToString(data["params"]),
		Mode:         constant.EVENT_MODE(cast.ToString(data["mode"])),
		Logable:      cast.ToBool(data["logable"]),
		AuthType:     constant.AUTH_TYPE(cast.ToUint8(data["authType"])),
		Description:  cast.ToString(data["description"]),
		CreatedAt:    cast.ToInt64(data["createdAt"]),
		UpdatedAt:    cast.ToInt64(data["updatedAt"]),
		DeletedAt:    cast.ToInt64(data["deletedAt"]),
		DeletedBy:    cast.ToString(data["deletedBy"]),
		Creator:      cast.ToString(data["creator"]),
	}
}

// Clone 创建当前EntityEvent的深拷贝
// 如果接收者为nil，则返回空的EntityEvent对象
func (e *EntityEvent) Clone() *EntityEvent {
	if e == nil {
		return &EntityEvent{}
	}
	return &EntityEvent{
		ID:           e.ID,
		EntityID:     e.EntityID,
		Name:         e.Name,
		Code:         e.Code,
		ExecutorType: e.ExecutorType,
		Executor:     e.Executor,
		Delay:        e.Delay,
		Timeout:      e.Timeout,
		Params:       e.Params,
		Mode:         e.Mode,
		Logable:      e.Logable,
		AuthType:     e.AuthType,
		Description:  e.Description,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
		DeletedAt:    e.DeletedAt,
		DeletedBy:    e.DeletedBy,
		Creator:      e.Creator,
	}
}

// 事件参数
// EventParam 定义事件参数的结构和验证规则
type EventParam struct {
	Name       string `json:"name"`       // 参数名称（仅支持英文、数字和下划线的组合）
	Type       string `json:"type"`       // 参数数据类型（如string、int、bool等）
	Range      string `json:"range"`      // 参数值的有效范围类型（如enum、range等）
	RangeValue string `json:"rangeValue"` // 参数的具体取值范围定义
	Required   bool   `json:"required"`   // 标识参数是否为必填项
}

// NewEventParamFromJson 从JSON字符串创建EventParam实例
// 如果解析失败则返回空的EventParam对象
func NewEventParamFromJson(v string) *EventParam {
	var data EventParam
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &EventParam{}
	}
	return &data
}

// NewEventParamFromMap 从map[string]interface{}创建EventParam实例
// 使用类型断言确保数据有效性，如果断言失败则返回空的EventParam对象
func NewEventParamFromMap(v interface{}) *EventParam {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &EventParam{}
	}
	return &EventParam{
		Name:       cast.ToString(data["name"]),
		Type:       cast.ToString(data["type"]),
		Range:      cast.ToString(data["range"]),
		RangeValue: cast.ToString(data["rangeValue"]),
		Required:   cast.ToBool(data["required"]),
	}
}

// Clone 创建当前EventParam的深拷贝
// 如果接收者为nil，则返回空的EventParam对象
func (e *EventParam) Clone() *EventParam {
	if e == nil {
		return &EventParam{}
	}
	return &EventParam{
		Name:       e.Name,
		Type:       e.Type,
		Range:      e.Range,
		RangeValue: e.RangeValue,
		Required:   e.Required,
	}
}

// FindParamFromArray 在参数数组中查找指定名称的参数
// 参数：
//   - name: 要查找的参数名称
//   - params: 参数数组
//
// 返回值：
//   - *EventParam: 找到的参数对象指针
//   - bool: 是否找到参数
func FindParamFromArray(name string, params []EventParam) (*EventParam, bool) {
	if params == nil {
		return nil, false
	}

	for _, param := range params {
		if strings.EqualFold(param.Name, name) {
			return &param, true
		}
	}
	return nil, false
}
