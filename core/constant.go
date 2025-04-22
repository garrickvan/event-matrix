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

// Package core 实现了事件矩阵系统的核心数据模型和业务逻辑
package core

import (
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/spf13/cast"
)

// ConstantDictType 定义常量字典的数据类型
type ConstantDictType string

const (
	// IntConstantDict 整数类型的常量字典
	IntConstantDict ConstantDictType = "int"
	// StrConstantDict 字符串类型的常量字典
	StrConstantDict ConstantDictType = "string"
)

// ConstantDict 表示系统中的常量字典，用于存储和管理各种配置项和枚举值
type ConstantDict struct {
	ID        string           `json:"id" gorm:"primaryKey"`
	Value     string           `json:"value" gorm:"index"`
	Label     string           `json:"label"`
	Dict      string           `json:"dict" gorm:"index"`
	DictName  string           `json:"dictName"`
	Project   string           `json:"project" gorm:"index"`
	Type      ConstantDictType `json:"type"`
	CreatedAt int64            `json:"createdAt"`
	UpdatedAt int64            `json:"updatedAt"`
	DeletedAt int64            `json:"deletedAt" gorm:"index"`
	DeletedBy string           `json:"deletedBy"`
}

const (
	// WORKER_CONFIG 工作节点配置类型
	WORKER_CONFIG = "worker_config"

	// 数据库类型常量
	// DB_POSTGRES PostgreSQL数据库
	DB_POSTGRES = "postgres"
	// DB_MYSQL MySQL数据库
	DB_MYSQL = "mysql"
	// DB_SQLITE SQLite数据库
	DB_SQLITE = "sqlite"
	// DB_SQSERVER SQL Server数据库
	DB_SQSERVER = "sqlserver"
	// DB_ORACLE Oracle数据库
	DB_ORACLE = "oracle"
	// DB_OTHER 其他类型数据库
	DB_OTHER = "other_db"

	// 缓存类型常量
	// CACHE_REDIS Redis缓存
	CACHE_REDIS = "redis"
	// CACHE_MEMCACHED Memcached缓存
	CACHE_MEMCACHED = "memcached"
	// CACHE_OTHER 其他类型缓存
	CACHE_OTHER = "other_cache"

	// 消息队列类型常量
	// MQ_KAFKA Kafka消息队列
	MQ_KAFKA = "kafka"
	// MQ_ROCKETMQ RocketMQ消息队列
	MQ_ROCKETMQ = "rocketmq"
	// MQ_RABBITMQ RabbitMQ消息队列
	MQ_RABBITMQ = "rabbitmq"
	// MQ_NATS NATS消息队列
	MQ_NATS = "nats"
	// MQ_PULSAR Pulsar消息队列
	MQ_PULSAR = "pulsar"
	// MQ_OTHER 其他类型消息队列
	MQ_OTHER = "other_mq"

	// AI_MODEL AI模型配置类型
	AI_MODEL = "ai_model"

	// CUSTOM 自定义配置类型
	CUSTOM = "custom"
)

// IsWorkerConfigType 判断给定的类型是否为工作节点配置类型
func IsWorkerConfigType(workerType string) bool {
	switch workerType {
	case WORKER_CONFIG:
		return true
	default:
		return false
	}
}

// IsCustomConfigType 判断给定的类型是否为自定义配置类型
func IsCustomConfigType(customType string) bool {
	switch customType {
	case CUSTOM:
		return true
	default:
		return false
	}
}

// IsDBConfigType 判断给定的类型是否为数据库配置类型
func IsDBConfigType(dbType string) bool {
	switch dbType {
	case DB_POSTGRES, DB_MYSQL, DB_SQLITE, DB_SQSERVER, DB_ORACLE, DB_OTHER:
		return true
	default:
		return false
	}
}

// IsCacheConfigType 判断给定的类型是否为缓存配置类型
func IsCacheConfigType(cacheType string) bool {
	switch cacheType {
	case CACHE_REDIS, CACHE_MEMCACHED, CACHE_OTHER:
		return true
	default:
		return false
	}
}

// IsMQConfigType 判断给定的类型是否为消息队列配置类型
func IsMQConfigType(mqType string) bool {
	switch mqType {
	case MQ_KAFKA, MQ_ROCKETMQ, MQ_RABBITMQ, MQ_NATS, MQ_PULSAR, MQ_OTHER:
		return true
	default:
		return false
	}
}

// SharedConfigure 表示系统中的共享配置，用于存储和管理各个工作节点共用的配置信息
type SharedConfigure struct {
	Key         string `json:"key" gorm:"primaryKey"`
	Type        string `json:"type" gorm:"index"`
	Value       string `json:"value"`       // json字符串
	UsedWorkers string `json:"usedWorkers"` // 记录多少个worker采用了该配置，不保证准确性
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
	DeletedAt   int64  `json:"deletedAt" gorm:"index"`
	DeletedBy   string `json:"deletedBy"`
}

// NewSharedCfgFromMap 从map类型数据创建SharedConfigure实例
func NewSharedCfgFromMap(v interface{}) *SharedConfigure {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &SharedConfigure{}
	}
	e := SharedConfigure{
		Key:         cast.ToString(data["key"]),
		Type:        cast.ToString(data["type"]),
		Value:       cast.ToString(data["value"]),
		UsedWorkers: cast.ToString(data["usedWorkers"]),
		CreatedAt:   cast.ToInt64(data["createdAt"]),
		UpdatedAt:   cast.ToInt64(data["updatedAt"]),
		DeletedAt:   cast.ToInt64(data["deletedAt"]),
		DeletedBy:   cast.ToString(data["deletedBy"]),
	}
	return &e
}

// NewConstantDictFromMap 从map类型数据创建ConstantDict实例
func NewConstantDictFromMap(v interface{}) *ConstantDict {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &ConstantDict{}
	}

	return &ConstantDict{
		ID:        cast.ToString(data["id"]),
		Value:     cast.ToString(data["value"]),
		Label:     cast.ToString(data["label"]),
		Dict:      cast.ToString(data["dict"]),
		DictName:  cast.ToString(data["dictName"]),
		Project:   cast.ToString(data["project"]),
		Type:      ConstantDictType(cast.ToString(data["type"])),
		CreatedAt: cast.ToInt64(data["createdAt"]),
		UpdatedAt: cast.ToInt64(data["updatedAt"]),
		DeletedAt: cast.ToInt64(data["deletedAt"]),
		DeletedBy: cast.ToString(data["deletedBy"]),
	}
}

// NewSharedCfgFromJson 从JSON字符串创建SharedConfigure实例
func NewSharedCfgFromJson(v string) *SharedConfigure {
	var data SharedConfigure
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &SharedConfigure{}
	}
	return &data
}

// NewConstantDictFromJson 从JSON字符串创建ConstantDict实例
func NewConstantDictFromJson(v string) *ConstantDict {
	var data ConstantDict
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &ConstantDict{}
	}
	return &data
}

// Clone 创建当前ConstantDict实例的深拷贝
func (c *ConstantDict) Clone() *ConstantDict {
	if c == nil {
		return &ConstantDict{}
	}
	return &ConstantDict{
		ID:       c.ID,
		Value:    c.Value,
		Label:    c.Label,
		Dict:     c.Dict,
		DictName: c.DictName,
		Project:  c.Project,
		Type:     c.Type,
	}
}
