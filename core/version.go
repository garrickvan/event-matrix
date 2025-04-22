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
	"strconv"
	"strings"

	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/spf13/cast"
)

// Version 表示系统中的版本对象，用于管理项目的版本信息
type Version struct {
	// ID 版本的唯一标识符
	ID string `json:"id" gorm:"primaryKey"`
	// Project 关联的项目标识
	Project string `json:"project" gorm:"index"`
	// Locked 版本是否锁定
	Locked bool `json:"locked"`
	// Online 版本是否在线，一个项目最多支持N个版本在线
	Online bool `json:"online"`
	// MajorVersion 主版本号，当进行不兼容的API更改时增加
	MajorVersion int `json:"majorVersion" gorm:"index"`
	// MinorVersion 次版本号，当添加向后兼容的新功能时增加
	MinorVersion int `json:"minorVersion" gorm:"index"`
	// PatchVersion 修订版本号，当进行向后兼容的问题修复时增加
	PatchVersion int `json:"patchVersion" gorm:"index"`
	// NoteOrLink 版本说明或文档链接
	NoteOrLink string `json:"noteOrLink"`
	// CreatedAt 创建时间戳
	CreatedAt int64 `json:"createdAt"`
	// UpdatedAt 更新时间戳
	UpdatedAt int64 `json:"updatedAt"`
	// Creator 创建者
	Creator string `json:"creator"`
}

// GenVersionLabel 生成版本标签
// 返回形如 "1.0.0" 的版本号字符串，由主版本号、次版本号和修订版本号组成
func (v *Version) GenVersionLabel() string {
	return strings.Join([]string{
		strconv.Itoa(v.MajorVersion),
		strconv.Itoa(v.MinorVersion),
		strconv.Itoa(v.PatchVersion),
	}, ".")
}

// NewVersionFromJson 从JSON字符串创建Version实例
// v 参数是一个符合Version结构的JSON字符串
// 返回创建的Version实例
func NewVersionFromJson(v string) *Version {
	var data Version
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &Version{}
	}
	return &data
}

// NewVersionFromMap 从map类型数据创建Version实例
// v 参数应该是一个包含Version字段值的map[string]interface{}
// 返回创建的Version实例
func NewVersionFromMap(v interface{}) *Version {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &Version{}
	}

	return &Version{
		ID:           cast.ToString(data["id"]),
		Project:      cast.ToString(data["project"]),
		Locked:       cast.ToBool(data["locked"]),
		Online:       cast.ToBool(data["online"]),
		MajorVersion: cast.ToInt(data["majorVersion"]),
		MinorVersion: cast.ToInt(data["minorVersion"]),
		PatchVersion: cast.ToInt(data["patchVersion"]),
		NoteOrLink:   cast.ToString(data["noteOrLink"]),
		CreatedAt:    cast.ToInt64(data["createdAt"]),
		UpdatedAt:    cast.ToInt64(data["updatedAt"]),
		Creator:      cast.ToString(data["creator"]),
	}
}

// Clone 创建当前Version实例的深拷贝
// 如果接收者为nil，则返回空的Version对象
// 返回一个与当前实例数据相同但独立的新实例
func (v *Version) Clone() *Version {
	if v == nil {
		return &Version{}
	}
	return &Version{
		ID:           v.ID,
		Project:      v.Project,
		Locked:       v.Locked,
		Online:       v.Online,
		MajorVersion: v.MajorVersion,
		MinorVersion: v.MinorVersion,
		PatchVersion: v.PatchVersion,
		NoteOrLink:   v.NoteOrLink,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    v.UpdatedAt,
		Creator:      v.Creator,
	}
}
