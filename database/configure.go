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

// Package database 提供数据库配置和管理功能
package database

// DB_TYPE 定义支持的数据库类型
type DB_TYPE string

const (
	// SQLITE SQLite数据库
	SQLITE DB_TYPE = "sqlite"
	// PGSQL PostgreSQL数据库
	PGSQL DB_TYPE = "pgsql"
	// MYSQL MySQL数据库
	MYSQL DB_TYPE = "mysql"
)

// DBConf 定义数据库配置结构体
// 支持通过 YAML 或 JSON 格式进行配置
type DBConf struct {
	// Type 数据库类型，支持 sqlite、mysql、pgsql
	Type DB_TYPE `yaml:"type" json:"type"`

	// Location 数据库位置
	// 对于 SQLite：表示数据库文件路径
	// 对于 MySQL/PostgreSQL：表示 host:port 格式的服务器地址
	Location string `yaml:"location" json:"location"`

	// Port 数据库端口号
	// 注意：对于 SQLite 该字段无效
	Port int `yaml:"port" json:"port"`

	// LogSql 是否启用 SQL 日志记录
	// true: 记录所有执行的 SQL 语句
	// false: 不记录 SQL 语句
	LogSql bool `yaml:"log_sql" json:"log_sql"`

	// UserName 数据库用户名
	// 注意：对于 SQLite 该字段无效
	UserName string `yaml:"user_name" json:"user_name"`

	// Password 数据库密码
	// 注意：对于 SQLite 该字段无效
	Password string `yaml:"password" json:"password"`

	// DBName 数据库名称
	// 对于 SQLite：通常忽略此字段
	// 对于 MySQL/PostgreSQL：指定要连接的数据库名
	DBName string `yaml:"db_name" json:"db_name"`

	// MaxIdleConns 连接池中的最大空闲连接数
	// 设置过大可能会占用过多系统资源
	MaxIdleConns int `yaml:"max_idle_conns" json:"max_idle_conns"`

	// MaxOpenConns 数据库的最大打开连接数
	// 0 表示无限制；建议根据服务器配置和预期负载设置合适的值
	MaxOpenConns int `yaml:"max_open_conns" json:"max_open_conns"`

	// SkipDefaultTransaction 是否跳过默认事务
	// true: 跳过默认事务，可能提高性能但降低数据安全性
	// false: 使用默认事务，保证数据一致性
	SkipDefaultTransaction bool `yaml:"skip_default_transaction" json:"skip_default_transaction"`
}
