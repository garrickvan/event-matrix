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
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
)

type EntityRecordForDataMgrParam struct {
	Endpoint    string `json:"endpoint"`
	Project     string `json:"project"`
	Context     string `json:"context"`
	Entity      string `json:"entity"`
	Version     string `json:"version"`
	SearchField string `json:"searchField"`
	SearchValue string `json:"searchValue"`
	Deleted     bool   `json:"deleted"`
	Page        int    `json:"page"`
	Size        int    `json:"pageSize"`
}

func OnEntityListForDataMgrHandler(ctx types.WorkerContext, paramStr string) error {
	var param EntityRecordForDataMgrParam
	if err := jsonx.UnmarshalFromStr(paramStr, &param); err != nil {
		return ctx.SetStatus(http.StatusInternalServerError).ResponseBuiltinJson(constant.INVALID_PARAM)
	}
	if param.Project == "" || param.Context == "" || param.Entity == "" {
		return ctx.SetStatus(http.StatusInternalServerError).ResponseBuiltinJson(constant.INVALID_PARAM)
	}
	if param.Page <= 0 {
		param.Page = 1
	}
	if param.Size <= 0 {
		param.Size = 10
	}
	tableName := param.Context + "_" + param.Entity
	db := ctx.Server().Repo().Use(param.Project).Table(tableName)
	if param.SearchField != "" && param.SearchValue != "" {
		db = db.Where(param.SearchField+" LIKE ?", param.SearchValue+"%")
	}
	if param.Deleted {
		db = db.Where("deleted_at != 0")
	} else {
		db = db.Where("deleted_at = 0")
	}
	// 排序
	attrs := ctx.Server().DomainCache().EntityAttrs(types.PathToEntityFromEvent(ctx.Event()))
	if attrs != nil {
		for _, attr := range attrs {
			if attr.Code == "created_at" {
				db = db.Order("created_at DESC")
				break
			}
		}
	}
	db.Offset((param.Page - 1) * param.Size).Limit(param.Size)
	var result []map[string]interface{}
	if err := db.Find(&result).Error; err != nil {
		logx.Log().Error("查询实体列表失败：" + err.Error())
		return ctx.SetStatus(http.StatusInternalServerError).ResponseBuiltinJson(constant.FAIL_TO_QUERY)
	}
	db = ctx.Server().Repo().Use(param.Project).Table(tableName)
	if param.SearchField != "" && param.SearchValue != "" {
		db = db.Where(param.SearchField+" LIKE ?", param.SearchValue+"%")
	}
	if param.Deleted {
		db = db.Where("deleted_at != 0")
	} else {
		db = db.Where("deleted_at = 0")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		logx.Log().Error("查询实体列表失败：" + err.Error())
		return ctx.SetStatus(http.StatusInternalServerError).ResponseBuiltinJson(constant.FAIL_TO_QUERY)
	}
	resp := jsonx.DefaultJsonWithMsg(constant.SUCCESS, "查询成功")
	jsonx.SetJsonList[map[string]interface{}](resp, result, total, param.Page)
	return ctx.SetStatus(http.StatusOK).ResponseJson(resp)
}

type UpdateRecordForDataMgrParam struct {
	Endpoint     string `json:"endpoint"`
	Project      string `json:"project"`
	Context      string `json:"context"`
	Entity       string `json:"entity"`
	RecordId     string `json:"recordId"`
	FieldCode    string `json:"fieldCode"`
	NewValue     string `json:"newValue"`
	VersionLabel string `json:"versionLabel"`
}

func OnUpdateRecordForDataMgrHandler(ctx types.WorkerContext, paramStr string) error {
	// 校验配置是否支持该worker端非规范更新数据
	if notAccept, ok := ctx.Data().(bool); !ok || notAccept {
		return ctx.SetStatus(http.StatusOK).ResponseBuiltinJson(constant.UNSUPPORTED_EVENT)
	}
	var param UpdateRecordForDataMgrParam
	if err := jsonx.UnmarshalFromStr(paramStr, &param); err != nil {
		return ctx.SetStatus(http.StatusOK).ResponseBuiltinJson(constant.INVALID_PARAM)
	}
	param.FieldCode = strings.TrimSpace(param.FieldCode)
	if param.Project == "" || param.Context == "" || param.Entity == "" || param.RecordId == "" || param.FieldCode == "" {
		return ctx.SetStatus(http.StatusOK).ResponseBuiltinJson(constant.INVALID_PARAM)
	}
	tableName := param.Context + "_" + param.Entity
	db := ctx.Server().Repo().Use(param.Project).Table(tableName)
	// 修正传参
	attrs := ctx.Server().DomainCache().EntityAttrs(types.PathToEntity{
		Project: param.Project,
		Version: param.VersionLabel,
		Context: param.Context,
		Entity:  param.Entity,
	})
	attr := core.FindAttrFromArray(param.FieldCode, attrs)
	if attr == nil {
		return ctx.SetStatus(http.StatusOK).ResponseBuiltinJson(constant.INVALID_PARAM)
	}
	var val interface{}
	if attr.FieldType == string(core.CUSTOM_FIELD_TYPE) {
		parser, ok := ctx.Server().Repo().GetCustomFieldParser(strings.TrimSpace(attr.ValueSource))
		if ok && parser != nil {
			val = parser.ParseParam(param.NewValue)
		} else {
			logx.Log().Error("未找到自定义字段[" + attr.ValueSource + "]的解析器")
			return ctx.SetStatus(http.StatusOK).ResponseBuiltinJson(constant.INVALID_PARAM)
		}
	} else {
		val = attr.FixValue(param.NewValue)
	}
	if err := db.Where("id = ?", param.RecordId).Updates(map[string]interface{}{param.FieldCode: val}).Error; err != nil {
		logx.Log().Error("更新实体记录失败：" + err.Error())
		return ctx.SetStatus(http.StatusOK).ResponseBuiltinJson(constant.FAIL_TO_UPDATE)
	}
	resp := jsonx.DefaultJsonWithMsg(constant.SUCCESS, "更新字段值成功")
	return ctx.ResponseJson(resp)
}
