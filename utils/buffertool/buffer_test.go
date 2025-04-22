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

package buffertool

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestBufferPoolGC(t *testing.T) {
	buf, release := GetBuffer(1024)
	release()
	buf[2] = 'x' // 修改对象内容，以便 GC 后验证是否被回收

	// 打印池中的对象（仅用于演示，实际不推荐直接访问池的内容）
	fmt.Println("Buffer in pool:", buf[:10])

	// 强制触发 GC
	runtime.GC()
	time.Sleep(time.Second) // 给 GC 一些时间运行

	// 再次从池中获取对象
	newBuf, _ := GetBuffer(1024)
	fmt.Println("New buffer from pool:", newBuf[:10])

	// 验证是否是同一个对象
	if &buf[0] == &newBuf[0] {
		fmt.Println("The same buffer is reused")
	} else {
		fmt.Println("A new buffer is allocated")
	}
}
