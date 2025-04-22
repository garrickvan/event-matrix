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
	"fmt"
	"net/http"
	"time"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils"
	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/intranet/dispatcher"
	"github.com/garrickvan/event-matrix/worker/types"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/spf13/cast"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/**
  目前为单机设计，未考虑到多机部署，如果需要拓展性能，需要考虑重新封装
**/

type TaskCenter struct {
	worker           *types.Worker
	svr              types.WorkerServer
	maxInProcessTask int
	inProcessTask    cmap.ConcurrentMap[string, *core.Task]
}

type TaskListParams struct {
	Page        int    `json:"page"`
	Size        int    `json:"size"`
	SearchField string `json:"searchField"`
	SearchValue string `json:"searchValue"`
}

const (
	TaskDB                  = "tasks"
	TaskCenterWorkerContext = "gateway"
	TaskCenterWorkerEntity  = "task_center"

	TASK_ADD_SUCCESS                                      = string(constant.SUCCESS)
	GW_T_W_TASK_CENTER_ADD_TASK types.INTRANET_EVENT_TYPE = 32000
	G_T_W_TASK_CENTER_QUERY     types.INTRANET_EVENT_TYPE = 32001
)

var (
	taskCenterWorker = types.Worker{
		Project:      core.INTERNAL_PROJECT,
		VersionLabel: constant.INITIAL_VERSION,
		Context:      TaskCenterWorkerContext,
		Entity:       TaskCenterWorkerEntity,
		SyncSchema:   false,
	}
)

func NewTaskCenter(svr types.WorkerServer, cfgKey string, maxInProcessTask int) *TaskCenter {
	taskCenterWorker.CfgKey = cfgKey
	tc := &TaskCenter{
		worker:           &taskCenterWorker,
		svr:              svr,
		maxInProcessTask: maxInProcessTask,
		inProcessTask:    cmap.New[*core.Task](),
	}
	return tc
}

/*
**

	初始化任务中心插件
	@param wm worker manager
	@param maxInProcessTask 最大同时处理任务数
*/
func (tc *TaskCenter) Setup() error {
	if err := tc.svr.Repo().AddDBFromSharedConfig(tc.worker.CfgKey); err != nil {
		return err
	}
	if !tc.svr.Repo().HasDB(TaskDB) {
		return fmt.Errorf("缺少数据库配置: [%s], 请检查配置，无法启动任务中心服务", TaskDB)
	}
	if err := tc.svr.Repo().Use(TaskDB).AutoMigrate(&core.Task{}); err != nil {
		return fmt.Errorf("数据库表迁移失败: %w", err)
	}
	if err := tc.svr.RegisterWorker(tc.worker); err != nil {
		return err
	}
	tc.svr.RegisterPlugin(tc)
	go tc.start()      // 处理待启动任务
	go tc.retrieTask() // 处理重试任务
	return nil
}

func (tc *TaskCenter) ReceiveCodes() []types.INTRANET_EVENT_TYPE {
	return []types.INTRANET_EVENT_TYPE{GW_T_W_TASK_CENTER_ADD_TASK, G_T_W_TASK_CENTER_QUERY}
}

func (tc *TaskCenter) Handle(ctx types.WorkerContext, typz types.INTRANET_EVENT_TYPE) error {
	switch typz {
	case GW_T_W_TASK_CENTER_ADD_TASK:
		return tc.addTaskHandler(ctx)
	case G_T_W_TASK_CENTER_QUERY:
		return tc.queryTaskHandler(ctx)
	default:
		return ctx.SetStatus(http.StatusForbidden).Response([]byte(constant.UNSUPPORTED_EVENT))
	}
}

func (tc *TaskCenter) remainingSize() int {
	return tc.maxInProcessTask - tc.inProcessTask.Count()
}

func (tc *TaskCenter) addTask(task *core.Task) bool {
	if tc == nil {
		return false
	}
	// 检查是否有剩余容量
	if tc.inProcessTask.Count() >= tc.maxInProcessTask {
		return false
	}
	if ok := tc.inProcessTask.Has(task.ID); ok {
		// 任务已存在
		return true
	}
	err := tc.saveTaskOnDB(task, core.TaskStatusInProgress)
	if err != nil {
		logx.Log().Error(err.Error())
		return false
	}
	go tc.handlerTask(task)
	tc.inProcessTask.Set(task.ID, task)
	return true
}

func (tc *TaskCenter) finishTask(taskID string, status core.TaskStatus, execServer string) error {
	task, ok := tc.inProcessTask.Get(taskID)
	if !ok || task == nil {
		return errors.New("任务不存在：" + taskID)
	}
	task.ExecServer = execServer
	err := tc.updateTaskToDB(task, status, task.Retries)
	if err != nil {
		return err
	}
	tc.inProcessTask.Remove(taskID)
	return nil
}

func (tc *TaskCenter) addTaskHandler(ctx types.WorkerContext) error {
	task := core.Task{}
	if tc == nil {
		return ctx.SetStatus(http.StatusForbidden).Response([]byte("任务中心插件未初始化"))
	}
	err := jsonx.UnmarshalFromBytes(ctx.Body(), &task)
	if err != nil {
		return ctx.SetStatus(http.StatusInternalServerError).Response([]byte("任务数据解析失败：" + err.Error()))
	}
	if task.ExecuteAt <= utils.GetNowMilli() {
		success := tc.addTask(&task)
		if success {
			return ctx.SetStatus(http.StatusOK).Response([]byte(TASK_ADD_SUCCESS))
		}
	}
	err = tc.saveTaskOnDB(&task, core.TaskStatusPending)
	if err != nil {
		return ctx.SetStatus(http.StatusInternalServerError).Response([]byte("任务保存失败：" + err.Error()))
	}
	return ctx.SetStatus(http.StatusOK).Response([]byte(TASK_ADD_SUCCESS))
}

func (tc *TaskCenter) saveTaskOnDB(task *core.Task, status core.TaskStatus) error {
	// 更新任务的状态
	task.Status = status
	task.UpdatedAt = utils.GetNowMilli() // 手动设置更新时间戳
	db := tc.svr.Repo().Use(TaskDB)
	// 执行 UPSERT 操作，手动指定更新时间
	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}}, // 冲突列，即主键ID
		DoUpdates: clause.Assignments(map[string]interface{}{
			"status":     status,
			"updated_at": task.UpdatedAt, // 明确指定更新时间
		}),
	}).Create(task).Error

	if err != nil {
		return err
	}
	return nil
}

func (tc *TaskCenter) updateTaskToDB(task *core.Task, status core.TaskStatus, retries int) error {
	db := tc.svr.Repo().Use(TaskDB)
	return db.Transaction(func(tx *gorm.DB) error {
		task.Status = status
		task.UpdatedAt = utils.GetNowMilli()
		task.Retries = retries
		result := tx.Model(task).Select("status", "updated_at", "retries", "exec_server").Updates(map[string]interface{}{
			"status":      status,
			"updated_at":  utils.GetNowMilli(),
			"retries":     retries,
			"exec_server": task.ExecServer,
		})
		if result.Error != nil {
			return result.Error
		}
		return nil
	})
}

func (tc *TaskCenter) start() {
	pageSize := 100
	// 每隔3秒从数据库中获取待处理任务，并处理
	for {
		time.Sleep(3 * time.Second)
		pageNo := 1
		remainingSize := tc.remainingSize()
		if remainingSize > 0 {
			// 定义外部的 tasks 切片
			tasks := []core.Task{}
			// 分页从数据库中获取任务状态为待处理，且执行时间已到的任务，直到任务队列填满为止
			for {
				remainingSize := tc.remainingSize()
				if remainingSize <= 0 {
					break
				}
				// 获取数据库中符合条件的任务
				db := tc.svr.Repo().Use(TaskDB).Model(&core.Task{}).
					Where("status = ? AND execute_at <= ?", core.TaskStatusPending, utils.GetNowMilli()).
					Limit(pageSize).Offset((pageNo - 1) * pageSize).Find(&tasks)

				// 处理数据库错误
				if db.Error != nil {
					logx.Log().Error("从数据库中获取任务失败：" + db.Error.Error())
					break
				}

				// 检查是否没有更多任务可获取
				if len(tasks) == 0 {
					break
				}

				// 遍历任务并加入任务队列
				for _, task := range tasks {
					// 在加入任务队列前再次检查队列容量
					if tc.remainingSize() <= 0 {
						break
					}
					// 加入任务
					tc.addTask(&task)
				}

				// 如果任务队列已满，停止分页获取任务
				if tc.remainingSize() <= 0 {
					break
				}

				// 清空 tasks，防止任务重复处理
				tasks = []core.Task{}
				pageNo++
			}
		}
	}
}

/*
*
*
重试机制规则
Retries	 延迟时间 (秒)	 延迟时间 (分钟)
1    5    0.08
2    40   0.72
3    135  2.25
4    320  5.33
5    625   10.42
6    1080  18.00
7    1715  28.58
8    2560  42.00
9    3600  60.00
10   3600  60.00
*/
func (tc *TaskCenter) retrieTask() {
	pageSize := 100

	// 计算重试任务的延时回退时间，最大延时1小时
	calculateBackoff := func(retries int) int64 {
		baseDelay := int64(5)                                      // 每次回退的基本时间为5秒
		maxDelay := int64(3600 * 1000)                             // 最大延时为1小时
		delay := baseDelay * int64(retries*retries*retries) * 1000 // retries³秒 -> 毫秒
		// 限制最大延时为1小时
		if delay > maxDelay {
			return maxDelay
		}
		return delay
	}

	// 每隔10秒从数据库中获取未在 inProcessTask 中但状态为 InProgress 或 Timeout 的任务，并重新加入任务队列，直到任务队列填满为止或没有更多任务可获取
	for {
		time.Sleep(10 * time.Second)
		pageNo := 1

		for {
			remainingSize := tc.remainingSize()
			if remainingSize <= 0 {
				break
			}

			tasks := []core.Task{}
			// 从数据库中分页获取状态为 InProgress 或 Timeout 的任务
			db := tc.svr.Repo().Use(TaskDB).
				Where("status IN (?, ?)", core.TaskStatusInProgress, core.TaskStatusTimeout).
				Limit(pageSize).Offset((pageNo - 1) * pageSize).Find(&tasks)

			if db.Error != nil {
				logx.Log().Error(db.Error.Error())
				break
			}

			// 如果没有更多任务了，退出分页循环
			if len(tasks) == 0 {
				break
			}

			// 遍历任务列表，处理每个任务
			for _, task := range tasks {
				exists := tc.inProcessTask.Has(task.ID)
				// 如果任务不在 inProcessTask 中，则重新添加任务
				if !exists {
					// 计算回退机制的延迟时间
					delay := calculateBackoff(task.Retries)

					// 当前时间戳
					now := utils.GetNowMilli()
					// 检查任务的执行时间是否应该延迟处理
					if now < task.ExecuteAt+delay {
						continue // 如果未达到延迟时间，跳过该任务
					}
					// 检查 remainingSize，避免超出容量
					remainingSize := tc.remainingSize()
					if remainingSize <= 0 {
						break
					}
					// 更新任务的 ExecuteAt 以延迟下次重试时间
					task.ExecuteAt = now + delay
					// 更新任务状态为 Pending 并增加重试次数，同时更新 ExecuteAt
					err := tc.updateTaskToDB(&task, core.TaskStatusPending, task.Retries+1)
					if err != nil {
						logx.Log().Error(err.Error())
						continue
					}
					// 将任务添加到 inProcessTask
					tc.inProcessTask.Set(task.ID, &task)
					// 处理任务（并发处理）
					go tc.handlerTask(&task)
				}
			}
			pageNo++
		}
	}
}

func (tc *TaskCenter) handlerTask(task *core.Task) {
	event, err := core.NewEventFromStr(task.Event)
	if err != nil {
		logx.Log().Error("任务事件解析失败：" + err.Error())
		tc.finishTask(task.ID, core.TaskStatusFailed, "")
		return
	}
	endpoint := dispatcher.GetWorkerEndpoint(event)

	if endpoint == "" {
		logx.Log().Error("任务处理地址为空：" + event.GetUniqueLabel())
		tc.finishTask(task.ID, core.TaskStatusFailed, "")
		return
	}
	// 发送任务到处理器
	status, serverId, err := tc.invokeTaskOnWorker(endpoint, event)
	if err != nil {
		logx.Log().Error("任务处理失败：" + err.Error())
		tc.finishTask(task.ID, core.TaskStatusFailed, serverId)
		return
	}
	tc.finishTask(task.ID, status, serverId)
}

func (tc *TaskCenter) invokeTaskOnWorker(workerEndpiont string, event *core.Event) (core.TaskStatus, string, error) {
	resp, err := dispatcher.Event(workerEndpiont, types.W_T_W_EVENT_CALL, event.Raw(), nil)
	if err != nil {
		return core.TaskStatusFailed, "", err
	}
	result := fastconv.SafeSplit(resp.TemporaryData(), constant.SPLIT_CHAR)
	if len(result) != 2 {
		return core.TaskStatusFailed, "", errors.New("任务处理结果格式错误：" + resp.TemporaryData())
	}
	taskStatus := cast.ToUint8(result[0])
	return core.TaskStatus(taskStatus), result[1], nil
}

func (tc *TaskCenter) queryTaskHandler(ctx types.WorkerContext) error {
	if tc == nil {
		return ctx.SetStatus(http.StatusForbidden).Response([]byte("任务中心插件未初始化"))
	}
	param := TaskListParams{}
	err := jsonx.UnmarshalFromBytes(ctx.Body(), &param)
	if err != nil {
		return ctx.SetStatus(http.StatusForbidden).Response([]byte("查询任务参数解析失败：" + err.Error()))
	}
	db := tc.svr.Repo().Use(TaskDB).Model(&core.Task{})
	var taskList []*core.Task
	resp := jsonx.DefaultJsonWithMsg(constant.SUCCESS, "查询成功")
	if param.SearchValue != "" && param.SearchField != "" {
		if param.SearchField == "status" {
			db.Where("status = ?", param.SearchValue)
		} else {
			db.Where(param.SearchField+" LIKE ?", "%"+param.SearchValue+"%")
		}
	}
	db.
		Offset((param.Page - 1) * param.Size).
		Limit(param.Size).
		Order("created_at desc").
		Find(&taskList)
	if len(taskList) > 0 {
		db := tc.svr.Repo().Use(TaskDB).Model(&core.Task{})
		var count int64
		if param.SearchValue != "" && param.SearchField != "" {
			if param.SearchField == "status" {
				db.Where("status = ?", param.SearchValue)
			} else {
				db.Where(param.SearchField+" LIKE ?", "%"+param.SearchValue+"%")
			}
		}
		db.Count(&count)
		jsonx.SetJsonList[*core.Task](resp, taskList, count, param.Page)
	} else {
		resp.Size = 0
	}
	return ctx.SetStatus(http.StatusOK).ResponseJson(resp)
}
