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

package constant

import "strconv"

// AUTH_TYPE 定义系统的授权类型
type AUTH_TYPE uint8

const (
	// NONEED_AUTH 公开免登陆授权，适用于公开API
	NONEED_AUTH AUTH_TYPE = 0
	// USER_AUTH 登录用户权限，需要用户登录后才能访问
	USER_AUTH AUTH_TYPE = 1
	// INTERNAL_AUTH 内部调用授权，仅供系统内部组件间调用使用
	INTERNAL_AUTH AUTH_TYPE = 2
)

// USER_ROLE 定义系统中的用户角色层级
type USER_ROLE int32

const (
	BANNED    USER_ROLE = -2         // 封禁用户
	ANONYMOUS USER_ROLE = -1         // 匿名用户
	NORMAL    USER_ROLE = 0          // 普通用户
	DEVELOPER USER_ROLE = 2000000000 // 开发者
	ADMIN     USER_ROLE = 2147483000 // 管理员
	SUPERUSER USER_ROLE = 2147483647 // 超级管理员
)

// String 将用户角色转换为字符串表示
func (u USER_ROLE) String() string {
	return strconv.FormatInt(int64(u), 10)
}

// IsUser 判断当前角色是否为普通用户权限
func (u USER_ROLE) IsUser() bool {
	if u == SUPERUSER {
		return true
	}
	return u >= NORMAL && u < DEVELOPER
}

// IsDeveloper 判断当前角色是否具有开发者权限
func (u USER_ROLE) IsDeveloper() bool {
	if u == SUPERUSER {
		return true
	}
	return u >= DEVELOPER && u < ADMIN
}

// IsAdmin 判断当前角色是否具有管理员权限
func (u USER_ROLE) IsAdmin() bool {
	if u == SUPERUSER {
		return true
	}
	return u >= ADMIN && u < SUPERUSER
}

// IsSuperuser 判断当前角色是否为超级管理员
func (u USER_ROLE) IsSuperuser() bool {
	return u == SUPERUSER
}
