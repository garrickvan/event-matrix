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
	"github.com/garrickvan/event-matrix/database"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/worker/types"
)

func SqlExecutor(ctx types.WorkerContext) (*jsonx.JsonResponse, int) {
	_, paramSettings, params, errJson := ctx.ValidatedParams()
	// 移除sql参数，统一用settings中的sql参数
	delete(params, "sql")
	if errJson != nil {
		return errJson, http.StatusOK
	}
	sqlSet, hasSql := core.FindParamFromArray("sql", paramSettings)
	if !hasSql || strings.TrimSpace(sqlSet.RangeValue) == "" {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "没有正确定义必要参数[sql]，无法执行事件"
		return errRespone, http.StatusOK
	}
	// 补充执行者ID
	executorSet, hasExecutor := core.FindParamFromArray("executor", paramSettings)
	if hasExecutor && executorSet.Type == string(core.UID_FIELD_TYPE) {
		userId := ctx.UserId()
		if userId != "" {
			params["executor"] = userId
		}
	}
	sqlType := strings.TrimSpace(sqlSet.Range)
	sqlStatement := strings.TrimSpace(sqlSet.RangeValue)
	event := ctx.Event()
	if event == nil {
		return jsonx.DefaultJson(constant.EVENT_NOT_EXIST), http.StatusOK
	}
	switch sqlType {
	case "normal":
		return execSql(sqlStatement, params, event, ctx, false), http.StatusOK
	case "transaction":
		return execSql(sqlStatement, params, event, ctx, true), http.StatusOK
	case "query":
		return execQuerySql(sqlStatement, params, event, ctx, false), http.StatusOK
	case "transaction-query":
		return execQuerySql(sqlStatement, params, event, ctx, true), http.StatusOK
	}
	// 未知的SQL类型
	return jsonx.DefaultJsonWithMsg(constant.FAIL_TO_PROCESS, "未知的SQL类型"), http.StatusOK
}

func execSql(
	sqlStatement string,
	params map[string]interface{},
	event *core.Event,
	ctx types.WorkerContext,
	isTransaction bool,
) *jsonx.JsonResponse {
	var count int64
	var err error
	db := ctx.Server().Repo().Use(event.Project).Table(event.GetTabelName())

	if isTransaction {
		count, err = database.TransactionRawSqlExec(db, sqlStatement, params)
	} else {
		count, err = database.RawSqlExec(db, sqlStatement, params)
	}

	if err != nil {
		return jsonx.DefaultJsonWithMsg(constant.FAIL_TO_PROCESS, err.Error())
	} else {
		resp := jsonx.DefaultJson(constant.SUCCESS)
		resp.Total = count
		return resp
	}
}

func execQuerySql(
	sqlStatement string,
	params map[string]interface{},
	event *core.Event,
	ctx types.WorkerContext,
	isTransaction bool,
) *jsonx.JsonResponse {
	var err error
	var rows []map[string]interface{}
	db := ctx.Server().Repo().Use(event.Project).Table(event.GetTabelName())

	if isTransaction {
		rows, err = database.TransactionRawQuerySqlExec(db, sqlStatement, params)
	} else {
		rows, err = database.RawQuerySqlExec(db, sqlStatement, params)
	}

	if err != nil {
		return jsonx.DefaultJsonWithMsg(constant.FAIL_TO_PROCESS, err.Error())
	}
	resp := jsonx.DefaultJsonWithMsg(constant.SUCCESS, "查询成功")
	for _, row := range rows {
		resp.List = append(resp.List, row)
	}
	resp.Size = len(resp.List)
	return resp
}
