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

package constant

// RESPONSE_CODE 定义API响应的状态码类型
type RESPONSE_CODE string

// 通用响应码
const (
	SUCCESS         RESPONSE_CODE = "ok"
	WORKER_ENDPOINT RESPONSE_CODE = "worker_endpoint"
)

// 操作失败响应码
const (
	FAIL_TO_CREATE  RESPONSE_CODE = "fail_to_create"
	FAIL_TO_QUERY   RESPONSE_CODE = "fail_to_query"
	FAIL_TO_UPDATE  RESPONSE_CODE = "fail_to_update"
	FAIL_TO_DELETE  RESPONSE_CODE = "fail_to_delete"
	FAIL_TO_PROCESS RESPONSE_CODE = "fail_to_process"
)

// 验证失败响应码
const (
	INVALID_SIGN     RESPONSE_CODE = "invalid_sign" // 这里指事件的签名错误
	INVALID_PARAM    RESPONSE_CODE = "invalid_param"
	INVALID_TOKEN    RESPONSE_CODE = "invalid_token"
	INVALID_PASSWORD RESPONSE_CODE = "invalid_password"
	MISSING_PARAM    RESPONSE_CODE = "missing_param"
	ALREADY_EXIST    RESPONSE_CODE = "already_exist"
)

// 用户相关响应码
const (
	USER_EXIST        RESPONSE_CODE = "user_exist"
	USER_BANNED       RESPONSE_CODE = "user_banned"
	USER_NOT_EXIST    RESPONSE_CODE = "user_not_exist"
	USER_SIGN_TIMEOUT RESPONSE_CODE = "user_sign_timeout"
	USER_UNAUTHORIZED RESPONSE_CODE = "user_unauthorized"
)

// 事件相关响应码
const (
	EVENT_TIMEOUT    RESPONSE_CODE = "event_timeout"
	EVENT_NOT_EXIST  RESPONSE_CODE = "event_not_exist"
	ENTITY_NOT_EXIST RESPONSE_CODE = "entity_not_exist"
)

// 不支持相关响应码
const (
	UNSUPPORTED_EVENT  RESPONSE_CODE = "unsupported_event"
	UNSUPPORTED_CLIENT RESPONSE_CODE = "unsupported_client"
)

// 错误相关响应码
const (
	UNHANDLED_ERROR   RESPONSE_CODE = "unhandled_error"
	UNKNOWN_DATA      RESPONSE_CODE = "unknown_data"
	EMPTY_DATA        RESPONSE_CODE = "empty_data"
	DATA_EXIST        RESPONSE_CODE = "data_exist"
	TOO_MANY_REQUESTS RESPONSE_CODE = "TOO_MANY_REQUESTS"
	LIMIT_REACHED     RESPONSE_CODE = "limit_reached"  // 最大上限
	FORBIDDEN_CALL    RESPONSE_CODE = "forbidden_call" // 不允许的调用方式
)

// 任务相关响应码
const (
	TASK_PENDING       RESPONSE_CODE = "task_pending"
	TASK_IN_PROGRESS   RESPONSE_CODE = "task_in_progress"
	TASK_FAILED        RESPONSE_CODE = "task_failed"
	TASK_TIMEOUT       RESPONSE_CODE = "task_timeout"
	TASK_UNKNOWN       RESPONSE_CODE = "task_unknown"
	TASK_LIMIT_REACHED RESPONSE_CODE = "task_limit_reached" // 任务最大上限
)

// 服务相关响应码
const (
	SERVICE_UNAVAILABLE RESPONSE_CODE = "service_unavailable" // 服务不可用
	REQUEST_TIMEOUT     RESPONSE_CODE = "request_timeout"     // 请求超时
	CONFLICT            RESPONSE_CODE = "conflict"            // 资源冲突
	NOT_IMPLEMENTED     RESPONSE_CODE = "not_implemented"     // 功能未实现
)

// 响应码消息映射
var responseCodeMessages = map[RESPONSE_CODE]string{
	SUCCESS:             "成功",
	WORKER_ENDPOINT:     "worker地址",
	FAIL_TO_CREATE:      "创建失败",
	FAIL_TO_QUERY:       "查询失败",
	FAIL_TO_UPDATE:      "更新失败",
	FAIL_TO_DELETE:      "删除失败",
	FAIL_TO_PROCESS:     "处理失败",
	INVALID_SIGN:        "无效的签名",
	INVALID_PARAM:       "无效的参数",
	INVALID_TOKEN:       "登录信息无效",
	INVALID_PASSWORD:    "账号信息错误",
	USER_EXIST:          "用户已存在",
	USER_BANNED:         "用户已被禁止",
	USER_NOT_EXIST:      "无效的账号或密码",
	USER_SIGN_TIMEOUT:   "用户签名超时",
	USER_UNAUTHORIZED:   "用户未授权",
	EVENT_TIMEOUT:       "事件超时",
	EVENT_NOT_EXIST:     "事件不存在",
	ENTITY_NOT_EXIST:    "实体不存在",
	UNSUPPORTED_CLIENT:  "不支持的客户端",
	UNSUPPORTED_EVENT:   "不支持的事件",
	UNHANDLED_ERROR:     "未处理的错误",
	UNKNOWN_DATA:        "未知数据格式",
	LIMIT_REACHED:       "达到最大上限",
	DATA_EXIST:          "数据已存在",
	FORBIDDEN_CALL:      "禁止的调用方式",
	MISSING_PARAM:       "缺少必要参数",
	ALREADY_EXIST:       "数据已存在",
	TOO_MANY_REQUESTS:   "请求过于频繁",
	TASK_PENDING:        "任务已添加",
	TASK_IN_PROGRESS:    "任务进行中",
	TASK_FAILED:         "任务失败",
	TASK_TIMEOUT:        "任务超时",
	TASK_UNKNOWN:        "任务未知",
	TASK_LIMIT_REACHED:  "任务达到最大上限",
	SERVICE_UNAVAILABLE: "服务不可用",
	REQUEST_TIMEOUT:     "请求超时",
	CONFLICT:            "资源冲突",
	NOT_IMPLEMENTED:     "功能未实现",
	EMPTY_DATA:          "数据为空",
}

// MsgForResponseCode 根据响应码获取对应的默认消息文本
// 提供了所有支持的响应码的标准消息文本
// 参数：
//   - code: 响应码
//
// 返回：对应的消息文本，如果未定义则返回"未处理的错误"
func MsgForResponseCode(code RESPONSE_CODE) string {
	if msg, ok := responseCodeMessages[code]; ok {
		return msg
	}
	return "未处理的错误"
}

// AllResponseCodes 返回所有预定义的响应码
// 用于集中管理所有响应码定义
func AllResponseCodes() []RESPONSE_CODE {
	codes := make([]RESPONSE_CODE, 0, len(responseCodeMessages))
	for code := range responseCodeMessages {
		codes = append(codes, code)
	}
	return codes
}
