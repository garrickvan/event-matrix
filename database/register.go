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

// Package database 提供数据库连接、初始化和管理功能
package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/garrickvan/event-matrix/utils"
	"github.com/garrickvan/event-matrix/utils/logx"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 数据库连接字符串模板常量
const (
	// pgDSNTmpl PostgreSQL数据库连接字符串模板
	// 包含主机、用户名、密码、数据库名、端口等信息，并设置时区为Asia/Shanghai
	pgDSNTmpl = "host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai"

	// myDSNTmpl MySQL数据库连接字符串模板
	// 包含用户名、密码、主机、端口、数据库名等信息，并设置UTF8MB4字符集和本地时区
	myDSNTmpl = "%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local"
)

// getGormDB 根据数据库配置创建GORM数据库连接
//
// 参数:
//   - dbConf: 数据库配置信息，包含连接参数和性能设置
//
// 返回值:
//   - *gorm.DB: GORM数据库连接实例
//   - error: 连接过程中的错误，成功则为nil
func getGormDB(dbConf DBConf) (*gorm.DB, error) {
	// 配置GORM基本参数
	gormConfig := gorm.Config{
		SkipDefaultTransaction: dbConf.SkipDefaultTransaction,
		PrepareStmt:            true, // 启用预编译语句，提高性能
		DisableAutomaticPing:   true, // 自动ping数据库检查连接
	}

	// 根据配置启用SQL日志
	if dbConf.LogSql {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	// 根据数据库类型创建对应的连接
	t := dbConf.Type
	tstr := strings.ToLower(string(t))
	switch tstr {
	case string(SQLITE):
		// 确保SQLite数据库目录存在
		utils.MakeDir(dbConf.Location)
		dbPath := dbConf.Location + "/" + string(dbConf.DBName) + ".db"
		db, err := gorm.Open(sqlite.Open(dbPath), &gormConfig)
		if err != nil {
			return nil, fmt.Errorf("连接SQLite数据库失败，路径：%s，错误：%w", dbPath, err)
		}
		return db, nil

	case string(PGSQL), "postgres":
		// 确保PostgreSQL数据库存在
		err := makeSurePGDBExists(dbConf)
		if err != nil {
			return nil, err
		}
		// 构建连接字符串并创建连接
		dsn := fmt.Sprintf(pgDSNTmpl, dbConf.Location, dbConf.UserName, dbConf.Password, dbConf.DBName, dbConf.Port)
		db, err := gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: false, // 设置为true将禁用隐式预处理语句
		}), &gormConfig)
		if err != nil {
			return nil, err
		}
		// 配置连接池参数
		dbClient, err := db.DB()
		dbClient.SetMaxIdleConns(dbConf.MaxIdleConns) // 设置最大空闲连接数
		dbClient.SetMaxOpenConns(dbConf.MaxOpenConns) // 设置最大打开连接数
		dbClient.SetConnMaxLifetime(time.Hour)        // 设置连接最大存活时间
		dbClient.SetConnMaxIdleTime(30 * time.Minute) // 设置连接最大空闲时间
		return db, nil

	case string(MYSQL):
		// 确保MySQL数据库存在
		err := makeSureMySQLDBExists(dbConf)
		if err != nil {
			return nil, err
		}
		// 构建连接字符串并创建连接
		dsn := fmt.Sprintf(myDSNTmpl, dbConf.UserName, dbConf.Password, dbConf.Location, dbConf.Port, dbConf.DBName)
		db, err := gorm.Open(mysql.Open(dsn), &gormConfig)
		if err != nil {
			return nil, err
		}
		// 配置连接池参数
		dbClient, err := db.DB()
		dbClient.SetMaxIdleConns(dbConf.MaxIdleConns)
		dbClient.SetMaxOpenConns(dbConf.MaxOpenConns)
		return db, nil

	default:
		return nil, fmt.Errorf("不支持的数据库类型，目前仅支持sqlite、pgsql和mysql")
	}
}

// makeSurePGDBExists 确保PostgreSQL数据库存在，如不存在则创建
//
// 参数:
//   - dbConf: 数据库配置信息
//
// 返回值:
//   - error: 操作过程中的错误，成功则为nil
func makeSurePGDBExists(dbConf DBConf) error {
	// 连接到postgres默认数据库
	dsn := fmt.Sprintf(pgDSNTmpl, dbConf.Location, dbConf.UserName, dbConf.Password, "postgres", dbConf.Port)
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: false, // 设置为true将禁用隐式预处理语句
	}), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("根数据库postgres连接失败：%w", err)
	}

	// 检查目标数据库是否存在
	var count int64
	result := db.Raw("SELECT COUNT(*) FROM pg_database WHERE datname=?", dbConf.DBName).Scan(&count)
	if result.Error != nil {
		return fmt.Errorf("查询数据库失败：%w", result.Error)
	}

	// 如果数据库不存在，则创建
	if count == 0 {
		if err := db.Exec("CREATE DATABASE " + dbConf.DBName).Error; err != nil {
			return fmt.Errorf("创建数据库失败：%w", err)
		}
	}
	return nil
}

// makeSureMySQLDBExists 确保MySQL数据库存在，如不存在则创建
//
// 参数:
//   - dbConf: 数据库配置信息
//
// 返回值:
//   - error: 操作过程中的错误，成功则为nil
func makeSureMySQLDBExists(dbConf DBConf) error {
	// 连接到MySQL服务器（不指定数据库）
	dsn := fmt.Sprintf(myDSNTmpl, dbConf.UserName, dbConf.Password, dbConf.Location, dbConf.Port, "")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("根数据库MySQL连接失败：%w", err)
	}

	// 检查目标数据库是否存在
	var count int64
	result := db.Raw("SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?", dbConf.DBName).Scan(&count)
	if result.Error != nil {
		return fmt.Errorf("查询数据库失败：%w", result.Error)
	}

	// 如果数据库不存在，则创建
	if count == 0 {
		if err := db.Exec("CREATE DATABASE " + dbConf.DBName).Error; err != nil {
			return fmt.Errorf("创建数据库失败：%w", err)
		}
	}
	return nil
}

// getGormDBWithRetry 带重试机制的数据库连接创建函数
//
// 参数:
//   - dbConf: 数据库配置信息
//   - retry: 最大重试次数
//
// 返回值:
//   - *gorm.DB: GORM数据库连接实例
//   - error: 连接过程中的错误，成功则为nil
func getGormDBWithRetry(dbConf DBConf, retry int) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	// 循环尝试连接，每次失败后等待递增的时间
	for i := 0; i < retry; i++ {
		db, err = getGormDB(dbConf)
		if err == nil {
			return db, nil
		} else {
			logx.Errorf("数据库连接失败，第%d次重试，错误：%v", i+1, err)
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return nil, fmt.Errorf("数据库连接失败，重试%d次后仍然失败：%v", retry, err)
}
