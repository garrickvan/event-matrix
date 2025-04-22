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

package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/utils/loadtool"
	"github.com/garrickvan/event-matrix/worker/types"
)

/**
 * 每个worker的心跳检查接口
 */
func checkWorkerHandler(ctx types.WorkerContext, params string) error {
	wids := strings.Split(params, constant.SPLIT_CHAR)
	if len(wids) == 0 {
		return ctx.ResponseBuiltinJson(constant.INVALID_PARAM)
	}
	results := []types.WorkerCheckResult{}
	devicesLoadRate := loadtool.GetLoadRate()
	for _, needCheckId := range wids {
		has := ctx.Server().HasWorker(needCheckId)
		results = append(results, types.WorkerCheckResult{
			WorkerId: needCheckId,
			Exist:    has,
			LoadRate: devicesLoadRate,
		})
	}
	return ctx.SetStatus(http.StatusOK).ResponseJson(results)
}

/**
 * 获取设备负载率接口
 */
func getLoadRateHandler(ctx types.WorkerContext, params string) error {
	loadRate := loadtool.GetLoadRate()
	loadRateStr := strconv.FormatFloat(loadRate, 'f', -1, 64)
	return ctx.SetStatus(http.StatusOK).ResponseString(loadRateStr)
}
