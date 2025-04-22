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
	"encoding/json"
	"testing"
	"time"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/spf13/cast"
)

// 准备测试数据
func mockEntityEventData() map[string]interface{} {
	return map[string]interface{}{
		"id":           "test-id",
		"entityId":     "entity-123",
		"name":         "test event",
		"code":         "EVENT_001",
		"executorType": 1,
		"executor":     "system",
		"delay":        1000,
		"timeout":      5000,
		"params":       "{}",
		"mode":         "ASYNC",
		"logable":      true,
		"authType":     2,
		"createdAt":    time.Now().Unix(),
		"updatedAt":    time.Now().Unix(),
		"deletedAt":    0,
		"deletedBy":    "",
		"creator":      "admin",
	}
}

// 原始cast方式
func BenchmarkParseWithCast(b *testing.B) {
	data := mockEntityEventData()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = &core.EntityEvent{
			ID:           cast.ToString(data["id"]),
			EntityID:     cast.ToString(data["entityId"]),
			Name:         cast.ToString(data["name"]),
			Code:         cast.ToString(data["code"]),
			ExecutorType: constant.EXECUTOR_TYPE(cast.ToUint8(data["executorType"])),
			Executor:     cast.ToString(data["executor"]),
			Delay:        cast.ToInt(data["delay"]),
			Timeout:      cast.ToInt(data["timeout"]),
			Params:       cast.ToString(data["params"]),
			Mode:         constant.EVENT_MODE(cast.ToString(data["mode"])),
			Logable:      cast.ToBool(data["logable"]),
			AuthType:     constant.AUTH_TYPE(cast.ToUint8(data["authType"])),
			CreatedAt:    cast.ToInt64(data["createdAt"]),
			UpdatedAt:    cast.ToInt64(data["updatedAt"]),
			DeletedAt:    cast.ToInt64(data["deletedAt"]),
			DeletedBy:    cast.ToString(data["deletedBy"]),
			Creator:      cast.ToString(data["creator"]),
		}
	}
}

// 优化后的json方式
func BenchmarkParseWithJson(b *testing.B) {
	data := mockEntityEventData()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		jsonData, _ := json.Marshal(data)
		var e core.EntityEvent
		_ = json.Unmarshal(jsonData, &e)
	}
}

// 测试结果
/*
BenchmarkParseWithCast-8         1641268               658.9 ns/op             0 B/op          0 allocs/op
BenchmarkParseWithJson-8           96928             12039 ns/op            2081 B/op         49 allocs/op

*/
