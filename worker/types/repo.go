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

package types

import (
	"github.com/garrickvan/event-matrix/database"
	"gorm.io/gorm"
)

// Repository 定义了仓库管理的核心接口，包含数据库管理和自定义字段解析等功能。
type Repository interface {
	// database.DBManager 是一个嵌入式接口，提供了数据库管理的基本功能。
	database.DBManager

	// AddDBFromSharedConfig 根据共享配置添加数据库实例。
	// 参数 sid 是共享配置的唯一标识符。
	// 返回错误信息，如果操作失败。
	AddDBFromSharedConfig(sid string) error

	// SyncSchema 同步 Worker 的模式（schema）到数据库。
	// 参数 w 是需要同步的 Worker 实例。
	// 返回错误信息，如果同步过程中出现问题。
	SyncSchema(w *Worker) error

	// RegisterCustomFieldParser 注册一个自定义字段解析器。
	// 参数 parser 是要注册的自定义字段解析器。
	RegisterCustomFieldParser(parser CustomFieldParser)

	// GetCustomFieldParser 获取指定类型的自定义字段解析器。
	// 参数 fieldType 是自定义字段的类型。
	// 返回值 cf 是匹配的自定义字段解析器，ok 表示是否找到对应的解析器。
	GetCustomFieldParser(fieldType string) (cf CustomFieldParser, ok bool)
}

/*
*
* 自定义字段解析器
*
  - 自定义字段解析器用于解析前端传来的参数，并将其转换为数据库字段的类型，前端统一使用string作为参数载体，然后由ParseParam方法进行解析。
  - 自定义类型在内置CURD方法中处理可能会出问题，如果遇到异常，请使用自定义事件进行处理。
  - 自定义字段解析器不支持默认的唯一性约束，如果需要唯一性约束，请自行实现。
  - 系统假定是自定义字段跟数据库特性绑定，所以很多系统的内置能力会失效

*
*/
type CustomFieldParser interface {
	FieldParserName() string                                      // 必填, 字段解析器名称
	Validate(interface{}) error                                   // 必填, 校验参数是否有效
	ParseParam(string) interface{}                                // 必填, 解析前端传来的参数
	CreateColumn(db *gorm.DB, tableName, columnName string) error // 必选, 创建字段
	DefaultValue() interface{}                                    // 可选, 默认值，不设置则使用nil
	Description() string                                          // 可选, 字段描述
}
