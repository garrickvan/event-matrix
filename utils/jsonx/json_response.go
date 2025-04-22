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

// Package jsonx 提供JSON处理相关的工具函数和结构体
package jsonx

import (
	"sync"
	"time"

	"github.com/garrickvan/event-matrix/constant"
)

// JsonResponse 定义了标准的JSON响应结构
// 用于在API接口中返回统一格式的响应数据
type JsonResponse struct {
	Code      string        `json:"code"`      // 响应码，表示操作结果状态
	CreatedAt int64         `json:"createdAt"` // 响应创建时间戳（毫秒）
	Message   string        `json:"message"`   // 响应消息，对状态的文字描述
	List      []interface{} `json:"list"`      // 响应数据列表
	Total     int64         `json:"total"`     // 数据总数（用于分页）
	Size      int           `json:"size"`      // 当前页数据大小
	Page      int           `json:"page"`      // 当前页码
}

// SetSizeInfo 设置分页相关信息
// 参数：
//   - total: 数据总数
//   - page: 当前页码
//   - size: 当前页数据量
func (r *JsonResponse) SetSizeInfo(total int64, page int, size int) {
	r.Page = page
	r.Size = size
	r.Total = total
}

// SetJsonList 将泛型列表数据设置到JsonResponse中
// 参数：
//   - resp: 要设置的JsonResponse对象
//   - list: 泛型数据列表
//   - total: 数据总数
//   - page: 当前页码
func SetJsonList[T any](resp *JsonResponse, list []T, total int64, page int) {
	// 直接将传入的切片元素逐个添加到r.List中
	for _, element := range list {
		resp.List = append(resp.List, element)
	}
	// 设置分页信息
	resp.SetSizeInfo(total, page, len(list))
}

// NewJsonResponseFromStr 从JSON字符串创建JsonResponse对象
// 参数：
//   - data: JSON格式的字符串
//
// 返回：
//   - *JsonResponse: 解析后的响应对象
//   - error: 解析过程中的错误，如果成功则为nil
func NewJsonResponseFromStr(data string) (*JsonResponse, error) {
	jsonResp := JsonResponse{}
	err := UnmarshalFromStr(data, &jsonResp)
	return &jsonResp, err
}

// NewJsonResponseFromBytes 从JSON字节数组创建JsonResponse对象
// 参数：
//   - data: JSON格式的字节数组
//
// 返回：
//   - *JsonResponse: 解析后的响应对象
//   - error: 解析过程中的错误，如果成功则为nil
func NewJsonResponseFromBytes(data []byte) (*JsonResponse, error) {
	jsonResp := JsonResponse{}
	err := UnmarshalFromBytes(data, &jsonResp)
	return &jsonResp, err
}

// GetStaticJsonResponseStr 根据响应码获取预生成的静态JSON响应字符串
// 使用预生成的静态响应可以提高性能，避免重复生成相同内容的响应
// 参数：
//   - code: 响应码
//
// 返回：对应响应码的JSON字符串，如果不存在则返回未处理错误的响应
func GetStaticJsonResponseStr(code constant.RESPONSE_CODE) string {
	if jsonData, exists := staticJsonResponses[code]; exists {
		return jsonData
	}
	return staticJsonResponses[constant.UNHANDLED_ERROR]
}

// DefaultJson 根据响应码动态生成标准JSON响应对象
// 参数：
//   - code: 响应码
//
// 返回：包含标准结构和默认值的JsonResponse对象
func DefaultJson(code constant.RESPONSE_CODE) *JsonResponse {
	message := constant.MsgForResponseCode(code)
	return &JsonResponse{
		Code:      string(code),
		CreatedAt: time.Now().UnixMilli(),
		Message:   message,
		List:      []interface{}{},
		Total:     0,
		Size:      0,
		Page:      0,
	}
}

// DefaultJsonWithMsg 根据响应码和自定义消息动态生成JSON响应对象
// 参数：
//   - code: 响应码
//   - message: 自定义响应消息，覆盖默认消息
//
// 返回：包含自定义消息的JsonResponse对象
func DefaultJsonWithMsg(code constant.RESPONSE_CODE, message string) *JsonResponse {
	return &JsonResponse{
		Code:      string(code),
		CreatedAt: time.Now().UnixMilli(),
		Message:   message,
		List:      []interface{}{},
		Total:     0,
		Size:      0,
		Page:      0,
	}
}

// generateDefaultResponse 生成标准格式的JSON响应字符串
// 用于初始化静态响应缓存
// 参数：
//   - code: 响应码
//
// 返回：序列化后的JSON字符串
func generateDefaultResponse(code constant.RESPONSE_CODE) string {
	defaultResponseStr, _ := MarshalToStr(&JsonResponse{
		Code:      string(code),
		CreatedAt: time.Now().UnixMilli(),
		Message:   constant.MsgForResponseCode(code),
		List:      []interface{}{},
		Total:     0,
		Size:      0,
		Page:      0,
	})
	return defaultResponseStr
}

var (
	staticJsonResponses map[constant.RESPONSE_CODE]string
	initOnce            sync.Once
)

// initStaticResponses 初始化静态响应映射表
func initStaticResponses() {
	staticJsonResponses = make(map[constant.RESPONSE_CODE]string)
	for _, code := range constant.AllResponseCodes() {
		staticJsonResponses[code] = generateDefaultResponse(code)
	}
}

func init() {
	initOnce.Do(initStaticResponses)
}
