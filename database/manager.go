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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/garrickvan/event-matrix/utils/logx"
	"gorm.io/gorm"
)

// GormDBManager 用于管理多个Gorm数据库连接的管理器
// 提供数据库连接池管理、自动重连和健康检查功能
type GormDBManager struct {
	dbs                        sync.Map      // 并发安全的数据库连接存储，键为数据库名称，值为*gorm.DB实例
	dbConfs                    sync.Map      // 并发安全的数据库配置存储，键为数据库名称，值为*DBConf实例
	checkOnce                  sync.Once     // 确保健康检查协程只启动一次
	stopChan                   chan struct{} // 用于停止健康检查协程的信号通道
	mu                         sync.RWMutex  // 用于保护defaultHealthCheckInterval的读写锁
	defaultHealthCheckInterval time.Duration // 默认的健康检查间隔时间，可动态调整
}

// NewGormDBManager 创建一个新的GormDBManager实例
// 初始化内部状态并设置默认的健康检查间隔为30秒
// 返回可立即使用的数据库管理器实例
func NewGormDBManager() *GormDBManager {
	return &GormDBManager{
		stopChan:                   make(chan struct{}),
		defaultHealthCheckInterval: 30 * time.Second,
	}
}

// ResetHealthCheckInterval 重置健康检查的间隔时间
// 参数：
//   - interval: 新的健康检查间隔时间
//
// 注意：如果设置的间隔小于5秒，将自动调整为5秒，以防止过于频繁的检查
func (m *GormDBManager) ResetHealthCheckInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if interval <= 5*time.Second {
		interval = 5 * time.Second
	}
	m.defaultHealthCheckInterval = interval
}

// RegisterDB 注册一个新的数据库连接到管理器
// 参数：
//   - cfg: 数据库配置信息，包含连接参数
//
// 返回：
//   - error: 注册过程中的错误，成功则为nil
//
// 注意：首次注册数据库时会自动启动健康检查协程
func (m *GormDBManager) RegisterDB(cfg *DBConf) error {
	if cfg == nil {
		return errors.New("数据库配置不能为空")
	}

	db, err := getGormDBWithRetry(*cfg, 3)
	if err != nil {
		logx.Log().Error("数据库" + cfg.DBName + "连接失败: " + err.Error())
		return err
	}
	m.dbs.Store(cfg.DBName, db)
	m.dbConfs.Store(cfg.DBName, cfg)
	m.checkOnce.Do(func() {
		go m.dbHealthCheck()
	})
	return nil
}

// Use 获取指定名称的数据库连接实例
// 参数：
//   - dbName: 数据库名称
//
// 返回：
//   - *gorm.DB: 数据库连接实例，如果不存在或连接失败则返回nil
//
// 注意：如果连接不存在或已断开，会尝试自动重新连接
func (m *GormDBManager) Use(dbName string) *gorm.DB {
	if db, ok := m.dbs.Load(dbName); ok {
		return db.(*gorm.DB)
	}

	// 尝试重新注册
	if err := m.reRegisterDB(dbName); err != nil {
		logx.Log().Error("重新注册数据库" + dbName + "失败: " + err.Error())
		return nil
	}

	if db, ok := m.dbs.Load(dbName); ok {
		return db.(*gorm.DB)
	}
	logx.Log().Error("数据库" + dbName + "重新注册后仍然不存在")
	return nil
}

// reRegisterDB 重新注册指定名称的数据库连接
// 当数据库连接断开或失效时，使用已保存的配置重新建立连接
// 参数：
//   - dbName: 要重新注册的数据库名称
//
// 返回：
//   - error: 重新注册过程中的错误，成功则为nil
func (m *GormDBManager) reRegisterDB(dbName string) error {
	conf, ok := m.dbConfs.Load(dbName)
	if !ok {
		return errors.New("无法获取数据库配置")
	}

	dbConf, ok := conf.(*DBConf)
	if !ok {
		return errors.New("数据库配置类型错误")
	}

	db, err := getGormDBWithRetry(*dbConf, 3)
	if err != nil {
		return fmt.Errorf("重新连接失败: %w", err)
	}

	m.dbs.Store(dbName, db)
	return nil
}

// HasDB 检查是否存在指定名称的数据库配置
// 参数：
//   - dbName: 要检查的数据库名称
//
// 返回：
//   - bool: 如果数据库配置存在则返回true，否则返回false
//
// 注意：此方法仅检查配置是否存在，不保证连接是否有效
func (m *GormDBManager) HasDB(dbName string) bool {
	_, ok := m.dbConfs.Load(dbName)
	return ok
}

// dbHealthCheck 定期检查所有数据库连接的健康状态
// 在后台协程中运行，定期对所有注册的数据库连接执行ping操作
// 如果检测到连接异常，会从连接池中移除该连接
// 该方法包含panic恢复机制，确保健康检查不会因单次错误而停止
func (m *GormDBManager) dbHealthCheck() {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				logx.Log().Error("dbHealthCheck发生错误: " + err.Error())
			} else {
				logx.Log().Error("dbHealthCheck发生未知错误")
			}
			time.Sleep(5 * time.Second)
			go m.dbHealthCheck()
		}
	}()

	for {
		select {
		case <-time.After(m.getHealthCheckInterval()):
			m.dbs.Range(func(key, value interface{}) bool {
				dbName := key.(string)
				db := value.(*gorm.DB)

				sqlDB, err := db.DB()
				if err != nil {
					logx.Log().Error("获取数据库" + dbName + "连接失败: " + err.Error())
					m.dbs.Delete(dbName)
					return true // 继续遍历
				}

				if err := sqlDB.Ping(); err != nil {
					logx.Log().Error("数据库" + dbName + "连接异常: " + err.Error())
					sqlDB.Close()
					m.dbs.Delete(dbName)
				}
				return true // 继续遍历
			})

		case <-m.stopChan:
			return
		}
	}
}

// getHealthCheckInterval 获取当前的健康检查间隔时间
// 使用读锁保护，确保并发安全
// 返回：
//   - time.Duration: 当前配置的健康检查间隔时间
func (m *GormDBManager) getHealthCheckInterval() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultHealthCheckInterval
}

// Close 关闭所有数据库连接并停止健康检查
// 在应用程序退出前调用，确保资源正确释放
// 返回：
//   - error: 关闭过程中的错误，目前始终返回nil
//
// 注意：调用此方法后，管理器将不再可用，需要创建新的实例
func (m *GormDBManager) Close() error {
	close(m.stopChan)

	m.dbs.Range(func(key, value interface{}) bool {
		dbName := key.(string)
		db := value.(*gorm.DB)

		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		m.dbs.Delete(dbName)
		return true
	})

	return nil
}
