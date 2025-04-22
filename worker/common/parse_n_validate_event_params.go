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
	"fmt"
	"net/url"
	"strings"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/spf13/cast"
)

/*
*
  - 获取请求的基础参数，用例：
    entityAttrs, paramSettings, params, errJson := wm.GetBaseRequestParams(event)
    if errJson!= nil {
    return errJson
    }
    // 业务逻辑处理...
*/
func ParseAndValidateParams(ctx types.WorkerContext) (
	[]core.EntityAttribute, []core.EventParam, map[string]interface{}, *jsonx.JsonResponse,
) {
	emptyEntityAttrs := []core.EntityAttribute{}
	emptyParamSettings := []core.EventParam{}
	emptyParams := map[string]interface{}{}

	event := ctx.Event()
	if event == nil {
		return emptyEntityAttrs, emptyParamSettings, emptyParams, jsonx.DefaultJson(constant.EVENT_NOT_EXIST)
	}

	entityEvent := ctx.EntityEvent()
	if entityEvent == nil {
		return emptyEntityAttrs, emptyParamSettings, emptyParams, jsonx.DefaultJson(constant.ENTITY_NOT_EXIST)
	}

	if entityEvent == nil {
		return emptyEntityAttrs, emptyParamSettings, emptyParams, jsonx.DefaultJson(constant.EVENT_NOT_EXIST)
	}
	entityAttrs := ctx.Server().DomainCache().EntityAttrs(types.PathToEntityFromEvent(event))
	if entityAttrs == nil || len(entityAttrs) == 0 {
		return emptyEntityAttrs, emptyParamSettings, emptyParams, jsonx.DefaultJson(constant.ENTITY_NOT_EXIST)
	}
	params := map[string]interface{}{}
	err := jsonx.UnmarshalFromStr(event.Params, &params)
	if err != nil {
		logx.Debug(event.GetFullEventLabel()+"参数解析失败: ", err)
		return emptyEntityAttrs, emptyParamSettings, emptyParams, jsonx.DefaultJson(constant.INVALID_PARAM)
	}
	paramSettings := []core.EventParam{}
	err = jsonx.UnmarshalFromStr(entityEvent.Params, &paramSettings)
	if err != nil {
		logx.Debug(event.GetFullEventLabel()+"参数设置解析失败: ", err, "\n", entityEvent.Params)
		return emptyEntityAttrs, emptyParamSettings, emptyParams, jsonx.DefaultJson(constant.INVALID_PARAM)
	}
	// 按参数设置进行参数校验
	for name, param := range params {
		setting, hasSet := core.FindParamFromArray(name, paramSettings)
		var attr *core.EntityAttribute
		if hasSet {
			attr = core.FindAttrFromArray(setting.Name, entityAttrs)
			// 根据已配置的表属性类型，校正参数类型
			if attr != nil &&
				setting.Type != string(core.AND_QUERY_FIELD_TYPE) &&
				setting.Type != string(core.OR_QUERY_FIELD_TYPE) &&
				setting.Type != string(core.ORDER_BY_FIELD_TYPE) {
				if attr.FieldType == string(core.CUSTOM_FIELD_TYPE) {
					parser, ok := ctx.Server().Repo().GetCustomFieldParser(attr.ValueSource)
					if ok && parser != nil {
						params[setting.Name] = parser.ParseParam(cast.ToString(param))
						param = params[setting.Name]
					} else {
						logx.Log().Warn(event.GetFullEventLabel() + "未找到自定义字段解析器: " + attr.ValueSource)
					}
				} else {
					params[setting.Name] = attr.FixValue(param)
					param = params[setting.Name]
				}
			} else {
				// 非实体属性的数据，则根据参数设置的类型进行数值转换
				if setting.Type != string(core.CUSTOM_FIELD_TYPE) {
					params[setting.Name] = core.FixAttributeValue(param, setting.Type)
					param = params[setting.Name]
				} else {
					customFieldParserKey := setting.RangeValue
					customFieldParserKey = strings.TrimSpace(customFieldParserKey)
					parser, ok := ctx.Server().Repo().GetCustomFieldParser(customFieldParserKey)
					if ok && parser != nil {
						params[setting.Name] = parser.ParseParam(cast.ToString(param))
						param = params[setting.Name]
					} else {
						logx.Log().Warn(event.GetFullEventLabel() + "未找到自定义字段解析器: " + customFieldParserKey)
					}
				}
			}
		}
		// 检查是否传入必要参数
		if setting != nil && setting.Required && param == nil {
			errResponse := jsonx.DefaultJson(constant.MISSING_PARAM)
			errResponse.Message = fmt.Sprintf("缺少必要参数: %s", setting.Name)
			return emptyEntityAttrs, emptyParamSettings, emptyParams, errResponse
		}

		// 校验参数是否符合要求
		errJson := validateParam(setting, param, entityAttrs, event, ctx)
		if errJson != nil {
			return emptyEntityAttrs, emptyParamSettings, emptyParams, errJson
		}
	}
	return entityAttrs, paramSettings, params, nil
}

func validateParam(
	setting *core.EventParam,
	param interface{},
	entityAttrs []core.EntityAttribute,
	event *core.Event,
	ctx types.WorkerContext,
) *jsonx.JsonResponse {
	if setting == nil {
		return nil
	}
	switch setting.Type {
	case "string", "id", "text":
		return stringParamValidate(setting, param, event)
	case "ref":
		return refParamValidate(setting, param, entityAttrs, event)
	case "constant":
		return constantParamValidate(setting, param, event, ctx)
	case "url":
		return urlParamValidate(setting, param, event)
	case "email":
		return emailParamValidate(setting, param, event)
	case "phone":
		return phoneParamValidate(setting, param, event)
	case "int8", "int16", "int32", "int64", "datetime", "float32", "float64":
		return numberParamValidate(setting, param, event)
	case "boolean":
		return booleanParamValidate(setting, param, event)
	case "custom":
		return customParamValidate(setting, param, entityAttrs, event, ctx)
	case "and_query", "or_query":
		return nil
	default:
		logx.Log().Warn(event.GetFullEventLabel() + "未知参数类型: " + setting.Name + " " + setting.Type)
	}
	return nil
}

func stringParamValidate(
	setting *core.EventParam,
	param interface{},
	event *core.Event,
) *jsonx.JsonResponse {
	str := cast.ToString(param)
	errJson := jsonx.DefaultJson(constant.INVALID_PARAM)

	switch setting.Range {
	case "any":
		return nil
	case "in":
		rangeVals := strings.Split(setting.RangeValue, ",")
		if !utils.InStrArray(str, rangeVals) {
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "nin":
		rangeVals := strings.Split(setting.RangeValue, ",")
		if utils.InStrArray(str, rangeVals) {
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "length":
		rangeVals := strings.Split(setting.RangeValue, ",")
		if len(rangeVals) == 1 {
			size := cast.ToInt(rangeVals[0])
			if len(str) != size {
				errJson.Message = "参数值长度不符合要求(" + setting.RangeValue + "): " + setting.Name
				return errJson
			}
		} else if len(rangeVals) >= 2 {
			min := cast.ToInt(rangeVals[0])
			max := cast.ToInt(rangeVals[1])
			if len(str) < min || len(str) > max {
				errJson.Message = "参数值长度不符合要求(" + setting.RangeValue + "): " + setting.Name
				return errJson
			}
		} else {
			logx.Log().Warn(event.GetFullEventLabel() + "参数值长度设置错误: " + setting.Name)
		}
	case "r_like":
		rangeVal := setting.RangeValue
		if !strings.HasPrefix(str, rangeVal) {
			errJson.Message = "参数值不是以" + rangeVal + "开头: " + setting.Name
			return errJson
		}
	case "l_like":
		rangeVal := setting.RangeValue
		if !strings.HasSuffix(str, rangeVal) {
			errJson.Message = "参数值不是以" + rangeVal + "结尾: " + setting.Name
			return errJson
		}
	case "a_like":
		rangeVal := setting.RangeValue
		if !strings.Contains(str, rangeVal) {
			errJson.Message = "参数值不包含" + rangeVal + ": " + setting.Name
			return errJson
		}
	default:
		logx.Log().Warn(event.GetFullEventLabel() + "未知校验类型: " + setting.Name + " " + setting.Range)
	}
	return nil
}

func refParamValidate(
	setting *core.EventParam,
	param interface{},
	entityAttrs []core.EntityAttribute,
	event *core.Event,
) *jsonx.JsonResponse {
	idStr := cast.ToString(param)
	if len(idStr) == 0 {
		errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
		errJson.Message = "参数值不能为空: " + setting.Name
		return errJson
	}
	// 不自动校验ref参数是否存在表中，否则可能导致性能问题
	return nil
}

func constantParamValidate(
	setting *core.EventParam,
	param interface{},
	event *core.Event,
	ctx types.WorkerContext,
) *jsonx.JsonResponse {
	constantVal := cast.ToString(param)
	if len(constantVal) == 0 {
		errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
		errJson.Message = "参数值不能为空: " + setting.Name
		return errJson
	}
	vals := strings.Split(setting.RangeValue, ".")
	if len(vals) != 2 {
		logx.Log().Warn(event.GetFullEventLabel() + "常量参数值设置错误: " + setting.Name)
		return nil
	}
	project, dict := vals[0], vals[1]
	constants := ctx.Server().DomainCache().Constants(project, dict)
	if constants == nil || len(constants) == 0 {
		logx.Log().Warn(event.GetFullEventLabel() + "未找到常量定义: " + setting.Name)
		return nil
	}
	for _, constant := range constants {
		if constant.Value == constantVal {
			return nil
		}
	}
	errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
	errJson.Message = "常量参数值不在系统定义中: " + setting.Name
	return errJson
}

func urlParamValidate(
	setting *core.EventParam,
	param interface{},
	event *core.Event,
) *jsonx.JsonResponse {
	urlStr := cast.ToString(param)
	if len(urlStr) > 0 {
		_, err := url.ParseRequestURI(urlStr)
		if err != nil {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不是有效的URL: " + setting.Name
			return errJson
		}
	}
	return nil
}

func emailParamValidate(
	setting *core.EventParam,
	param interface{},
	event *core.Event,
) *jsonx.JsonResponse {
	emailStr := cast.ToString(param)
	if len(emailStr) > 0 {
		if !utils.IsEmail(emailStr) {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不是有效的Email地址: " + setting.Name
			return errJson
		}
	}
	return nil
}

func phoneParamValidate(
	setting *core.EventParam,
	param interface{},
	event *core.Event,
) *jsonx.JsonResponse {
	phoneStr := cast.ToString(param)
	if len(phoneStr) > 0 {
		if !utils.IsPhoneNumber(phoneStr) {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不是有效的手机号码: " + setting.Name
			return errJson
		}
	}
	return nil
}
func numberParamValidate(
	setting *core.EventParam,
	param interface{},
	event *core.Event,
) *jsonx.JsonResponse {
	rangeVals := strings.Split(setting.RangeValue, ",")
	// 将param转换为float64类型，这样能支持float32和float64
	paramFloat := cast.ToFloat64(param)

	switch setting.Range {
	case "any":
		return nil
	case "in":
		// 处理in范围值
		var floatVals []float64
		for _, val := range rangeVals {
			floatVals = append(floatVals, cast.ToFloat64(val))
		}
		if !utils.InFloat64Array(paramFloat, floatVals) {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "nin":
		// 处理nin范围值
		var floatVals []float64
		for _, val := range rangeVals {
			floatVals = append(floatVals, cast.ToFloat64(val))
		}
		if utils.InFloat64Array(paramFloat, floatVals) {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "gt":
		if len(rangeVals) < 1 {
			logx.Log().Warn(event.GetFullEventLabel() + "参数值范围设置错误: " + setting.Name)
			return nil
		}
		rangeVal := cast.ToFloat64(rangeVals[0])
		if paramFloat <= rangeVal {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "gte":
		if len(rangeVals) < 1 {
			logx.Log().Warn(event.GetFullEventLabel() + "参数值范围设置错误: " + setting.Name)
			return nil
		}
		rangeVal := cast.ToFloat64(rangeVals[0])
		if paramFloat < rangeVal {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "lt":
		if len(rangeVals) < 1 {
			logx.Log().Warn(event.GetFullEventLabel() + "参数值范围设置错误: " + setting.Name)
			return nil
		}
		rangeVal := cast.ToFloat64(rangeVals[0])
		if paramFloat >= rangeVal {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "lte":
		if len(rangeVals) < 1 {
			logx.Log().Warn(event.GetFullEventLabel() + "参数值范围设置错误: " + setting.Name)
			return nil
		}
		rangeVal := cast.ToFloat64(rangeVals[0])
		if paramFloat > rangeVal {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "range":
		if len(rangeVals) < 2 {
			logx.Log().Warn(event.GetFullEventLabel() + "参数值范围设置错误: " + setting.Name)
			return nil
		}
		min := cast.ToFloat64(rangeVals[0])
		max := cast.ToFloat64(rangeVals[1])
		if paramFloat <= min || paramFloat >= max {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "eq_range":
		if len(rangeVals) < 2 {
			logx.Log().Warn(event.GetFullEventLabel() + "参数值范围设置错误: " + setting.Name)
			return nil
		}
		min := cast.ToFloat64(rangeVals[0])
		max := cast.ToFloat64(rangeVals[1])
		if paramFloat < min || paramFloat > max {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "out":
		if len(rangeVals) < 2 {
			logx.Log().Warn(event.GetFullEventLabel() + "参数值范围设置错误: " + setting.Name)
			return nil
		}
		min := cast.ToFloat64(rangeVals[0])
		max := cast.ToFloat64(rangeVals[1])
		if paramFloat >= min && paramFloat <= max {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	case "eq_out":
		if len(rangeVals) < 2 {
			logx.Log().Warn(event.GetFullEventLabel() + "参数值范围设置错误: " + setting.Name)
			return nil
		}
		min := cast.ToFloat64(rangeVals[0])
		max := cast.ToFloat64(rangeVals[1])
		if paramFloat > min && paramFloat < max {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
	default:
		logx.Log().Warn(event.GetFullEventLabel() + "未知校验类型: " + setting.Name + " " + setting.Range)
	}
	return nil
}

func booleanParamValidate(
	setting *core.EventParam,
	param interface{},
	event *core.Event,
) *jsonx.JsonResponse {
	if param == true || param == false {
		return nil
	}
	boolStr := strings.ToLower(cast.ToString(param))
	if boolStr == "true" || boolStr == "false" {
		return nil
	}
	errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
	errJson.Message = "参数值不是有效的布尔值: " + setting.Name
	return errJson
}

func customParamValidate(
	setting *core.EventParam,
	param interface{},
	entityAttrs []core.EntityAttribute,
	event *core.Event,
	ctx types.WorkerContext,
) *jsonx.JsonResponse {
	attr := core.FindAttrFromArray(setting.Name, entityAttrs)
	if attr == nil {
		errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
		errJson.Message = "自定义字段的实体属性不存在: " + setting.Name
		return errJson
	}
	if parser, ok := ctx.Server().Repo().GetCustomFieldParser(attr.ValueSource); ok && parser != nil {
		if err := parser.Validate(param); err != nil {
			errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
			errJson.Message = "参数值不符合要求: " + setting.Name
			return errJson
		}
		return nil
	} else {
		logx.Log().Warn(event.GetFullEventLabel() + "未找到自定义字段的解析器: " + setting.Name)
		errJson := jsonx.DefaultJson(constant.INVALID_PARAM)
		errJson.Message = "未找到自定义字段的解析器: " + setting.Name
		return errJson
	}
}
