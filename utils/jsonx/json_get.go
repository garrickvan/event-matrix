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

package jsonx

import "github.com/tidwall/gjson"

// 注意：gjson使用了零拷贝技术，获取string时需要保证底层字符串不可变，或持续持有结果需要复制

// GetInt64FromJson 从JSON字符串中提取指定路径的int64值
// 如果路径不存在或值类型不匹配，返回0
func GetInt64FromJson(json string, path string) int64 {
	result := gjson.Get(json, path)
	if result.Exists() {
		return result.Int()
	}
	return 0
}

// GetStringFromJson 从JSON字符串中提取指定路径的字符串值
// 如果路径不存在，返回空字符串
func GetStringFromJson(json string, path string) string {
	result := gjson.Get(json, path)
	if result.Exists() {
		return result.String()
	}
	return ""
}

// GetBoolFromJson 从JSON字符串中提取指定路径的布尔值
// 如果路径不存在，返回false
func GetBoolFromJson(json string, path string) bool {
	result := gjson.Get(json, path)
	if result.Exists() {
		return result.Bool()
	}
	return false
}

// GetFloat64FromJson 从JSON字符串中提取指定路径的float64值
// 如果路径不存在或值类型不匹配，返回0.0
func GetFloat64FromJson(json string, path string) float64 {
	result := gjson.Get(json, path)
	if result.Exists() {
		return result.Float()
	}
	return 0.0
}

// GetArrayFromJson 从JSON字符串中提取指定路径的数组
// 如果路径不存在，返回空数组
func GetArrayFromJson(json string, path string) []gjson.Result {
	result := gjson.Get(json, path)
	if result.Exists() {
		return result.Array()
	}
	return []gjson.Result{}
}

// GetMapFromJson 从JSON字符串中提取指定路径的对象
// 如果路径不存在，返回空map
func GetMapFromJson(json string, path string) map[string]gjson.Result {
	result := gjson.Get(json, path)
	if result.Exists() {
		return result.Map()
	}
	return map[string]gjson.Result{}
}

// IsJson 检查字符串是否是有效的JSON格式
func IsJson(strVal string) bool {
	return gjson.Valid(strVal)
}

// UnmarshalToMap 将JSON字符串解析为map[string]interface{}
// 如果解析失败，返回空map
func UnmarshalToMap(json string) map[string]interface{} {
	m, ok := gjson.Parse(json).Value().(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return m
}
