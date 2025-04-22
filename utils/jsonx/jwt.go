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

// Package jsonx 提供JSON处理相关的工具函数和结构体
package jsonx

import (
	"encoding/base64"
	"errors"
	"strings"
)

// GetJwtTokenClaims 从JWT令牌中提取声明（claims）信息
// 不验证令牌签名，仅解析payload部分获取包含的数据
//
// 参数：
//   - token: 完整的JWT令牌字符串
//
// 返回：
//   - map[string]interface{}: 包含令牌中声明的键值对
//   - error: 解析过程中的错误，如格式错误或解码失败
func GetJwtTokenClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("JWT格式异常")
	}
	// 解码第二部分（payload）
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("Error decoding payload: " + err.Error())
	}
	// 使用 JSON 解码 claims
	claims := make(map[string]interface{}, 4) // 8是预估的claims数量
	err = UnmarshalFromBytes(payload, &claims)
	if err != nil {
		return nil, errors.New("Error decoding JSON: " + err.Error())
	}
	return claims, nil
}
