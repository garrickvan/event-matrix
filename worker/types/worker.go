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
	"crypto/md5"
	"encoding/hex"
	"strings"
	"sync"

	"github.com/garrickvan/event-matrix/constant"
)

// Worker 结构体表示一个工作者实例，包含了工作者的各种属性和状态信息
// 该结构体用于与数据库交互，同时也用于 JSON 数据的序列化和反序列化
type Worker struct {
	// 工作者的唯一标识，作为数据库的主键
	ID string `json:"id" gorm:"primaryKey"`
	// 服务器的唯一标识，数据库中创建索引以加快查询速度
	ServerId string `json:"serverId" gorm:"index"`
	// 项目名称，数据库中创建索引以加快查询速度
	Project string `json:"project" gorm:"index"`
	// 版本标签，数据库中创建索引以加快查询速度
	VersionLabel string `json:"versionLabel" gorm:"index"`
	// 上下文信息，数据库中创建索引以加快查询速度
	Context string `json:"context" gorm:"index"`
	// 实体信息
	Entity string `json:"entity" gorm:"index"`

	// 事件模式，使用自定义的常量类型
	Mode constant.EVENT_MODE `json:"mode"`
	// 数据库配置键
	CfgKey string `json:"cfgKey"`
	// 心跳间隔，单位为秒
	HeartbeatGap int `json:"heartbeatGap"`
	// 负载均衡间隔，单位为秒
	RebalanceTime int `json:"rebalanceTime"`
	// 公网端点
	PublicEndpoint string `json:"publicEndpoint"`
	// 内域端点
	IntranetEndpoint string `json:"intranetEndpoint"`

	// 自定义执行器列表，使用逗号分隔
	CustomExecutors string `json:"customExecutors"`
	// 任务执行器列表，使用逗号分隔
	TaskExecutors string `json:"taskExecutors"`
	// 创建时间戳
	CreatedAt int64 `json:"createdAt"`
	// 更新时间戳
	UpdatedAt int64 `json:"updatedAt"`
	// 当前服务器的时区偏移量
	UtcOffset int `json:"utcOffset"`

	// 负载均衡权重，不存储到数据库，参与 JSON 序列化
	LoadRate float64 `gorm:"-" json:"loadRate"`
	// 是否同步表结构，不存储到数据库，也不参与 JSON 序列化
	SyncSchema bool `gorm:"-" json:"-"`
	// 最后一次心跳时间戳，不存储到数据库，参与 JSON 序列化
	LastHeartbeat int64 `gorm:"-" json:"lastHeartbeat"`

	// 自定义执行器映射，并发安全，不存储到数据库，也不参与 JSON 序列化
	customExecutorMap *sync.Map `gorm:"-" json:"-"`
	// 任务执行器映射，并发安全，不存储到数据库，也不参与 JSON 序列化
	taskExecutorMap *sync.Map `gorm:"-" json:"-"`
}

// NewWorker 函数用于创建一个新的 Worker 实例
//
// 参数:
// - project: 项目名称
// - versionLabel: 版本标签
// - context: 上下文信息
// - entity: 实体信息
// - cfgKey: 配置键
// - rebalanceTime: 负载均衡时间间隔（秒），如果小于等于0则默认设置为3秒
//
// 返回值:
// - *Worker: 返回新创建的 Worker 实例
func NewWorker(project, versionLabel, context, entity, cfgKey string, rebalanceTime int) *Worker {
	if rebalanceTime <= 0 {
		rebalanceTime = 3 // 默认负载均衡每3秒一次
	}
	worker := &Worker{
		Project:           project,
		VersionLabel:      versionLabel,
		Context:           context,
		Entity:            entity,
		CfgKey:            cfgKey,
		RebalanceTime:     rebalanceTime,
		SyncSchema:        true,
		customExecutorMap: &sync.Map{},
		taskExecutorMap:   &sync.Map{},
	}
	return worker
}

/*
*
颗粒度到实体，数据库ID用ServerId + Projcet + VersionLabel + Context + Entity + Mode生成
最好每个worker在一台服务器上只运行一个进程实例，非要同一个服务器对象运行多个实例，请确保不要使用同一个日志目录
*
*/
func (w *Worker) GenID() string {
	parts := []string{
		w.ServerId,
		w.Project,
		w.VersionLabel,
		w.Context,
		w.Entity,
		string(w.Mode),
	}
	data := strings.Join(parts, "")
	hash := md5.Sum([]byte(data))
	w.ID = hex.EncodeToString(hash[:])
	return w.ID
}

// AddCustomExecutor 向 Worker 添加一个自定义执行器
//
// 参数:
//
//	name: 自定义执行器的名称
//	executor: 自定义执行器对象
func (w *Worker) AddCustomExecutor(name string, executor WorkerExecutor) {
	if w.customExecutorMap == nil {
		w.customExecutorMap = &sync.Map{}
	}
	w.customExecutorMap.Store(name, executor)
}

// FindCustomExecutor 从Worker的自定义执行器映射中查找指定名称的执行器
//
// 参数:
//
//	name: 要查找的执行器名称
//
// 返回值:
//
//	WorkerExecutor: 如果找到指定的执行器，则返回该执行器；否则返回nil
//	bool: 如果找到指定的执行器，则返回true；否则返回false
func (w *Worker) FindCustomExecutor(name string) (WorkerExecutor, bool) {
	if w.customExecutorMap == nil {
		return nil, false
	}

	if f, ok := w.customExecutorMap.Load(name); ok {
		if w, ok := f.(WorkerExecutor); ok {
			return w, ok
		}
	}
	return nil, false
}

// AddTaskExecutor 将指定的任务执行器添加到 Worker 的任务执行器映射中
//
// 参数:
//
//	name - 任务执行器的名称，类型为 string
//	executor - 任务执行器，类型为 WorkerTaskExecutor
func (w *Worker) AddTaskExecutor(name string, executor WorkerTaskExecutor) {
	if w.taskExecutorMap == nil {
		w.taskExecutorMap = &sync.Map{}
	}
	w.taskExecutorMap.Store(name, executor)
}

// FindTaskExecutor 方法根据给定的任务名称查找对应的任务执行器。
//
// 参数：
//
//	name - string 类型，表示任务名称。
//
// 返回值：
//
//	WorkerTaskExecutor - 如果找到任务执行器，则返回该执行器，否则返回 nil。
//	bool - 如果找到任务执行器，则返回 true，否则返回 false。
func (w *Worker) FindTaskExecutor(name string) (WorkerTaskExecutor, bool) {
	if w.taskExecutorMap == nil {
		return nil, false
	}

	if f, ok := w.taskExecutorMap.Load(name); ok {
		if w, ok := f.(WorkerTaskExecutor); ok {
			return w, ok
		}
	}
	return nil, false
}

// BuildExecutors 构建并生成自定义执行器和任务执行器
//
// 参数:
//
//	无
//
// 返回值:
//
//	无
//
// 说明:
//
//	该方法用于生成并设置自定义执行器（CustomExecutors）和任务执行器（TaskExecutors）。
//	首先检查自定义执行器映射（customExecutorMap）是否为空，如果为空，则将CustomExecutors设置为空字符串并返回。
//	否则，遍历customExecutorMap并将键（执行器名称）添加到executors切片中，
//	然后使用constant.SPLIT_CHAR将executors切片连接成字符串并赋值给CustomExecutors。
//	接着检查任务执行器映射（taskExecutorMap）是否为空，如果为空，则将TaskExecutors设置为空字符串并返回。
//	否则，遍历taskExecutorMap并将键（任务名称）添加到taskExecutors切片中，
//	然后使用constant.SPLIT_CHAR将taskExecutors切片连接成字符串并赋值给TaskExecutors。
func (w *Worker) BuildExecutors() {
	// 生成customExecutors
	if w.customExecutorMap == nil {
		w.CustomExecutors = ""
		return
	}
	executors := []string{}
	w.customExecutorMap.Range(func(k, v interface{}) bool {
		if s, ok := k.(string); ok {
			executors = append(executors, s)
		}
		return true
	})
	w.CustomExecutors = strings.Join(executors, constant.SPLIT_CHAR)
	// 生成taskExecutors
	if w.taskExecutorMap == nil {
		w.TaskExecutors = ""
		return
	}
	taskExecutors := []string{}
	w.taskExecutorMap.Range(func(k, v interface{}) bool {
		if s, ok := k.(string); ok {
			taskExecutors = append(taskExecutors, s)
		}
		return true
	})
	w.TaskExecutors = strings.Join(taskExecutors, constant.SPLIT_CHAR)
}

// IsIncomplete 判断一个 Worker 是否未完成
//
// 如果 Worker 的 ID、ServerId、PublicEndpoint、Project、VersionLabel、Context、Entity、Mode 中任何一个为空，则返回 true，表示该 Worker 未完成；否则返回 false，表示该 Worker 已完成。
func (w *Worker) IsIncomplete() bool {
	return w.ID == "" || w.ServerId == "" || w.PublicEndpoint == "" || w.Project == "" || w.VersionLabel == "" || w.Context == "" || w.Entity == "" || w.Mode == ""
}

// GetVersionEntityLabel 返回包含项目、上下文、实体和版本标签的字符串
// 形如 sys.user.avatar@0.1.0 的唯一标识
// 格式为：项目名.上下文.实体@版本标签
//
// 参数：
// - 无
//
// 返回值：
// - string: 包含项目、上下文、实体和版本标签的字符串
func (w *Worker) GetVersionEntityLabel() string {
	parts := []string{w.Project, ".", w.Context, ".", w.Entity, "@", w.VersionLabel}
	return strings.Join(parts, "")
}

// BuildWorkerFromVersionEntityLabel 根据给定的版本实体标签生成Worker对象
//
// 参数：
//
//	lable: 版本实体标签，格式为 "domain@versionLabel"，其中 domain 格式为 "project.context.entity"
//
// 返回值：
//
//	*Worker: 返回生成的 Worker 对象指针，如果解析失败则返回 nil
func BuildWorkerFromVersionEntityLabel(lable string) *Worker {
	parts := strings.Split(lable, "@")
	if len(parts) != 2 {
		return nil
	}
	domain, versionLabel := parts[0], parts[1]
	parts = strings.Split(domain, ".")
	if len(parts) != 3 {
		return nil
	}
	project, context, entity := parts[0], parts[1], parts[2]
	return NewWorker(project, versionLabel, context, entity, "", 0)
}

// GetTabelName 获取Worker对象的表名
//
// 参数：
//
//	无
//
// 返回值：
//
//	string: 表名，格式为"Context_Entity"
func (w *Worker) GetTabelName() string {
	parts := []string{
		w.Context,
		w.Entity,
	}
	return strings.Join(parts, "_")
}
