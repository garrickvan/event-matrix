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

package repo

import (
	"fmt"
	"strings"

	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/database"
	"github.com/garrickvan/event-matrix/utils/logx"
	"gorm.io/gorm"
)

// 常量定义提升可维护性
const (
	indexPrefix     = "idx_"
	deletedAtColumn = "deleted_at"
	idColumn        = "id"
)

func (rp *RepositoryImpl) autoMigrateTableSqlite(db *gorm.DB, tableName string, entityAttrs []core.EntityAttribute) {
	if len(entityAttrs) == 0 {
		logx.Debug("自动迁移: 空字段列表，跳过处理")
		return
	}

	// 使用事务保证表创建的原子性
	err := db.Transaction(func(tx *gorm.DB) error {
		table := tx.Table(tableName)

		// 表不存在时执行创建操作
		if !tx.Migrator().HasTable(tableName) {
			if err := rp.createSqliteTable(tx, tableName); err != nil {
				return err
			}
		}

		// 字段迁移处理
		for _, attr := range entityAttrs {
			if database.CheckFieldExists(table, tableName, attr.Code) {
				logx.Infof("字段已存在: %s.%s", tableName, attr.Code)
				continue
			}

			if err := rp.createSqliteColumn(tx, tableName, attr); err != nil {
				logx.Errorf("字段迁移失败: %s.%s - %v", tableName, attr.Code, err)
				// 根据业务需求决定是否终止迁移流程
				// return err 如果要保证全成功，此处返回错误
			}
		}
		return nil
	})

	if err != nil {
		logx.Errorf("自动迁移事务失败: %s - %v", tableName, err)
	}
}

func (rp *RepositoryImpl) createSqliteTable(tx *gorm.DB, tableName string) error {
	createStmt := strings.Join([]string{
		"CREATE TABLE " + tableName + " (",
		idColumn + " TEXT PRIMARY KEY NOT NULL,",
		deletedAtColumn + " INTEGER DEFAULT 0",
		");",
	}, "\n")

	if err := tx.Exec(createStmt).Error; err != nil {
		return err
	}
	// 添加deleted_at索引
	createIndexSQL := fmt.Sprintf(`
CREATE INDEX IF NOT EXISTS idx_%s_deleted_at ON %s (deleted_at)`,
		tableName, tableName)

	if err := tx.Exec(createIndexSQL).Error; err != nil {
		return fmt.Errorf("failed to create deleted_at index: %v", err)
	}
	logx.Info("表创建成功: " + tableName)
	return nil
}

func (rp *RepositoryImpl) createSqliteColumn(tx *gorm.DB, tableName string, attr core.EntityAttribute) error {
	// 处理自定义字段类型
	if attr.FieldType == "custom" {
		return rp.handleCustomField(tx, tableName, attr)
	}

	// 标准字段处理
	columnDef, err := rp.buildColumnDefinition(attr)
	if err != nil {
		return err
	}

	// 执行字段添加
	alterStmt := "ALTER TABLE " + tableName + " ADD COLUMN " + columnDef
	if err := tx.Exec(alterStmt).Error; err != nil {
		return err
	}

	// 索引处理
	if attr.Indexed {
		if err := rp.createIndex(tx, tableName, attr.Code); err != nil {
			return err
		}
	}

	logx.Debugf("字段创建成功: %s.%s", tableName, attr.Code)
	return nil
}

func (rp *RepositoryImpl) buildColumnDefinition(attr core.EntityAttribute) (string, error) {
	columnType, err := mapFieldType(attr.FieldType)
	if err != nil {
		return "", err
	}

	constraints := []string{}
	if attr.Unique {
		constraints = append(constraints, "UNIQUE")
	}
	if attr.FieldType == "id" && attr.Code == idColumn {
		constraints = append(constraints, "NOT NULL")
	}

	return attr.Code + " " + columnType + " " + strings.Join(constraints, " "), nil
}

func mapFieldType(fieldType string) (string, error) {
	switch fieldType {
	case "id":
		return "TEXT PRIMARY KEY", nil
	case "string", "constant", "text", "ref", "uid", "url", "email", "phone":
		return "TEXT", nil
	case "int8", "int32", "int64":
		return "INTEGER", nil
	case "float32", "float64":
		return "REAL", nil
	case "boolean":
		return "INTEGER", nil // SQLite 布尔兼容处理
	case "datetime":
		return "INTEGER", nil
	default:
		return "", fmt.Errorf("未知字段类型: %s", fieldType)
	}
}

func (rp *RepositoryImpl) handleCustomField(tx *gorm.DB, tableName string, attr core.EntityAttribute) error {
	key := strings.TrimSpace(attr.ValueSource)
	customField, ok := rp.customFields[key]
	if !ok {
		return fmt.Errorf("未注册的自定义字段解析器: %s (字段: %s)", key, attr.Code)
	}

	if err := customField.CreateColumn(tx, tableName, attr.Code); err != nil {
		return err
	}
	return nil
}

func (rp *RepositoryImpl) createIndex(tx *gorm.DB, tableName, columnName string) error {
	indexStmt := "CREATE INDEX IF NOT EXISTS " + indexName(tableName, columnName) +
		" ON " + tableName + "(" + columnName + ")"
	return tx.Exec(indexStmt).Error
}

func indexName(tableName, columnName string) string {
	return indexPrefix + tableName + "_" + columnName
}
