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
	"errors"
	"net/http"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/worker/intranet/dispatcher"
)

type TaskSubmitter struct {
	taskCenterEndpoint string
}

var (
	taskSubmitter     *TaskSubmitter
	TaskEndpointEvent = &core.Event{
		Project: core.INTERNAL_PROJECT,
		Version: constant.INITIAL_VERSION,
		Context: TaskCenterWorkerContext,
		Entity:  TaskCenterWorkerEntity,
	}
)

func init() {
	TaskEndpointEvent.GenerateSign()
}

func NewTaskSubmitter() *TaskSubmitter {
	taskSubmitter = &TaskSubmitter{
		taskCenterEndpoint: "",
	}
	return taskSubmitter
}

func (ts *TaskSubmitter) AddTask(task *core.Task) error {
	if ts == nil {
		return errors.New("任务提交器未初始化")
	}
	if task == nil {
		return errors.New("任务为空")
	}
	if TaskEndpointEvent == nil {
		return errors.New("任务中心事件未初始化")
	}
	if ts.taskCenterEndpoint == "" {
		endpoint := dispatcher.GetWorkerEndpoint(TaskEndpointEvent)
		if endpoint == "" {
			return errors.New("获取任务中心地址失败，网络错误或任务中心未启动")
		} else {
			ts.taskCenterEndpoint = endpoint
		}
	}
	resp, err := dispatcher.Event(ts.taskCenterEndpoint, GW_T_W_TASK_CENTER_ADD_TASK, task, nil)
	if err != nil {
		ts.taskCenterEndpoint = "" // 网络错误，清空taskCenterEndpoint
		return err
	}
	if resp.Status() != http.StatusOK {
		return errors.New(resp.TemporaryData())
	}
	return nil
}
