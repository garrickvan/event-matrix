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

package common

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
)

func HandleTask(task types.WorkerTaskExecutor, ctx types.WorkerContext) error {
	// 运行拦截器
	if stop := runInterceptors(ctx); stop {
		return nil
	}
	// 执行任务
	return taskTimeoutInvoker(task, ctx)
}

// 任务超时调用
func taskTimeoutInvoker(task types.WorkerTaskExecutor, ctx types.WorkerContext) error {
	entityEvent := ctx.EntityEvent()
	event := ctx.Event()
	bodyBytes := ctx.Body()
	// 设置默认超时时间为 10 秒
	if entityEvent.Timeout <= 0 {
		entityEvent.Timeout = 10
	}
	t := time.Duration(entityEvent.Timeout) * time.Second

	ip := ctx.IP()
	timeoutCtx, cancel := context.WithTimeout(context.Background(), t)
	resultStatus := make(chan core.TaskStatus, 1)
	defer cancel()

	// 启动任务的 goroutine，使用 defer 和 recover 防止 panic 崩溃
	go func() {
		defer func() {
			if r := recover(); r != nil {
				var errStr string
				if err, ok := r.(error); ok {
					errStr = err.Error()
				}
				logx.Log().Error("Task execution panicked: " + errStr)
				resultStatus <- core.TaskStatusFailed
			}
			close(resultStatus)
		}()

		// 捕获 task 返回的状态或错误
		status := task(ctx)
		resultStatus <- status
	}()

	// 处理 select 语句的结果
	select {
	case result := <-resultStatus:
		if entityEvent.Logable {
			// 保存任务日志
			core.SaveEventLog(ip, "", event.Source, ctx.UserId(), fastconv.BytesToString(bodyBytes), result.Code(), event, ctx.Server().ServerId())
		}
		// 运行过滤器
		for _, filter := range ctx.Server().Filters() {
			if filter == nil {
				continue
			}
			var jsResp *jsonx.JsonResponse
			if result == core.TaskStatusSuccess {
				jsResp = jsonx.DefaultJson(constant.SUCCESS)
			} else {
				jsResp = jsonx.DefaultJson(constant.FAIL_TO_PROCESS)
			}
			if stop := filter(ctx, jsResp); stop {
				return nil
			}
		}
		// 返回任务执行结果
		response := []string{
			strconv.Itoa(int(result)),
			ctx.Server().ServerId(),
		}
		return ctx.SetStatus(http.StatusOK).ResponseString(strings.Join(response, constant.SPLIT_CHAR))
	case <-timeoutCtx.Done():
		// 操作超时，返回任务超时状态
		if entityEvent.Logable {
			core.SaveEventLog(ip, "任务超时", event.Source, ctx.UserId(), string(bodyBytes), core.TaskStatusTimeout.Code(), event, ctx.Server().ServerId())
		}
		response := []string{
			string(core.TaskStatusTimeout.Code()),
			ctx.Server().ServerId(),
		}
		return ctx.SetStatus(http.StatusRequestTimeout).Response([]byte(strings.Join(response, constant.SPLIT_CHAR)))
	}
}
