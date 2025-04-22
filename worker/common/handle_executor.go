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
	"runtime/debug"
	"time"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
)

func HandleExecutor(funz types.WorkerExecutor, ctx types.WorkerContext) error {
	// 运行拦截器
	if stop := runInterceptors(ctx); stop {
		return nil
	}
	// 执行执行器
	return executorTimeoutInvoker(funz, ctx)
}

// 运行拦截器
func runInterceptors(ctx types.WorkerContext) bool {
	for _, intercept := range ctx.Server().Intercepts() {
		if intercept == nil {
			continue
		}
		if stop := intercept(ctx); stop {
			return true
		}
	}
	return false
}

// 执行器超时调用
func executorTimeoutInvoker(funz types.WorkerExecutor, ctx types.WorkerContext) error {
	entityEvent := ctx.EntityEvent()
	event := ctx.Event()
	bodyBytes := ctx.Body()
	// 设置默认超时时间为 3 秒
	if entityEvent.Timeout <= 0 {
		entityEvent.Timeout = 3
	}
	t := time.Duration(entityEvent.Timeout) * time.Second
	ip := ctx.IP()
	timeoutCtx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()

	resultJsResp := make(chan *jsonx.JsonResponse, 1)

	// 启动任务执行的 goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				var errStr string
				if err, ok := r.(error); ok {
					errStr = err.Error()
				} else {
					errStr = "unknown error"
				}
				// 打印调用栈
				stackTrace := debug.Stack()
				logx.Log().Error("worker execution panicked: " + errStr + "\n" + string(stackTrace))
				ctx.SetStatus(http.StatusInternalServerError).ResponseBuiltinJson(constant.FAIL_TO_PROCESS)
				resultJsResp <- nil
			}
		}()

		jsResp, httpStatus := funz(ctx)
		ctx.SetStatus(httpStatus)
		resultJsResp <- jsResp
		close(resultJsResp)
	}()

	select {
	case jsResp := <-resultJsResp:
		// 记录日志
		if entityEvent.Logable && jsResp.Code == string(constant.SUCCESS) {
			comment := ""
			if time.Since(time.UnixMilli(event.CreatedAt)) > constant.SLOW_REQUST_TIME {
				comment = "慢请求"
			}
			userId := ""
			if ctx.UserId() == "" {
				userId = "unknown"
			}
			core.SaveEventLog(ip, comment, event.Source, userId, fastconv.BytesToString(bodyBytes), constant.RESPONSE_CODE(jsResp.Code), event, ctx.Server().ServerId())
		}
		// 运行过滤器
		for _, filter := range ctx.Server().Filters() {
			if filter == nil {
				continue
			}
			if stop := filter(ctx, jsResp); stop {
				return nil
			}
		}
		// 返回 JSON 响应
		if jsResp != nil {
			return ctx.ResponseJson(jsResp)
		} else {
			// 执行自定义返回逻辑
			return ctx.ResponseBuiltinJson(constant.FAIL_TO_PROCESS)
		}
	case <-timeoutCtx.Done():
		// 如果操作超时
		return ctx.SetStatus(http.StatusRequestTimeout).ResponseBuiltinJson(constant.EVENT_TIMEOUT)
	}
}

// 验证事件，并返回事件对象
func ValidatedEvent(bodyBytes []byte) (*core.Event, constant.RESPONSE_CODE) {
	event, err := core.NewEventFromBytes(bodyBytes)
	if err != nil || event.IsEmpty() {
		return nil, constant.UNKNOWN_DATA
	}
	if !event.VerifySign() {
		return nil, constant.INVALID_SIGN
	}
	return event, constant.SUCCESS
}
