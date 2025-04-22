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

import (
	"strings"

	"github.com/spf13/cast"
)

// SERVER_MODE 定义服务器运行模式的类型
type SERVER_MODE string

const (
	// DEV 开发模式
	DEV SERVER_MODE = "dev"
	// PROD 生产模式
	PROD SERVER_MODE = "prod"
)

// IsProdMode 判断当前服务是否运行在生产模式
// 参数 mode 可以是任意类型，函数会将其转换为字符串进行比较
func IsProdMode(mode interface{}) bool {
	if mode == nil {
		return false
	}
	mode = strings.ToLower(cast.ToString(mode))
	return mode == string(PROD) || mode == "production"
}
