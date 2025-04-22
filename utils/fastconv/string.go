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

package fastconv

import (
	"strings"
	"unsafe"
)

// StringToBytes 将字符串转换为字节切片
//
// 参数:
//
//	s - 要转换的字符串
//
// 返回值:
//
//	转换后的字节切片
func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// BytesToString 将字节切片转换为字符串
//
// 参数:
//
//	b - 要转换的字节切片
//
// 返回值:
//
//	转换后的字符串
func BytesToString(b []byte) string {
	// 使用 unsafe 包中的函数将字节切片转换为字符串，避免内存复制
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// SafeSplit 安全地分割字符串，避免零拷贝越界导致的分割错误。
// 该函数会先检查输入字符串是否为空，若为空则直接返回空切片，避免不必要的处理。
// 此外，函数会先对输入字符串进行克隆，确保原始字符串不会被修改。
//
// 参数:
//   - s: 需要拆分的字符串。
//   - sep: 用于拆分字符串的分隔符。
//
// 返回值:
//   - []string: 拆分后的字符串切片。如果输入字符串为空，则返回空切片。
func SafeSplit(s string, sep string) []string {
	// 如果输入字符串为空，直接返回空切片
	if s == "" {
		return []string{}
	}

	// 克隆输入字符串，确保原始字符串不会被修改
	cs := strings.Clone(s)

	// 使用分隔符拆分字符串并返回结果
	return strings.Split(cs, sep)
}

// SafeSplitFromBytes 安全地分割字节切片，避免零拷贝越界导致的分割错误。
// 该函数会先检查输入字节切片是否为空，若为空则直接返回空切片，避免不必要的处理。
// 函数内部先将字节切片转换为字符串后进行分割操作。
//
// 参数:
//   - b: 需要拆分的字节切片。
//   - sep: 用于拆分字符串的分隔符。
//
// 返回值:
//   - []string: 拆分后的字符串切片。如果输入字节切片为空，则返回空切片。
func SafeSplitFromBytes(b []byte, sep string) []string {
	// 如果输入字节切片为空，直接返回空切片
	if len(b) == 0 {
		return []string{}
	}
	s := string(b)
	return strings.Split(s, sep)
}
