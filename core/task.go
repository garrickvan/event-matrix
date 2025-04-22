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
	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/spf13/cast"
)

// TaskStatus 定义任务状态的枚举类型
type TaskStatus uint8

const (
	// TaskStatusPending 任务待处理状态
	TaskStatusPending TaskStatus = 0
	// TaskStatusInProgress 任务处理中状态
	TaskStatusInProgress TaskStatus = 1
	// TaskStatusSuccess 任务成功完成状态
	TaskStatusSuccess TaskStatus = 2
	// TaskStatusFailed 任务失败状态
	TaskStatusFailed TaskStatus = 3
	// TaskStatusTimeout 任务超时状态
	TaskStatusTimeout TaskStatus = 4
)

// Code 将TaskStatus转换为对应的响应码
// 返回与任务状态对应的系统响应码
func (ts TaskStatus) Code() constant.RESPONSE_CODE {
	switch ts {
	case TaskStatusPending:
		return constant.TASK_PENDING
	case TaskStatusInProgress:
		return constant.TASK_IN_PROGRESS
	case TaskStatusSuccess:
		return constant.SUCCESS
	case TaskStatusFailed:
		return constant.TASK_FAILED
	case TaskStatusTimeout:
		return constant.TASK_TIMEOUT
	default:
		return constant.TASK_UNKNOWN
	}
}

// Task 表示系统中的任务对象，用于跟踪和管理事件的执行
type Task struct {
	// ID 任务的唯一标识符
	ID string `gorm:"primaryKey" json:"id"`
	// EventID 关联的事件ID
	EventID string `gorm:"index" json:"eventId"`
	// EventLabel 事件标签，用于快速识别事件类型
	EventLabel string `gorm:"index" json:"eventLabel"`
	// Event 事件内容，JSON格式字符串
	Event string `json:"event"`
	// Status 任务状态
	Status TaskStatus `gorm:"index" json:"status"`
	// Retries 重试次数
	Retries int `json:"retries"`
	// ExecServer 执行任务的服务器ID
	ExecServer string `json:"execServer"`
	// CreatedAt 创建时间戳
	CreatedAt int64 `json:"createdAt"`
	// ExecuteAt 计划执行时间戳
	ExecuteAt int64 `json:"executeAt" gorm:"index"`
	// UpdatedAt 更新时间戳
	UpdatedAt int64 `json:"updatedAt"`
}

// NewTaskFromMap 从map类型数据创建Task实例
// v 参数应该是一个包含Task字段值的map[string]interface{}
// 返回创建的Task实例
func NewTaskFromMap(v interface{}) *Task {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &Task{}
	}

	return &Task{
		ID:         cast.ToString(data["id"]),
		EventID:    cast.ToString(data["eventId"]),
		EventLabel: cast.ToString(data["eventLabel"]),
		Event:      cast.ToString(data["event"]),
		Status:     TaskStatus(cast.ToInt(data["status"])),
		Retries:    cast.ToInt(data["retries"]),
		ExecServer: cast.ToString(data["execServer"]),
		CreatedAt:  cast.ToInt64(data["createdAt"]),
		ExecuteAt:  cast.ToInt64(data["executeAt"]),
		UpdatedAt:  cast.ToInt64(data["updatedAt"]),
	}
}

// NewTaskFromJson 从JSON字符串创建Task实例
// v 参数是一个符合Task结构的JSON字符串
// 返回创建的Task实例
func NewTaskFromJson(v string) *Task {
	var data Task
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &Task{}
	}
	return &data
}

// Clone 创建当前Task实例的深拷贝
// 如果接收者为nil，则返回空的Task对象
// 返回一个与当前实例数据相同但独立的新实例
func (t *Task) Clone() *Task {
	if t == nil {
		return &Task{}
	}
	return &Task{
		ID:         t.ID,
		EventID:    t.EventID,
		EventLabel: t.EventLabel,
		Event:      t.Event,
		Status:     t.Status,
		Retries:    t.Retries,
		ExecServer: t.ExecServer,
		CreatedAt:  t.CreatedAt,
		ExecuteAt:  t.ExecuteAt,
		UpdatedAt:  t.UpdatedAt,
	}
}
