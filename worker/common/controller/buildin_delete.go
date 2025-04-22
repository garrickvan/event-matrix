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
	"strings"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/spf13/cast"
)

func DeleteExecutor(ctx types.WorkerContext) (*jsonx.JsonResponse, int) {
	entityAttrs, paramSettings, params, errJson := ctx.ValidatedParams()
	if errJson != nil {
		return errJson, http.StatusOK
	}
	// 检查必要参数ids是否存在
	_, hasIDs := core.FindParamFromArray("ids", paramSettings)
	if !hasIDs {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "没有定义必要参数[ids]"
		return errRespone, http.StatusOK
	}
	if _, ok := params["ids"]; !ok {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "少传必要参数[ids]"
		return errRespone, http.StatusOK
	}
	// 检查参数ids是否合规
	ids := cast.ToString(params["ids"])
	ids = strings.TrimSpace(ids)
	if len(ids) == 0 {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "少传必要参数[ids]"
		return errRespone, http.StatusOK
	}
	idsArray := strings.Split(ids, ",")
	if len(idsArray) == 0 {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "少传必要参数[ids]"
		return errRespone, http.StatusOK
	}
	if len(idsArray) > 200 {
		errRespone := jsonx.DefaultJson(constant.INVALID_PARAM)
		errRespone.Message = "参数[ids]数量过多"
		return errRespone, http.StatusOK
	}
	// 生成软删除参数
	updateParams := map[string]interface{}{
		"deleted_at": utils.GetNowMilli(),
	}
	hasDeletedAt := false
	hasDeletedBy := false
	for _, attr := range entityAttrs {
		if attr.Code == "deleted_at" && attr.FieldType == string(core.DATETIME_FIELD_TYPE) {
			hasDeletedAt = true
		}
		if attr.Code == "deleted_by" && attr.FieldType == string(core.UID_FIELD_TYPE) {
			hasDeletedBy = true
		}
	}
	// 检查是否定义了删除时间
	if !hasDeletedAt {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "实体中没有指定删除时间[deleted_at]，无法执行删除操作"
		return errRespone, http.StatusOK
	}
	// 补充删除者ID
	if hasDeletedBy {
		userId := ctx.UserId()
		if userId != "" {
			updateParams["deleted_by"] = userId
		}
	}
	event := ctx.Event()
	if event == nil {
		return jsonx.DefaultJson(constant.EVENT_NOT_EXIST), http.StatusOK
	}
	// 删除数据
	table := ctx.Server().Repo().Use(event.Project).Table(event.GetTabelName())
	result := table.Where("id IN?", idsArray).Updates(updateParams)
	if result.Error != nil {
		errRespone := jsonx.DefaultJson(constant.FAIL_TO_DELETE)
		errRespone.Message = result.Error.Error()
		return errRespone, http.StatusOK
	}
	if result.RowsAffected == 0 {
		errRespone := jsonx.DefaultJson(constant.FAIL_TO_DELETE)
		errRespone.Message = "没有数据被删除，请检查参数[ids]是否正确"
		return errRespone, http.StatusOK
	}
	// 响应结果
	resp := jsonx.DefaultJsonWithMsg(constant.SUCCESS, "全部删除成功")
	resp.Total = result.RowsAffected
	return resp, http.StatusOK
}

func RestoreExecutor(ctx types.WorkerContext) (*jsonx.JsonResponse, int) {
	entityAttrs, paramSettings, params, errJson := ctx.ValidatedParams()
	if errJson != nil {
		return errJson, http.StatusOK
	}
	// 检查必要参数ids是否存在
	_, hasIDs := core.FindParamFromArray("ids", paramSettings)
	if !hasIDs {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "没有定义必要参数[ids]"
		return errRespone, http.StatusOK
	}
	if _, ok := params["ids"]; !ok {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "少传必要参数[ids]"
		return errRespone, http.StatusOK
	}
	// 检查参数ids是否合规
	ids := cast.ToString(params["ids"])
	ids = strings.TrimSpace(ids)
	if len(ids) == 0 {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "少传必要参数[ids]"
		return errRespone, http.StatusOK
	}
	idsArray := strings.Split(ids, constant.SPLIT_CHAR)
	if len(idsArray) == 0 {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "少传必要参数[ids]"
		return errRespone, http.StatusOK
	}
	if len(idsArray) > 200 {
		errRespone := jsonx.DefaultJson(constant.INVALID_PARAM)
		errRespone.Message = "参数[ids]数量过多"
		return errRespone, http.StatusOK
	}
	// 生成伪删除参数
	updateParams := map[string]interface{}{
		"deleted_at": 0,
	}
	hasDeletedAt := false
	hasDeletedBy := false
	for _, attr := range entityAttrs {
		if attr.Code == "deleted_at" && attr.FieldType == string(core.DATETIME_FIELD_TYPE) {
			hasDeletedAt = true
		}
		if attr.Code == "deleted_by" && attr.FieldType == string(core.UID_FIELD_TYPE) {
			hasDeletedBy = true
		}
	}
	// 检查是否定义了删除时间
	if !hasDeletedAt {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "实体中没有指定删除时间[deleted_at]，无法执行恢复操作"
		return errRespone, http.StatusOK
	}
	// 清空删除者ID
	if hasDeletedBy {
		updateParams["deleted_by"] = ""
	}
	event := ctx.Event()
	if event == nil {
		return jsonx.DefaultJson(constant.EVENT_NOT_EXIST), http.StatusOK
	}
	// 恢复数据
	table := ctx.Server().Repo().Use(event.Project).Table(event.GetTabelName())
	result := table.Where("id IN?", idsArray).Updates(updateParams)
	if result.Error != nil {
		errRespone := jsonx.DefaultJson(constant.FAIL_TO_PROCESS)
		errRespone.Message = result.Error.Error()
		return errRespone, http.StatusOK
	}
	if result.RowsAffected == 0 {
		errRespone := jsonx.DefaultJson(constant.FAIL_TO_PROCESS)
		errRespone.Message = "没有数据被恢复，请检查参数[ids]是否正确"
		return errRespone, http.StatusOK
	}
	// 响应结果
	resp := jsonx.DefaultJsonWithMsg(constant.SUCCESS, "全部恢复成功")
	resp.Total = result.RowsAffected
	return resp, http.StatusOK
}
