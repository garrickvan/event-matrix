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
	"fmt"
	"net/http"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/spf13/cast"
)

func UpdateExecutor(ctx types.WorkerContext) (*jsonx.JsonResponse, int) {
	event := ctx.Event()
	if event == nil {
		return jsonx.DefaultJson(constant.EVENT_NOT_EXIST), http.StatusOK
	}
	entityAttrs, paramSettings, params, errJson := ctx.ValidatedParams()
	if errJson != nil {
		return errJson, http.StatusOK
	}
	// 检查是否定义了ID参数
	_, hasID := core.FindParamFromArray("id", paramSettings)
	if !hasID {
		errRespone := jsonx.DefaultJsonWithMsg(constant.MISSING_PARAM, "没有定义必要参数[id]")
		return errRespone, http.StatusOK
	}
	// 获取要更新的ID
	id, ok := params["id"]
	if !ok || id == "" {
		errRespone := jsonx.DefaultJsonWithMsg(constant.MISSING_PARAM, "缺少必要的 ID 参数")
		return errRespone, http.StatusOK
	}
	// 构建更新数据
	updateData := map[string]interface{}{}
	for key, val := range params {
		if key == "id" {
			continue
		}
		attr := core.FindAttrFromArray(key, entityAttrs)
		if attr == nil {
			continue
		}
		// 只更新已定义的属性
		if attr.FieldType == string(core.CUSTOM_FIELD_TYPE) {
			updateData[key] = val
		} else {
			// 对基础的数据类型进行值修正
			updateData[key] = attr.FixValue(val)
		}
	}
	// 检查唯一字段是否冲突
	for _, attr := range entityAttrs {
		if attr.Unique && attr.Code != "id" {
			val := updateData[attr.Code]
			if alreadyExistWithID(event, ctx, attr, val, cast.ToString(id)) {
				errRespone := jsonx.DefaultJson(constant.ALREADY_EXIST)
				errRespone.Message = fmt.Sprintf("属性[%s]已存在", attr.Name)
				return errRespone, http.StatusOK
			}
		}
		if attr.Code == "updated_at" && attr.FieldType == string(core.DATETIME_FIELD_TYPE) {
			updateData[attr.Code] = utils.GetNowMilli()
		}
	}
	// 更新数据到数据库
	result := ctx.Server().Repo().Use(event.Project).Table(event.GetTabelName()).Where("id = ?", id).Updates(updateData)
	if result.Error != nil {
		return jsonx.DefaultJson(constant.FAIL_TO_UPDATE), http.StatusOK
	}
	if result.RowsAffected == 0 {
		errJson := jsonx.DefaultJsonWithMsg(constant.FAIL_TO_UPDATE, "更新失败，未找到对应记录")
		return errJson, http.StatusOK
	}
	// 过滤保密字段
	for _, attr := range entityAttrs {
		if attr.IsSecrecy {
			delete(updateData, attr.Code)
		}
	}
	// 返回更新结果
	updateData["id"] = id
	resp := jsonx.DefaultJson(constant.SUCCESS)
	jsonx.SetJsonList[map[string]interface{}](resp, []map[string]interface{}{updateData}, 1, 1)
	return resp, http.StatusOK
}

func alreadyExistWithID(event *core.Event, ctx types.WorkerContext, attr core.EntityAttribute, val interface{}, id string) bool {
	count := int64(0)
	queryMap := map[string]interface{}{
		attr.Code: val,
	}
	// 检查是否存在相同的属性值，但过滤掉相同的ID
	ctx.Server().Repo().Use(event.Project).
		Table(event.GetTabelName()).
		Where(queryMap).
		Where("id != ?", id). // 可以更新相同ID的记录
		Count(&count)
	return count > 0
}
