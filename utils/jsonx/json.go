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

import (
	json "github.com/bytedance/sonic"
	"github.com/garrickvan/event-matrix/utils/fastconv"
)

// Package jsonx 提供统一的JSON序列化和反序列化功能
// 使用高性能的sonic库替代标准库，并提供了便捷的JSON操作方法
// 主要功能：
// 1. JSON序列化与反序列化
// 2. JSON路径查询
// 3. 类型转换和验证

// MarshalToBytes 将任意类型的数据序列化为JSON字节数组
// 参数：
//   - v: 要序列化的数据
//
// 返回：
//   - []byte: 序列化后的JSON字节数组
//   - error: 序列化过程中的错误，如果成功则为nil
func MarshalToBytes(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// MarshalToStr 将任意类型的数据序列化为JSON字符串
// 使用零拷贝技术将字节数组转换为字符串，提高性能
func MarshalToStr(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return fastconv.BytesToString(bytes), nil
}

// UnmarshalFromStr 将JSON字符串反序列化为指定类型
// 使用零拷贝技术将字符串转换为字节数组，提高性能，反序列化结果Sonic内部进行了拷贝，内存安全
func UnmarshalFromStr(data string, v interface{}) error {
	dataBytes := fastconv.StringToBytes(data)
	return json.Unmarshal(dataBytes, v)
}

// UnmarshalFromBytes 将JSON字节数组反序列化为指定类型，反序列化结果Sonic内部进行了拷贝，内存安全
func UnmarshalFromBytes(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
