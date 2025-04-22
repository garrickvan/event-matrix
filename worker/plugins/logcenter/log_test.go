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

package logcenter

import (
	"fmt"
	"testing"
)

// 测试
func TestFileName(t *testing.T) {
	// 动态配置的前缀和后缀
	prefix1 := "runtime"
	prefix2 := "event"
	suffix := "slice_log"

	// 测试文件名
	testFileNames := []string{
		"event.202407270236.slice_log",
		"runtime.202407270237.slice_log",
		"invalid.202407270237.slice_log",
		"runtime.20240727.slice_log",
		"event.latest.slice_log",
		"runtime.slice_log",
	}

	for _, fileName := range testFileNames {
		if isValidLogFileName(fileName, prefix1, prefix2, suffix) {
			fmt.Printf("文件名 %s 符合格式\n", fileName)
		} else {
			fmt.Printf("文件名 %s 不符合格式\n", fileName)
		}
	}
}
