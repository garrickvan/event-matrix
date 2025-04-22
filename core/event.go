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
	"crypto/sha1"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/spf13/cast"
)

// EVENT_GATEWAY_SVR_NAME 事件网关服务名称常量
const EVENT_GATEWAY_SVR_NAME = "event_gateway"

// Event 表示系统中的事件对象，用于描述业务操作和状态变更
type Event struct {
	// ID 事件的唯一标识符
	ID string `json:"id" gorm:"primaryKey"`
	// Project 项目号，标识事件所属项目
	Project string `json:"project" gorm:"index"`
	// Version 版本号，标识事件所属版本
	Version string `json:"version"`
	// Context 上下文号，标识事件所属上下文环境
	Context string `json:"context"`
	// Entity 实体号，标识事件所属实体
	Entity string `json:"entity"`
	// Event 事件号，标识事件类型
	Event string `json:"event" gorm:"index"`
	// Source 事件来源，标识事件发起者
	Source string `json:"source" gorm:"index"`
	// Params 事件参数，JSON格式字符串
	Params string `json:"params"`
	// AccessToken 访问令牌，用于权限验证
	AccessToken string `json:"accessToken"`
	// CreatedAt 创建时间戳
	CreatedAt int64 `json:"createdAt"`
	// Sign 签名，用于验证事件完整性
	Sign string `json:"sign"`
	// raw 原始数据，不序列化到JSON
	raw string `json:"-"`
}

// NewEventFromBytes 从字节数组创建Event实例
// data 参数是包含Event数据的JSON字节数组
// 返回创建的Event实例和可能的错误
func NewEventFromBytes(data []byte) (*Event, error) {
	var e Event
	err := jsonx.UnmarshalFromBytes(data, &e)
	if err != nil {
		return nil, err
	}
	e.raw = fastconv.BytesToString(data)
	return &e, nil
}

// NewEventFromStr 从字符串创建Event实例
// data 参数是包含Event数据的JSON字符串
// 返回创建的Event实例和可能的错误
func NewEventFromStr(data string) (*Event, error) {
	var e Event
	err := jsonx.UnmarshalFromStr(data, &e)
	if err != nil {
		return nil, err
	}
	e.raw = data
	return &e, nil
}

// NewEventFromMap 从map类型数据创建Event实例
// v 参数应该是一个包含Event字段值的map[string]interface{}
// 返回创建的Event实例
func NewEventFromMap(v interface{}) *Event {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &Event{}
	}
	return &Event{
		ID:          cast.ToString(data["id"]),
		Project:     cast.ToString(data["project"]),
		Version:     cast.ToString(data["version"]),
		Context:     cast.ToString(data["context"]),
		Entity:      cast.ToString(data["entity"]),
		Event:       cast.ToString(data["event"]),
		Source:      cast.ToString(data["source"]),
		Params:      cast.ToString(data["params"]),
		AccessToken: cast.ToString(data["accessToken"]),
		CreatedAt:   cast.ToInt64(data["createdAt"]),
		Sign:        cast.ToString(data["sign"]),
	}
}

// Clone 创建当前Event实例的深拷贝
// 如果接收者为nil，则返回空的Event对象
// 返回一个与当前实例数据相同但独立的新实例
func (e *Event) Clone() *Event {
	if e == nil {
		return &Event{}
	}
	return &Event{
		ID:          e.ID,
		Project:     e.Project,
		Version:     e.Version,
		Context:     e.Context,
		Entity:      e.Entity,
		Event:       e.Event,
		Source:      e.Source,
		Params:      e.Params,
		AccessToken: e.AccessToken,
		CreatedAt:   e.CreatedAt,
		Sign:        e.Sign,
	}
}

// Raw 获取事件的原始JSON字符串
// 如果原始字符串为空，则尝试将当前实例序列化为JSON字符串
// 返回事件的原始JSON字符串
func (e *Event) Raw() string {
	if e == nil {
		return ""
	}
	if e.raw == "" {
		raw, err := jsonx.MarshalToStr(e)
		if err != nil {
			return ""
		}
		e.raw = raw
	}
	return e.raw
}

// IsEmpty 检查事件是否为空
// 当事件为nil或签名为空时返回true
// 返回事件是否为空的布尔值
func (e *Event) IsEmpty() bool {
	return e == nil || len(e.Sign) == 0
}

// GenerateSign 为事件生成签名
// 使用事件的各个字段组合后进行SHA1哈希计算，并将结果存储在Sign字段
func (e *Event) GenerateSign() {
	if e == nil {
		return
	}
	parts := []string{
		e.ID,
		e.Project,
		e.Version,
		e.Context,
		e.Entity,
		e.Event,
		e.Source,
		e.Params,
		e.AccessToken,
		strconv.FormatInt(e.CreatedAt, 10),
	}
	signString := strings.Join(parts, "")
	signByte := sha1.Sum([]byte(signString))
	e.Sign = hex.EncodeToString(signByte[:])
}

// VerifySign 验证事件签名是否有效
// 通过重新计算签名并与事件的Sign字段比较来验证
// 返回签名是否有效的布尔值
func (e *Event) VerifySign() bool {
	if e == nil || e.Sign == "" {
		return false
	}
	parts := []string{
		e.ID,
		e.Project,
		e.Version,
		e.Context,
		e.Entity,
		e.Event,
		e.Source,
		e.Params,
		e.AccessToken,
		strconv.FormatInt(e.CreatedAt, 10),
	}
	signString := strings.Join(parts, "")
	signByte := sha1.Sum([]byte(signString))
	return hex.EncodeToString(signByte[:]) == e.Sign
}

// GetFullEventLabel 获取完整的事件标签
// 返回形如 sys.user.avatar->update 的标识字符串
func (e *Event) GetFullEventLabel() string {
	parts := []string{
		e.Project,
		".",
		e.Context,
		".",
		e.Entity,
		"->",
		e.Event,
	}
	return strings.Join(parts, "")
}

// GetUniqueLabel 获取事件的唯一标签
// 返回形如 sys.user.avatar->update@0.1.0 的唯一标识字符串
func (e *Event) GetUniqueLabel() string {
	if e == nil {
		return ""
	}
	parts := []string{
		e.Project,
		".",
		e.Context,
		".",
		e.Entity,
		"->",
		e.Event,
		"@",
		e.Version,
	}
	return strings.Join(parts, "")
}

// GetEntityUrl 获取实体的URL路径
// 返回形如 /sys/user/avatar 的URL路径字符串
func (e *Event) GetEntityUrl() string {
	if e == nil {
		return ""
	}
	parts := []string{
		"/",
		e.Project,
		"/",
		e.Context,
		"/",
		e.Entity,
	}
	return strings.Join(parts, "")
}

// GetEntityLabel 获取实体的标签
// 返回形如 sys.user.personal_info 的实体标签字符串
func (e *Event) GetEntityLabel() string {
	if e == nil {
		return ""
	}
	parts := []string{
		e.Project,
		".",
		e.Context,
		".",
		e.Entity,
	}
	return strings.Join(parts, "")
}

// GetVersionEntityLabel 获取带版本的实体标签
// 返回形如 sys.user.avatar@1.0.0 的带版本实体标签字符串
func (e *Event) GetVersionEntityLabel() string {
	if e == nil {
		return ""
	}
	parts := []string{
		e.Project,
		".",
		e.Context,
		".",
		e.Entity,
		"@",
		e.Version,
	}
	return strings.Join(parts, "")
}

// GetTabelName 获取实体对应的数据表名
// 返回由上下文和实体名组合而成的表名字符串
func (e *Event) GetTabelName() string {
	if e == nil {
		return ""
	}
	parts := []string{
		e.Context,
		e.Entity,
	}
	return strings.Join(parts, "_")
}
