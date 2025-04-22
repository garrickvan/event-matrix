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
	"testing"
)

type TestStruct struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

func TestMarshalToStrAndUnmarshalFromStr(t *testing.T) {
	// 准备测试数据
	testData := []TestStruct{
		{Name: "Alice", Age: 25, Email: "alice@example.com"},
		{Name: "Bob", Age: 30, Email: "bob@example.com"},
	}

	// 序列化
	jsonStr, err := MarshalToStr(testData)
	if err != nil {
		t.Fatalf("MarshalToStr failed: %v", err)
	}

	// 反序列化
	var result []TestStruct
	err = UnmarshalFromStr(jsonStr, &result)
	if err != nil {
		t.Fatalf("UnmarshalFromStr failed: %v", err)
	}

	// 验证反序列化结果
	if len(result) != len(testData) {
		t.Fatalf("Expected %d items, got %d", len(testData), len(result))
	}

	for i, item := range result {
		if item.Name != testData[i].Name || item.Age != testData[i].Age || item.Email != testData[i].Email {
			t.Errorf("Mismatch at index %d: expected %+v, got %+v", i, testData[i], item)
		}
	}
}
