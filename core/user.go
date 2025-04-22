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

// User 表示系统中的用户对象，包含用户的基本信息和权限设置
type User struct {
	// ID 用户的唯一标识符
	ID string `json:"id" gorm:"primaryKey"`
	// Name 用户名，唯一且可索引
	Name string `json:"name" gorm:"unique;index"`
	// Email 用户邮箱，需要加密存储
	Email string `json:"email" gorm:"index"`
	// Phone 用户电话，需要加密存储
	Phone string `json:"phone" gorm:"index"`
	// Avatar 用户头像URL
	Avatar string `json:"avatar"`
	// Nickname 用户昵称
	Nickname string `json:"nickname"`
	// Password 用户密码
	Password string `json:"password"`
	// Gender 用户性别
	Gender string `json:"gender"`
	// SystemRole 系统角色
	SystemRole constant.USER_ROLE `json:"systemRole"`
	// PrivateSalt 用户私有加密盐值
	PrivateSalt string `json:"privateSalt"`
	// UniqueCode 用户唯一身份码
	UniqueCode string `json:"uniqueCode" gorm:"index"`
	// Introduce 用户自我介绍
	Introduce string `json:"introduce"`
	// CreatedAt 创建时间戳
	CreatedAt int64 `json:"createdAt"`
	// CreatedBy 创建者ID
	CreatedBy string `json:"createdBy"`
	// UpdatedAt 更新时间戳
	UpdatedAt int64 `json:"updatedAt"`
	// DeletedBy 删除操作执行者
	DeletedBy string `json:"deletedBy"`
	// DeletedAt 删除时间戳
	DeletedAt int64 `json:"deletedAt" gorm:"index"`
}

// Clone 创建当前User实例的深拷贝
// 如果接收者为nil，则返回空的User对象
// 返回一个与当前实例数据相同但独立的新实例
func (u *User) Clone() *User {
	if u == nil {
		return &User{}
	}
	// 创建新的User对象
	clone := &User{
		ID:          u.ID,
		Name:        u.Name,
		Email:       u.Email,
		Phone:       u.Phone,
		Avatar:      u.Avatar,
		Nickname:    u.Nickname,
		Password:    u.Password,
		Gender:      u.Gender,
		SystemRole:  u.SystemRole,
		PrivateSalt: u.PrivateSalt,
		UniqueCode:  u.UniqueCode,
		Introduce:   u.Introduce,
		CreatedAt:   u.CreatedAt,
		CreatedBy:   u.CreatedBy,
		UpdatedAt:   u.UpdatedAt,
		DeletedBy:   u.DeletedBy,
		DeletedAt:   u.DeletedAt,
	}
	return clone
}

// NewUserFromMap 从map类型数据创建User实例
// v 参数应该是一个包含User字段值的map[string]interface{}
// 返回创建的User实例
func NewUserFromMap(v interface{}) *User {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &User{}
	}

	return &User{
		ID:          cast.ToString(data["id"]),
		Name:        cast.ToString(data["name"]),
		Email:       cast.ToString(data["email"]),
		Phone:       cast.ToString(data["phone"]),
		Avatar:      cast.ToString(data["avatar"]),
		Nickname:    cast.ToString(data["nickname"]),
		Password:    cast.ToString(data["password"]),
		Gender:      cast.ToString(data["gender"]),
		SystemRole:  constant.USER_ROLE(cast.ToInt32(data["systemRole"])),
		PrivateSalt: cast.ToString(data["privateSalt"]),
		UniqueCode:  cast.ToString(data["uniqueCode"]),
		Introduce:   cast.ToString(data["introduce"]),
		CreatedAt:   cast.ToInt64(data["createdAt"]),
		CreatedBy:   cast.ToString(data["createdBy"]),
		UpdatedAt:   cast.ToInt64(data["updatedAt"]),
		DeletedBy:   cast.ToString(data["deletedBy"]),
		DeletedAt:   cast.ToInt64(data["deletedAt"]),
	}
}

// NewUserFromJson 从JSON字符串创建User实例
// v 参数是一个符合User结构的JSON字符串
// 返回创建的User实例
func NewUserFromJson(v string) *User {
	var data User
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &User{}
	}
	return &data
}
