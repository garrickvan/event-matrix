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

import "sync"

// bufferPool 表示一个特定大小的字节缓冲区池
type bufferPool struct {
	pool    *sync.Pool // 底层sync.Pool实例
	maxSize int        // 该池支持的最大缓冲区大小
}

// pools 预定义了一系列不同大小的缓冲区池
// 从64字节到512KB，按2的幂次方递增
var pools = []bufferPool{
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 64) }}, maxSize: 64},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 128) }}, maxSize: 128},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 256) }}, maxSize: 256},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 512) }}, maxSize: 512},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 1*1024) }}, maxSize: 1 * 1024},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 2*1024) }}, maxSize: 2 * 1024},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 4*1024) }}, maxSize: 4 * 1024},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 8*1024) }}, maxSize: 8 * 1024},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 16*1024) }}, maxSize: 16 * 1024},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 32*1024) }}, maxSize: 32 * 1024},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 64*1024) }}, maxSize: 64 * 1024},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 128*1024) }}, maxSize: 128 * 1024},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 256*1024) }}, maxSize: 256 * 1024},
	{pool: &sync.Pool{New: func() interface{} { return make([]byte, 0, 512*1024) }}, maxSize: 512 * 1024},
}

// GetBuffer 获取指定大小的字节缓冲区及其清理函数
// 参数:
//
//	size: 需要的缓冲区大小(字节)
//
// 返回:
//
//	[]byte: 获取到的缓冲区
//	func(): 使用完毕后应调用的清理函数
//
// 注意:
//   - 如果请求大小超过最大池规格(512KB)，会直接分配新缓冲区
//   - 清理函数会将缓冲区归还到合适的池中
func GetBuffer(size int) ([]byte, func()) {
	// 查找匹配的缓冲池
	for _, bp := range pools {
		if size <= bp.maxSize {
			buf := bp.pool.Get().([]byte)
			origCap := cap(buf)
			buf = buf[:size] // 设置用户需要的长度

			return buf, func() {
				// 使用完整切片表达式确保恢复原始容量
				bp.pool.Put(buf[:0:origCap])
			}
		}
	}

	// 超过最大池规格时直接分配
	return make([]byte, size), func() {}
}
