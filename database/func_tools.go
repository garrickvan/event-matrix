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

// Package database 提供数据库配置、管理和操作功能
package database

import (
	"database/sql"

	"github.com/garrickvan/event-matrix/utils/logx"
	"gorm.io/gorm"
)

// CheckFieldExists 检查指定表中是否存在特定字段
// 目前仅支持sqlite、pgsql、mysql和sqlserver数据库
//
// 参数:
//   - db: GORM数据库连接实例
//   - tableName: 要检查的表名
//   - fieldName: 要检查的字段名
//
// 返回值:
//   - bool: 如果字段存在返回true，否则返回false
func CheckFieldExists(db *gorm.DB, tableName string, fieldName string) bool {
	var query string
	var exists bool

	// 根据不同数据库类型构建对应的SQL查询语句
	switch db.Dialector.Name() {
	case "mysql":
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = DATABASE()
				  AND table_name = ?
				  AND column_name = ?
			)
		`
	case "postgres":
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = ?
				  AND column_name = ?
			)
		`
	case "sqlite":
		query = `
			SELECT 1
			FROM pragma_table_info(?)
			WHERE name = ?
		`
	case "sqlserver":
		query = `
			SELECT CASE WHEN EXISTS (
				SELECT 1
				FROM INFORMATION_SCHEMA.COLUMNS
				WHERE TABLE_NAME = ?
				  AND COLUMN_NAME = ?
			) THEN 1 ELSE 0 END
		`
	default:
		// 不支持的数据库类型记录警告日志并返回false
		logx.Log().Warn("不支持的数据库类型，目前仅支持sqlite、pgsql和mysql")
		return false
	}

	// 执行查询并获取结果
	err := db.Raw(query, tableName, fieldName).Scan(&exists).Error
	if err != nil {
		logx.Log().Error("查询字段失败：" + err.Error())
		return false
	}
	return exists
}

// RawSqlExec 执行非查询SQL语句（如INSERT、UPDATE、DELETE等）
//
// 参数:
//   - db: GORM数据库连接实例
//   - sqlStatement: 要执行的SQL语句，可包含命名参数
//   - params: 命名参数的值映射
//
// 返回值:
//   - int64: 受影响的行数
//   - error: 执行过程中的错误，成功则为nil
func RawSqlExec(db *gorm.DB, sqlStatement string, params map[string]interface{}) (int64, error) {
	// 将参数映射转换为sql.Named参数列表
	args := make([]interface{}, 0, len(params))
	for k, v := range params {
		args = append(args, sql.Named(k, v))
	}

	// 执行SQL语句
	tx := db.Exec(sqlStatement, args...)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}

// RawQuerySqlExec 执行查询SQL语句并返回结果集
//
// 参数:
//   - db: GORM数据库连接实例
//   - sqlStatement: 要执行的查询SQL语句
//   - params: 查询参数的值映射
//
// 返回值:
//   - []map[string]interface{}: 查询结果集，每行数据表示为字段名到值的映射
//   - error: 执行过程中的错误，成功则为nil
func RawQuerySqlExec(db *gorm.DB, sqlStatement string, params map[string]interface{}) ([]map[string]interface{}, error) {
	// 执行SQL查询并获取结果行
	rows, err := db.Raw(sqlStatement, params).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}

	// 获取结果集的列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 准备用于扫描的值存储
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// 逐行扫描结果
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// 将当前行数据转换为map
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// 将字节数组转换为字符串
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			rowMap[col] = val
		}
		results = append(results, rowMap)
	}
	return results, nil
}

// TransactionRawSqlExec 在事务中执行非查询SQL语句
// 提供自动提交或回滚事务的功能
//
// 参数:
//   - db: GORM数据库连接实例
//   - sqlStatement: 要执行的SQL语句，可包含命名参数
//   - params: 命名参数的值映射
//
// 返回值:
//   - int64: 受影响的行数
//   - error: 执行过程中的错误，成功则为nil
func TransactionRawSqlExec(db *gorm.DB, sqlStatement string, params map[string]interface{}) (int64, error) {
	var rowsAffected int64

	// 在事务中执行SQL语句
	err := db.Transaction(func(tx *gorm.DB) error {
		i, err := RawSqlExec(tx, sqlStatement, params)
		if err != nil {
			return err // 返回错误会导致事务回滚
		}
		rowsAffected = i
		return nil // 返回nil会导致事务提交
	})
	return rowsAffected, err
}

// TransactionRawQuerySqlExec 在事务中执行查询SQL语句并返回结果集
// 提供自动提交或回滚事务的功能
//
// 参数:
//   - db: GORM数据库连接实例
//   - sqlStatement: 要执行的查询SQL语句
//   - params: 查询参数的值映射
//
// 返回值:
//   - []map[string]interface{}: 查询结果集，每行数据表示为字段名到值的映射
//   - error: 执行过程中的错误，成功则为nil
func TransactionRawQuerySqlExec(db *gorm.DB, sqlStatement string, params map[string]interface{}) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// 在事务中执行查询
	err := db.Transaction(func(tx *gorm.DB) error {
		r, err := RawQuerySqlExec(tx, sqlStatement, params)
		if err != nil {
			return err // 返回错误会导致事务回滚
		}
		results = r
		return nil // 返回nil会导致事务提交
	})
	return results, err
}
