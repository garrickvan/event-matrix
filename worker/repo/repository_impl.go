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
	"errors"
	"reflect"
	"strings"

	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/database"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/spf13/cast"
)

type RepositoryImpl struct {
	database.DBManager
	customFields map[string]types.CustomFieldParser
	ws           types.WorkerServer
}

func NewRepository(ws types.WorkerServer) *RepositoryImpl {
	dbm := database.NewGormDBManager()
	return &RepositoryImpl{
		DBManager:    dbm,
		ws:           ws,
		customFields: map[string]types.CustomFieldParser{},
	}
}

func (rp *RepositoryImpl) RegisterCustomFieldParser(customField types.CustomFieldParser) {
	name := customField.FieldParserName()
	if name == "" {
		panic("自定义字段解析器必须设置字段类型")
	}
	if _, ok := rp.customFields[name]; ok {
		logx.Log().Error("自定义字段解析器已存在: " + name)
		return
	}
	rp.customFields[name] = customField
}

func (rp *RepositoryImpl) GetCustomFieldParser(typz string) (types.CustomFieldParser, bool) {
	customField, ok := rp.customFields[typz]
	return customField, ok
}

func (rp *RepositoryImpl) AddDBFromSharedConfig(sid string) error {
	if sid == "" {
		return errors.New("数据库配置不能为空")
	}
	config := rp.ws.SharedConfigure(sid)
	if config != nil {
		if core.IsDBConfigType(config.Type) {
			return rp.addDBFromSharedConfig(*config)
		} else {
			logx.Debug("非数据库配置 " + config.Type)
			return nil
		}
	} else {
		return errors.New("配置中心不存在数据库配置: " + sid)
	}
}

func (rp *RepositoryImpl) addDBFromSharedConfig(config core.SharedConfigure) error {
	dbConf := database.DBConf{}
	if err := jsonx.UnmarshalFromStr(config.Value, &dbConf); err != nil {
		logx.Debug("解析数据库配置失败: " + err.Error())
		return err
	}
	if dbConf.DBName == "" || dbConf.Type == "" || dbConf.Location == "" {
		return errors.New("数据库配置不完整")
	}
	if rp.HasDB(dbConf.DBName) {
		return nil
	}
	error := rp.RegisterDB(&dbConf)
	if error != nil {
		return error
	}
	return nil
}

func (rp *RepositoryImpl) SyncSchema(w *types.Worker) error {
	dbName := w.Project
	if rp.HasDB(dbName) {
		entityAttrs := rp.ws.DomainCache().EntityAttrs(types.PathToEntityFromWorker(w))
		if len(entityAttrs) < 1 {
			return errors.New("没有找到实体属性: " + dbName + "." + w.Context + "." + w.Entity + "@" + w.VersionLabel)
		}
		rp.autoMigrateTable(w, entityAttrs)
	} else if dbName == core.INTERNAL_PROJECT {
		logx.Debug("跳过内部项目: " + dbName)
	} else {
		return errors.New("目标数据库: " + dbName + " 没有配置，请检查配置文件")
	}
	return nil
}

// 自动迁移表结构
// 1. 遍历实体属性，构建表结构体
// 2. 利用反射机制构建表结构体
// 3. 调用gorm的AutoMigrate方法
// 注意：字段只会迁移一次，后续不会再迁移，更改自定义字段需要重新使用其他的字段名，这样能保证版本数据的兼容性。
func (rp *RepositoryImpl) autoMigrateTable(w *types.Worker, entityAttrs []core.EntityAttribute) {
	if entityAttrs == nil || len(entityAttrs) < 1 {
		return
	}
	tableName := w.GetTabelName()
	db := rp.Use(w.Project)
	if db == nil {
		logx.Log().Error("数据库连接失败: " + w.Project)
		return
	}
	if strings.Index(db.Dialector.Name(), "sqlite") >= 0 {
		rp.autoMigrateTableSqlite(db, tableName, entityAttrs)
		return
	}
	table := db.Table(tableName)
	// 利用反射机制构建表的结构体
	tableStructs := []reflect.StructField{}
	for i, v := range entityAttrs {
		// 跳过已存在数据库中的自定义字段
		exist := database.CheckFieldExists(table, tableName, v.Code)
		if exist {
			logx.Info("字段:【" + v.Code + "】已存在于数据库，跳过自动创建")
			continue
		}
		structField := rp.getStrutFieldForAutoMigrate(v)
		if structField == nil {
			continue
		}
		structField.Name = "F_" + cast.ToString(i) // 临时命名，避免冲突，tag会覆盖
		tableStructs = append(tableStructs, *structField)
	}
	class := reflect.StructOf(tableStructs)
	tableClass := reflect.New(class).Interface()
	err := table.AutoMigrate(tableClass)
	if err != nil {
		logx.Log().Error("自动迁移表错误: " + err.Error())
	} else {
		// 自定义字段处理
		for _, v := range entityAttrs {
			if v.FieldType == "custom" {
				rp.handleCustomField(table, tableName, v)
			}
		}
		logx.Debug("自动迁移表成功: " + w.Project + "." + tableName)
	}
}

/*
**

	string: "字符",
	constant: "常量字典",
	text: "文本",
	ref: "实体引用",
	int8: "短整数",
	int32: "整数",
	int64: "长整数",
	float32: "浮点数",
	float64: "长浮点数",
	boolean: "布尔值",
	datetime: "日期时间",
	id: "ID",
	custom: "自定义",
	uid: "用户ID",
	url: "URL",
	email: "邮箱",
	phone: "手机号"

**
*/
func (rp *RepositoryImpl) getStrutFieldForAutoMigrate(attr core.EntityAttribute) *reflect.StructField {
	f := reflect.StructField{}
	switch attr.FieldType {
	case "id":
		f.Type = reflect.TypeOf("")
		f.Tag = reflect.StructTag("gorm:\"column:id;primary_key;varchar(36)\"")
		return &f
	case "string", "constant", "text", "ref", "uid", "url", "email", "phone":
		f.Type = reflect.TypeOf("")
	case "int8":
		f.Type = reflect.TypeOf(int8(0))
	case "int32":
		f.Type = reflect.TypeOf(int32(0))
	case "int64":
		f.Type = reflect.TypeOf(int64(0))
	case "float32":
		f.Type = reflect.TypeOf(float32(0))
	case "float64":
		f.Type = reflect.TypeOf(float64(0))
	case "boolean":
		f.Type = reflect.TypeOf(false)
	case "datetime":
		f.Type = reflect.TypeOf(int64(0))
		// 软删除字段
		if attr.Code == "deleted_at" {
			f.Tag = reflect.StructTag("gorm:\"column:deleted_at;index;default:0\"")
			return &f
		}
	case "custom":
		// 最后一步处理自定义字段
		return nil
	default:
		logx.Debug("未知类型的字段: " + attr.FieldType)
		return nil
	}

	getTag := func(entityAttr *core.EntityAttribute) string {
		tagParts := []string{"column:" + entityAttr.Code}
		if entityAttr.Unique {
			tagParts = append(tagParts, "unique")
		}
		if entityAttr.Indexed {
			tagParts = append(tagParts, "index")
		}
		return "gorm:\"" + strings.Join(tagParts, ";") + "\""
	}

	f.Tag = reflect.StructTag(getTag(&attr))
	return &f
}
