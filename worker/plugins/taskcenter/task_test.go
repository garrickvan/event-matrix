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

package taskcenter

import (
	"fmt"
	"testing"

	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils"
)

func TestAddTask(t *testing.T) {
	// NEEDTO: 初始化dispatcher
	tSubmitter := NewTaskSubmitter()
	err := tSubmitter.AddTask(&core.Task{
		ID:         "task1",
		Event:      "This is a sample event for task 1",
		EventLabel: "Sample Label 1",
		Status:     core.TaskStatusPending,
		Retries:    0,
		CreatedAt:  utils.GetNowMilli(),
		ExecuteAt:  utils.GetNowMilli() + 10000,
		UpdatedAt:  utils.GetNowMilli(),
	})
	if err == nil {
		fmt.Println("添加任务成功")
	} else {
		fmt.Println(err)
		t.Fail()
	}
}
