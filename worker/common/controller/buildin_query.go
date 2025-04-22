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
	"strings"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

func QueryExecutor(ctx types.WorkerContext) (*jsonx.JsonResponse, int) {
	event := ctx.Event()
	if event == nil {
		return jsonx.DefaultJson(constant.EVENT_NOT_EXIST), http.StatusOK
	}
	entityAttrs, paramSettings, params, errJson := ctx.ValidatedParams()
	if errJson != nil {
		return errJson, http.StatusOK
	}

	_, hasPage := core.FindParamFromArray("page", paramSettings)
	_, hasSize := core.FindParamFromArray("page_size", paramSettings)
	if !hasPage {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "缺少必须参数page"
		return errRespone, http.StatusOK
	}
	if !hasSize {
		errRespone := jsonx.DefaultJson(constant.MISSING_PARAM)
		errRespone.Message = "缺少必须参数page_size"
		return errRespone, http.StatusOK
	}
	deleted, ok := params["deleted"]
	if !ok {
		deleted = false
	}
	// 构建查询条件
	var count int64
	countQuery := buildQuerySchema(ctx, event, paramSettings, params, entityAttrs, cast.ToBool(deleted))
	// 查询总数
	countQuery = countQuery.Count(&count)
	if countQuery.Error != nil {
		logx.Log().Error("查询错误：" + countQuery.Error.Error())
		return jsonx.DefaultJson(constant.FAIL_TO_QUERY), http.StatusOK
	}
	result := jsonx.DefaultJsonWithMsg(constant.SUCCESS, "查询成功")
	if count == 0 {
		result.Message = "查询结果为空"
		return result, http.StatusOK
	}
	// 构建查询分页信息
	page := cast.ToInt(params["page"])
	pageSize := cast.ToInt(params["page_size"])
	query := buildQuerySchema(ctx, event, paramSettings, params, entityAttrs, cast.ToBool(deleted))
	// 排序
	for _, v := range paramSettings {
		order_by := ""
		if v.Type == "order_by" {
			order_by += v.Name + " " + v.Range + " "
		}
		if order_by != "" {
			query = query.Order(order_by)
		}
	}
	// 分页
	query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	queryData := make([]map[string]interface{}, 0)
	query = query.Find(&queryData)
	if query.Error != nil {
		logx.Log().Error("查询错误：" + query.Error.Error())
		return jsonx.DefaultJson(constant.FAIL_TO_QUERY), http.StatusOK
	}
	// 构建返回结果
	for i := 0; i < len(queryData); i++ {
		for _, v := range entityAttrs {
			if v.IsSecrecy {
				delete(queryData[i], v.Code)
			}
		}
	}
	jsonx.SetJsonList[map[string]interface{}](result, queryData, count, page)
	return result, http.StatusOK
}

func buildQuerySchema(
	ctx types.WorkerContext,
	event *core.Event,
	paramSettings []core.EventParam,
	params map[string]interface{},
	entityAttrs []core.EntityAttribute,
	deleted bool,
) *gorm.DB {
	db := ctx.Server().Repo().Use(event.Project).Table(event.GetTabelName())
	for _, v := range paramSettings {
		if v.Name == "page" || v.Name == "page_size" || v.Name == "deleted" {
			continue
		}
		if v.Type == "order_by" {
			continue
		}
		if v.Type == "and_query" {
			db = buildQuery(db, &v, params, entityAttrs, true)
			continue
		}
		if v.Type == "or_query" {
			db = buildQuery(db, &v, params, entityAttrs, false)
			continue
		}
	}
	if deleted {
		db = db.Where("deleted_at != 0")
	} else {
		db = db.Where("deleted_at = 0")
	}
	return db
}

func buildQuery(db *gorm.DB, setting *core.EventParam, params map[string]interface{}, entityAttrs []core.EntityAttribute, isAnd bool) *gorm.DB {
	arg, ok := params[setting.Name]
	if !ok {
		return db
	}
	var attr *core.EntityAttribute
	for _, v := range entityAttrs {
		if v.Code == setting.Name {
			attr = &v
			break
		}
	}
	if attr == nil {
		return db
	}
	// 自定义字段类型不支持内置的查询方式
	if attr.FieldType == string(core.CUSTOM_FIELD_TYPE) {
		return db
	}
	switch setting.Range {
	case "any":
		return db
	case "in":
		return queryIn(db, attr, arg, isAnd)
	case "nin":
		return queryNin(db, attr, arg, isAnd)
	case "length":
		return queryLength(db, attr, arg, isAnd)
	case "r_like":
		return queryRLike(db, attr, arg, isAnd)
	case "l_like":
		return queryLLike(db, attr, arg, isAnd)
	case "a_like":
		return queryALike(db, attr, arg, isAnd)
	case "gt":
		return queryGt(db, attr, arg, isAnd)
	case "gte":
		return queryGte(db, attr, arg, isAnd)
	case "lt":
		return queryLt(db, attr, arg, isAnd)
	case "lte":
		return queryLte(db, attr, arg, isAnd)
	case "range":
		return queryRange(db, attr, arg, isAnd)
	case "eq_range":
		return queryEqRange(db, attr, arg, isAnd)
	case "out":
		return queryOut(db, attr, arg, isAnd)
	case "eq_out":
		return queryEqOut(db, attr, arg, isAnd)
	default:
		logx.Log().Warn("未支持的And查询范围类型: " + setting.Range + " 字段: " + setting.Name)
	}
	return db
}

func queryIn(
	db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool,
) *gorm.DB {
	if arg == nil {
		return db
	}
	args := strings.Split(cast.ToString(arg), ",")
	switch attr.FieldType {
	case "string", "id", "constant", "text", "ref", "uid", "url", "email", "phone":
		if isAnd {
			return db.Where(attr.Code+" IN (?)", args)
		} else {
			return db.Or(attr.Code+" IN (?)", args)
		}
	case "int8":
		int8Args := make([]int8, len(args))
		for i, v := range args {
			int8Args[i] = cast.ToInt8(v)
		}
		if isAnd {
			return db.Where(attr.Code+" IN (?)", int8Args)
		} else {
			return db.Or(attr.Code+" IN (?)", int8Args)
		}
	case "int16":
		int16Args := make([]int16, len(args))
		for i, v := range args {
			int16Args[i] = cast.ToInt16(v)
		}
		if isAnd {
			return db.Where(attr.Code+" IN (?)", int16Args)
		} else {
			return db.Or(attr.Code+" IN (?)", int16Args)
		}
	case "int32":
		int32Args := make([]int32, len(args))
		for i, v := range args {
			int32Args[i] = cast.ToInt32(v)
		}
		if isAnd {
			return db.Where(attr.Code+" IN (?)", int32Args)
		} else {
			return db.Or(attr.Code+" IN (?)", int32Args)
		}
	case "int64", "datetime":
		int64Args := make([]int64, len(args))
		for i, v := range args {
			int64Args[i] = cast.ToInt64(v)
		}
		if isAnd {
			return db.Where(attr.Code+" IN (?)", int64Args)
		} else {
			return db.Or(attr.Code+" IN (?)", int64Args)
		}
	case "float32":
		float32Args := make([]float32, len(args))
		for i, v := range args {
			float32Args[i] = cast.ToFloat32(v)
		}
		if isAnd {
			return db.Where(attr.Code+" IN (?)", float32Args)
		} else {
			return db.Or(attr.Code+" IN (?)", float32Args)
		}
	case "float64":
		float64Args := make([]float64, len(args))
		for i, v := range args {
			float64Args[i] = cast.ToFloat64(v)
		}
		if isAnd {
			return db.Where(attr.Code+" IN (?)", float64Args)
		} else {
			return db.Or(attr.Code+" IN (?)", float64Args)
		}
	case "boolean":
		boolArgs := make([]bool, len(args))
		for i, v := range args {
			boolArgs[i] = cast.ToBool(v)
		}
		if isAnd {
			return db.Where(attr.Code+" IN (?)", boolArgs)
		} else {
			return db.Or(attr.Code+" IN (?)", boolArgs)
		}
	default:
		if isAnd {
			logx.Log().Warn("未支持的And in查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		} else {
			logx.Log().Warn("未支持的Or in查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		}
	}
	return db
}

func queryNin(
	db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool,
) *gorm.DB {
	if arg == nil {
		return db
	}
	args := strings.Split(cast.ToString(arg), ",")
	switch attr.FieldType {
	case "string", "id", "constant", "text", "ref", "uid", "url", "email", "phone":
		if isAnd {
			return db.Where(attr.Code+" NOT IN (?)", args)
		} else {
			return db.Or(attr.Code+" NOT IN (?)", args)
		}
	case "int8":
		int8Args := make([]int8, len(args))
		for i, v := range args {
			int8Args[i] = cast.ToInt8(v)
		}
		if isAnd {
			return db.Where(attr.Code+" NOT IN (?)", int8Args)
		} else {
			return db.Or(attr.Code+" NOT IN (?)", int8Args)
		}
	case "int16":
		int16Args := make([]int16, len(args))
		for i, v := range args {
			int16Args[i] = cast.ToInt16(v)
		}
		if isAnd {
			return db.Where(attr.Code+" NOT IN (?)", int16Args)
		} else {
			return db.Or(attr.Code+" NOT IN (?)", int16Args)
		}
	case "int32":
		int32Args := make([]int32, len(args))
		for i, v := range args {
			int32Args[i] = cast.ToInt32(v)
		}
		if isAnd {
			return db.Where(attr.Code+" NOT IN (?)", int32Args)
		} else {
			return db.Or(attr.Code+" NOT IN (?)", int32Args)
		}
	case "int64", "datetime":
		int64Args := make([]int64, len(args))
		for i, v := range args {
			int64Args[i] = cast.ToInt64(v)
		}
		if isAnd {
			return db.Where(attr.Code+" NOT IN (?)", int64Args)
		} else {
			return db.Or(attr.Code+" NOT IN (?)", int64Args)
		}
	case "float32":
		float32Args := make([]float32, len(args))
		for i, v := range args {
			float32Args[i] = cast.ToFloat32(v)
		}
		if isAnd {
			return db.Where(attr.Code+" NOT IN (?)", float32Args)
		} else {
			return db.Or(attr.Code+" NOT IN (?)", float32Args)
		}
	case "float64":
		float64Args := make([]float64, len(args))
		for i, v := range args {
			float64Args[i] = cast.ToFloat64(v)
		}
		if isAnd {
			return db.Where(attr.Code+" NOT IN (?)", float64Args)
		} else {
			return db.Or(attr.Code+" NOT IN (?)", float64Args)
		}
	case "boolean":
		boolArgs := make([]bool, len(args))
		for i, v := range args {
			boolArgs[i] = cast.ToBool(v)
		}
		if isAnd {
			return db.Where(attr.Code+" NOT IN (?)", boolArgs)
		} else {
			return db.Or(attr.Code+" NOT IN (?)", boolArgs)
		}
	default:
		if isAnd {
			logx.Log().Warn("未支持的And not in查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		} else {
			logx.Log().Warn("未支持的Or not in查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		}
	}
	return db
}

func queryLength(
	db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool,
) *gorm.DB {
	if arg == nil {
		return db
	}
	args := strings.Split(cast.ToString(arg), ",")
	if len(args) < 1 {
		return db
	}
	min := cast.ToInt(args[0])
	max := min
	if len(args) >= 2 {
		max = cast.ToInt(args[1])
	}

	switch attr.FieldType {
	case "string", "id", "constant", "text", "ref", "uid", "url", "email", "phone":
		if min == max {
			if isAnd {
				return db.Where("LENGTH("+attr.Code+") = ?", min)
			} else {
				return db.Or("LENGTH("+attr.Code+") = ?", min)
			}
		}
		if isAnd {
			db = db.Where("LENGTH("+attr.Code+") >= ?", min)
			return db.Where("LENGTH("+attr.Code+") <= ?", max)
		} else {
			db = db.Or("LENGTH("+attr.Code+") >= ?", min)
			return db.Or("LENGTH("+attr.Code+") <= ?", max)
		}
	default:
		if isAnd {
			logx.Log().Warn("未支持的And length查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		} else {
			logx.Log().Warn("未支持的Or length查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		}
	}
	return db
}

func queryRLike(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}
	switch attr.FieldType {
	case "string", "id", "constant", "text", "ref", "uid", "url", "email", "phone":
		if isAnd {
			return db.Where(attr.Code+" LIKE ?", cast.ToString(arg)+"%")
		} else {
			return db.Or(attr.Code+" LIKE ?", cast.ToString(arg)+"%")
		}
	default:
		if isAnd {
			logx.Log().Warn("未支持的And r_like查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		} else {
			logx.Log().Warn("未支持的Or r_like查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		}
	}
	return db
}

func queryLLike(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}
	switch attr.FieldType {
	case "string", "id", "constant", "text", "ref", "uid", "url", "email", "phone":
		if isAnd {
			return db.Where(attr.Code+" LIKE ?", "%"+cast.ToString(arg))
		} else {
			return db.Or(attr.Code+" LIKE ?", "%"+cast.ToString(arg))
		}
	default:
		if isAnd {
			logx.Log().Warn("未支持的And l_like查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		} else {
			logx.Log().Warn("未支持的Or l_like查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		}
	}
	return db
}

func queryALike(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}
	switch attr.FieldType {
	case "string", "id", "constant", "text", "ref", "uid", "url", "email", "phone":
		if isAnd {
			return db.Where(attr.Code+" LIKE ?", "%"+cast.ToString(arg)+"%")
		} else {
			return db.Or(attr.Code+" LIKE ?", "%"+cast.ToString(arg)+"%")
		}
	default:
		if isAnd {
			logx.Log().Warn("未支持的And a_like查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		} else {
			logx.Log().Warn("未支持的Or a_like查询字段类型: " + attr.FieldType + " 字段: " + attr.Code)
		}
	}
	return db
}

func queryGt(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}

	applyLteQuery := func(db *gorm.DB, code string, value interface{}) *gorm.DB {
		if isAnd {
			return db.Where(code+" > ?", value)
		}
		return db.Or(code+" > ?", value)
	}

	switch attr.FieldType {
	case "int8":
		return applyLteQuery(db, attr.Code, cast.ToInt8(arg))
	case "int16":
		return applyLteQuery(db, attr.Code, cast.ToInt16(arg))
	case "int32":
		return applyLteQuery(db, attr.Code, cast.ToInt32(arg))
	case "int64", "datetime":
		return applyLteQuery(db, attr.Code, cast.ToInt64(arg))
	case "float32":
		return applyLteQuery(db, attr.Code, cast.ToFloat32(arg))
	case "float64":
		return applyLteQuery(db, attr.Code, cast.ToFloat64(arg))
	default:
		logx.Log().Warn(fmt.Sprintf("未支持的%s gt查询字段类型: %s 字段: %s",
			ifThenElse(isAnd, "And", "Or"), attr.FieldType, attr.Code))
		return db
	}
}

func queryGte(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}

	applyLteQuery := func(db *gorm.DB, code string, value interface{}) *gorm.DB {
		if isAnd {
			return db.Where(code+" >= ?", value)
		}
		return db.Or(code+" >= ?", value)
	}

	switch attr.FieldType {
	case "int8":
		return applyLteQuery(db, attr.Code, cast.ToInt8(arg))
	case "int16":
		return applyLteQuery(db, attr.Code, cast.ToInt16(arg))
	case "int32":
		return applyLteQuery(db, attr.Code, cast.ToInt32(arg))
	case "int64", "datetime":
		return applyLteQuery(db, attr.Code, cast.ToInt64(arg))
	case "float32":
		return applyLteQuery(db, attr.Code, cast.ToFloat32(arg))
	case "float64":
		return applyLteQuery(db, attr.Code, cast.ToFloat64(arg))
	default:
		logx.Log().Warn(fmt.Sprintf("未支持的%s gte查询字段类型: %s 字段: %s",
			ifThenElse(isAnd, "And", "Or"), attr.FieldType, attr.Code))
		return db
	}
}

func queryLt(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}

	applyLteQuery := func(db *gorm.DB, code string, value interface{}) *gorm.DB {
		if isAnd {
			return db.Where(code+" < ?", value)
		}
		return db.Or(code+" < ?", value)
	}

	switch attr.FieldType {
	case "int8":
		return applyLteQuery(db, attr.Code, cast.ToInt8(arg))
	case "int16":
		return applyLteQuery(db, attr.Code, cast.ToInt16(arg))
	case "int32":
		return applyLteQuery(db, attr.Code, cast.ToInt32(arg))
	case "int64", "datetime":
		return applyLteQuery(db, attr.Code, cast.ToInt64(arg))
	case "float32":
		return applyLteQuery(db, attr.Code, cast.ToFloat32(arg))
	case "float64":
		return applyLteQuery(db, attr.Code, cast.ToFloat64(arg))
	default:
		logx.Log().Warn(fmt.Sprintf("未支持的%s lt查询字段类型: %s 字段: %s",
			ifThenElse(isAnd, "And", "Or"), attr.FieldType, attr.Code))
		return db
	}
}

func queryLte(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}

	applyLteQuery := func(db *gorm.DB, code string, value interface{}) *gorm.DB {
		if isAnd {
			return db.Where(code+" <= ?", value)
		}
		return db.Or(code+" <= ?", value)
	}

	switch attr.FieldType {
	case "int8":
		return applyLteQuery(db, attr.Code, cast.ToInt8(arg))
	case "int16":
		return applyLteQuery(db, attr.Code, cast.ToInt16(arg))
	case "int32":
		return applyLteQuery(db, attr.Code, cast.ToInt32(arg))
	case "int64", "datetime":
		return applyLteQuery(db, attr.Code, cast.ToInt64(arg))
	case "float32":
		return applyLteQuery(db, attr.Code, cast.ToFloat32(arg))
	case "float64":
		return applyLteQuery(db, attr.Code, cast.ToFloat64(arg))
	default:
		logx.Log().Warn(fmt.Sprintf("未支持的%s lte查询字段类型: %s 字段: %s",
			ifThenElse(isAnd, "And", "Or"), attr.FieldType, attr.Code))
		return db
	}
}

func ifThenElse(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

func queryRange(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}

	// 解析范围参数
	args := strings.Split(cast.ToString(arg), ",")
	if len(args) < 1 {
		return db
	}
	min, max := args[0], args[0]
	if len(args) > 1 {
		max = args[1]
	}

	// 通用逻辑抽象
	applyRangeQuery := func(db *gorm.DB, code string, minVal, maxVal interface{}) *gorm.DB {
		if min == max {
			if isAnd {
				return db.Where(code+" = ?", minVal)
			}
			return db.Or(code+" = ?", minVal)
		}
		if isAnd {
			return db.Where(code+" > ?", minVal).Where(code+" < ?", maxVal)
		}
		return db.Or(code+" > ?", minVal).Or(code+" < ?", maxVal)
	}

	// 根据字段类型进行动态处理
	switch attr.FieldType {
	case "int8", "int16", "int32", "int64", "datetime":
		return applyRangeQuery(db, attr.Code, cast.ToInt64(min), cast.ToInt64(max))
	case "float32":
		return applyRangeQuery(db, attr.Code, cast.ToFloat32(min), cast.ToFloat32(max))
	case "float64":
		return applyRangeQuery(db, attr.Code, cast.ToFloat64(min), cast.ToFloat64(max))
	default:
		logx.Log().Warn(fmt.Sprintf("未支持的%s range查询字段类型: %s 字段: %s", ifThenElse(isAnd, "And", "Or"), attr.FieldType, attr.Code))
		return db
	}
}

func queryEqRange(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}

	// 解析范围参数
	args := strings.Split(cast.ToString(arg), ",")
	if len(args) < 1 {
		return db
	}
	min := args[0]
	max := args[0]
	if len(args) > 1 {
		max = args[1]
	}

	// 通用逻辑抽象
	applyQuery := func(db *gorm.DB, code string, minVal, maxVal interface{}) *gorm.DB {
		if min == max {
			if isAnd {
				return db.Where(code+" = ?", minVal)
			}
			return db.Or(code+" = ?", minVal)
		}
		if isAnd {
			return db.Where(code+" >= ?", minVal).Where(code+" <= ?", maxVal)
		}
		return db.Or(code+" >= ?", minVal).Or(code+" <= ?", maxVal)
	}

	// 根据字段类型进行动态处理
	switch attr.FieldType {
	case "int8", "int16", "int32", "int64", "datetime":
		return applyQuery(db, attr.Code, cast.ToInt64(min), cast.ToInt64(max))
	case "float32":
		return applyQuery(db, attr.Code, cast.ToFloat32(min), cast.ToFloat32(max))
	case "float64":
		return applyQuery(db, attr.Code, cast.ToFloat64(min), cast.ToFloat64(max))
	default:
		logx.Log().Warn(fmt.Sprintf("未支持的%s eq_range查询字段类型: %s 字段: %s", ifThenElse(isAnd, "And", "Or"), attr.FieldType, attr.Code))
		return db
	}
}

func queryOut(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}

	// 解析范围参数
	args := strings.Split(cast.ToString(arg), ",")
	if len(args) < 1 {
		return db
	}
	min := args[0]
	max := args[0]
	if len(args) > 1 {
		max = args[1]
	}

	// 通用逻辑抽象
	applyQuery := func(db *gorm.DB, code string, minVal, maxVal interface{}) *gorm.DB {
		if min == max {
			if isAnd {
				return db.Where(code+" != ?", minVal)
			}
			return db.Or(code+" != ?", minVal)
		}
		if isAnd {
			return db.Where(code+" < ?", minVal).Or(code+" > ?", maxVal)
		}
		return db.Or(code+" < ?", minVal).Or(code+" > ?", maxVal)
	}

	// 根据字段类型进行动态处理
	switch attr.FieldType {
	case "int8", "int16", "int32", "int64", "datetime":
		return applyQuery(db, attr.Code, cast.ToInt64(min), cast.ToInt64(max))
	case "float32":
		return applyQuery(db, attr.Code, cast.ToFloat32(min), cast.ToFloat32(max))
	case "float64":
		return applyQuery(db, attr.Code, cast.ToFloat64(min), cast.ToFloat64(max))
	default:
		logx.Log().Warn(fmt.Sprintf("未支持的%s out查询字段类型: %s 字段: %s", ifThenElse(isAnd, "And", "Or"), attr.FieldType, attr.Code))
		return db
	}
}

func queryEqOut(db *gorm.DB, attr *core.EntityAttribute, arg interface{}, isAnd bool) *gorm.DB {
	if arg == nil {
		return db
	}

	// 解析范围参数
	args := strings.Split(cast.ToString(arg), ",")
	if len(args) < 1 {
		return db
	}
	min := args[0]
	max := args[0]
	if len(args) > 1 {
		max = args[1]
	}

	// 定义通用处理逻辑
	applyQuery := func(db *gorm.DB, code string, minVal, maxVal interface{}) *gorm.DB {
		if min == max {
			if isAnd {
				return db.Where(code+" != ?", minVal)
			}
			return db.Or(code+" != ?", minVal)
		}
		if isAnd {
			return db.Where(code+" <= ?", minVal).Or(code+" >= ?", maxVal)
		}
		return db.Or(code+" <= ?", minVal).Or(code+" >= ?", maxVal)
	}

	// 根据字段类型进行动态处理
	switch attr.FieldType {
	case "int8", "int16", "int32", "int64", "datetime":
		return applyQuery(db, attr.Code, cast.ToInt64(min), cast.ToInt64(max))
	case "float32":
		return applyQuery(db, attr.Code, cast.ToFloat32(min), cast.ToFloat32(max))
	case "float64":
		return applyQuery(db, attr.Code, cast.ToFloat64(min), cast.ToFloat64(max))
	default:
		logx.Log().Warn(fmt.Sprintf("未支持的%s eq_out查询字段类型: %s 字段: %s", ifThenElse(isAnd, "And", "Or"), attr.FieldType, attr.Code))
		return db
	}
}
