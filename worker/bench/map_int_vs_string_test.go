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

package bench

import (
	"strconv"
	"testing"
)

const testSize = 10000

func BenchmarkIntMap(b *testing.B) {
	m := make(map[int]int)
	for i := 0; i < testSize; i++ {
		m[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < testSize; j++ {
			_ = m[j]
		}
	}
}

func BenchmarkStringMap(b *testing.B) {
	m := make(map[string]int)
	for i := 0; i < testSize; i++ {
		m[strconv.Itoa(i)] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < testSize; j++ {
			_ = m[strconv.Itoa(j)]
		}
	}
}

// 结果
// BenchmarkIntMap-8             25          43112306 ns/op               0 B/op          0 allocs/op
// BenchmarkStringMap-8           6         200109965 ns/op         7718882 B/op     999900 allocs/op
