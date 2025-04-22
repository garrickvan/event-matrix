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
	"github.com/garrickvan/event-matrix/utils"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/spf13/cast"
)

// EventLog 表示事件日志，用于记录事件的执行过程和结果
type EventLog struct {
	// ID 日志的唯一标识符
	ID string `json:"id" gorm:"primaryKey"`
	// EventNode 事件节点标识，由project、context、entity和event组合而成
	EventNode string `json:"eventNode" gorm:"index"`
	// Ip 事件发起的IP地址
	Ip string `json:"ip"`
	// EventSource 事件来源
	EventSource string `json:"eventSource"`
	// Comment 日志备注信息
	Comment string `json:"comment"`
	// EventRaw 事件原始数据
	EventRaw string `json:"eventRaw"`
	// FinishAt 事件完成时间戳
	FinishAt int64 `json:"finishAt"`
	// ExecAt 事件执行时间（纳秒）
	ExecAt int64 `json:"execAt"`
	// FinishStatus 事件完成状态码
	FinishStatus constant.RESPONSE_CODE `json:"finishStatus"`
	// ServerId 处理事件的服务器ID
	ServerId string `json:"serverId"`
	// Creator 日志创建者
	Creator string `json:"creator"`
}

// NewEventLogFromJson 从JSON字符串创建EventLog实例
// v 参数是一个符合EventLog结构的JSON字符串
// 返回创建的EventLog实例
func NewEventLogFromJson(v string) *EventLog {
	var data EventLog
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &EventLog{}
	}
	return &data
}

// NewEventLogFromMap 从map类型数据创建EventLog实例
// v 参数应该是一个包含EventLog字段值的map[string]interface{}
// 返回创建的EventLog实例
func NewEventLogFromMap(v interface{}) *EventLog {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &EventLog{}
	}
	return &EventLog{
		ID:           cast.ToString(data["id"]),
		EventNode:    cast.ToString(data["eventNode"]),
		Ip:           cast.ToString(data["ip"]),
		EventSource:  cast.ToString(data["eventSource"]),
		Comment:      cast.ToString(data["comment"]),
		EventRaw:     cast.ToString(data["eventRaw"]),
		FinishAt:     cast.ToInt64(data["finishAt"]),
		ExecAt:       cast.ToInt64(data["execAt"]),
		FinishStatus: constant.RESPONSE_CODE(cast.ToString(data["finishStatus"])),
		ServerId:     cast.ToString(data["serverId"]),
		Creator:      cast.ToString(data["creator"]),
	}
}

// Clone 创建当前EventLog实例的深拷贝
// 如果接收者为nil，则返回空的EventLog对象
// 返回一个与当前实例数据相同但独立的新实例
func (e *EventLog) Clone() *EventLog {
	if e == nil {
		return &EventLog{}
	}
	return &EventLog{
		ID:           e.ID,
		EventNode:    e.EventNode,
		Ip:           e.Ip,
		EventSource:  e.EventSource,
		Comment:      e.Comment,
		EventRaw:     e.EventRaw,
		FinishAt:     e.FinishAt,
		ExecAt:       e.ExecAt,
		FinishStatus: e.FinishStatus,
		ServerId:     e.ServerId,
		Creator:      e.Creator,
	}
}

// SaveEventLog 保存事件日志
// 将事件日志转换为JSON格式并写入日志系统
// 参数:
//   - ip: 事件发起的IP地址
//   - comment: 日志备注信息
//   - source: 事件来源
//   - userID: 用户ID
//   - eventStr: 事件字符串
//   - status: 事件完成状态码
//   - event: 事件对象
//   - serverId: 服务器ID
func SaveEventLog(
	ip, comment, source, userID, eventStr string, status constant.RESPONSE_CODE, event *Event, serverId string,
) {
	log := NewEventLog(ip, comment, source, userID, eventStr, status, event, serverId)
	// 将 SystemLog 转换为 JSON 格式
	jsonData, err := jsonx.MarshalToBytes(log)
	if err != nil {
		logx.Log().Error("Marshal event loog to json failed: " + err.Error())
		return
	}
	if logx.EventLogger == nil {
		logx.Log().Error("Event logger is not initialized")
		return
	}
	// 记录的日志为 info 级别，初始化配置要不高于 info 级别
	logx.EventLogger.Zap().Info(string(jsonData))
}

// NewEventLog 创建新的事件日志实例
// 参数:
//   - ip: 事件发起的IP地址
//   - comment: 日志备注信息
//   - source: 事件来源
//   - userID: 用户ID
//   - eventStr: 事件字符串
//   - status: 事件完成状态码
//   - event: 事件对象
//   - serverId: 服务器ID
//
// 返回创建的EventLog实例
func NewEventLog(
	ip, comment, source, userID, eventStr string, status constant.RESPONSE_CODE, event *Event, serverId string,
) *EventLog {
	startTime := event.CreatedAt
	finishAt := utils.GetNowMilli()
	log := EventLog{
		ID:           utils.GenID(),
		Ip:           ip,
		EventNode:    event.GetFullEventLabel(),
		EventSource:  source,
		Comment:      comment,
		EventRaw:     eventStr,
		FinishAt:     finishAt,
		ExecAt:       finishAt - startTime,
		FinishStatus: status,
		Creator:      userID,
		ServerId:     serverId,
	}
	return &log
}
