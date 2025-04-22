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
)

func CreateExecutor(ctx types.WorkerContext) (*jsonx.JsonResponse, int) {
	event := ctx.Event()
	if event == nil {
		return jsonx.DefaultJson(constant.EVENT_NOT_EXIST), http.StatusOK
	}
	entityAttrs, _, params, errJson := ctx.ValidatedParams()
	if errJson != nil {
		return errJson, http.StatusOK
	}
	// 构建数据
	newData := map[string]interface{}{}
	for _, attr := range entityAttrs {
		var preVal interface{}
		// 从参数中获取值
		if val, ok := params[attr.Code]; ok {
			preVal = val
		} else {
			// 若参数中没有值，则使用默认值
			if attr.FieldType == string(core.CUSTOM_FIELD_TYPE) {
				if parser, ok := ctx.Server().Repo().GetCustomFieldParser(attr.ValueSource); ok && parser != nil {
					preVal = parser.DefaultValue()
				}
			} else {
				preVal = attr.GetDefaultVal()
			}
		}
		// 补充必要参数
		if attr.Code == "created_by" && (preVal == nil || preVal == "") {
			userId := ctx.UserId()
			if userId != "" {
				preVal = userId
			}
		}
		newData[attr.Code] = preVal
	}
	// 确保id字段有值
	if id, ok := newData["id"]; !ok || id == nil || id == "" {
		newData["id"] = utils.GenID() // 使用UUID生成器生成唯一ID
	}
	// 唯一数据查重
	for _, attr := range entityAttrs {
		if attr.Unique && attr.Code != "id" {
			val := newData[attr.Code]
			if alreadyExist(event, ctx, attr, val) {
				errRespone := jsonx.DefaultJson(constant.ALREADY_EXIST)
				errRespone.Message = fmt.Sprintf("属性[%s]已存在", attr.Name)
				return errRespone, http.StatusOK
			}
		}
		if attr.Code == "updated_at" && attr.FieldType == string(core.DATETIME_FIELD_TYPE) {
			newData[attr.Code] = utils.GetNowMilli()
		}
		if attr.Code == "created_at" && attr.FieldType == string(core.DATETIME_FIELD_TYPE) {
			newData[attr.Code] = utils.GetNowMilli()
		}
		if attr.Code == "deleted_at" && attr.FieldType == string(core.DATETIME_FIELD_TYPE) {
			newData[attr.Code] = 0
		}
	}
	// 保存数据
	result := ctx.Server().Repo().Use(event.Project).Table(event.GetTabelName()).Create(newData)
	if result.Error != nil {
		return jsonx.DefaultJson(constant.FAIL_TO_CREATE), http.StatusOK
	}
	if result.RowsAffected == 0 {
		return jsonx.DefaultJson(constant.FAIL_TO_CREATE), http.StatusOK
	}
	// 过滤保密字段的数据
	for _, attr := range entityAttrs {
		if attr.IsSecrecy {
			delete(newData, attr.Code)
		}
	}
	// 返回结果
	resp := jsonx.DefaultJson(constant.SUCCESS)
	jsonx.SetJsonList[map[string]interface{}](resp, []map[string]interface{}{newData}, 1, 1)
	return resp, http.StatusOK
}

func alreadyExist(event *core.Event, ctx types.WorkerContext, attr core.EntityAttribute, val interface{}) bool {
	count := int64(0)
	queryMap := map[string]interface{}{
		attr.Code: val,
	}
	ctx.Server().Repo().Use(event.Project).Table(event.GetTabelName()).Where(queryMap).Count(&count)
	if count > 0 {
		return true
	}
	return false
}
