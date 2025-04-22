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

package database

import (
	"time"

	"gorm.io/gorm"
)

// DBManager 定义数据库管理器的标准接口
// 提供数据库连接的注册、使用、健康检查和资源管理功能
type DBManager interface {
	// RegisterDB 注册一个新的数据库连接
	// 参数：
	//   - cfg: 数据库配置信息
	// 返回：
	//   - error: 注册过程中的错误，成功则为nil
	RegisterDB(cfg *DBConf) error

	// ResetHealthCheckInterval 重置数据库连接健康检查的间隔时间
	// 参数：
	//   - interval: 新的健康检查间隔时间
	ResetHealthCheckInterval(interval time.Duration)

	// Use 获取指定名称的数据库连接实例
	// 参数：
	//   - dbName: 数据库名称
	// 返回：
	//   - *gorm.DB: 数据库连接实例，如果不存在则返回nil
	Use(dbName string) *gorm.DB

	// HasDB 检查指定名称的数据库配置是否存在
	// 参数：
	//   - dbName: 数据库名称
	// 返回：
	//   - bool: 数据库配置存在返回true，否则返回false
	HasDB(dbName string) bool

	// Close 关闭数据库管理器，释放所有资源
	// 返回：
	//   - error: 关闭过程中的错误，成功则为nil
	Close() error
}
