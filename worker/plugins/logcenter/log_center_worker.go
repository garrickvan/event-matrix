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

package logcenter

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/intranet/dispatcher"
	"github.com/garrickvan/event-matrix/worker/types"
)

type LogCenter struct {
	svr    types.WorkerServer
	worker *types.Worker

	runtimeDBCfgKey, eventDBCfgKey string
}

/**
  日志中心的工作端, 需要更高并发时可重写为分布式架构
*/

const (
	RuntimeLogDB           = "runtime_log"
	EventLogDB             = "event_log"
	LogCenterWorkerContext = "gateway"
	LogCenterWorkerEntity  = "log_center"

	GW_T_W_RUNTIME_LOG_SUBMIT types.INTRANET_EVENT_TYPE = 31000
	GW_T_W_EVENT_LOG_SUBMIT   types.INTRANET_EVENT_TYPE = 31001
	G_T_W_LOG_CENTER_QUERY    types.INTRANET_EVENT_TYPE = 31002
)

var (
	logCenterWorker = types.Worker{
		Project:      core.INTERNAL_PROJECT,
		VersionLabel: constant.INITIAL_VERSION,
		Context:      LogCenterWorkerContext,
		Entity:       LogCenterWorkerEntity,
		SyncSchema:   false,
	}
)

func NewLogCenter(ws types.WorkerServer, runtimeDBCfgKey, eventDBCfgKey string) *LogCenter {
	logCenterWorker.CfgKey = eventDBCfgKey
	return &LogCenter{
		svr:             ws,
		worker:          &logCenterWorker,
		runtimeDBCfgKey: runtimeDBCfgKey,
		eventDBCfgKey:   eventDBCfgKey,
	}
}
func (lc *LogCenter) Setup() error {
	if err := lc.svr.Repo().AddDBFromSharedConfig(lc.runtimeDBCfgKey); err != nil {
		return err
	}
	if err := lc.svr.Repo().AddDBFromSharedConfig(lc.eventDBCfgKey); err != nil {
		return err
	}
	if err := lc.initDB(); err != nil {
		return err
	}
	if err := lc.svr.RegisterWorker(lc.worker); err != nil {
		return err
	}
	lc.svr.RegisterPlugin(lc)
	dispatcher.ReportConfigUsedBy(lc.runtimeDBCfgKey, lc.worker.ID)
	return nil
}

func (lc *LogCenter) initDB() error {
	noDBs := []string{}
	if !lc.svr.Repo().HasDB(RuntimeLogDB) {
		noDBs = append(noDBs, RuntimeLogDB)
	}
	if !lc.svr.Repo().HasDB(EventLogDB) {
		noDBs = append(noDBs, EventLogDB)
	}
	if len(noDBs) > 0 {
		return errors.New("缺少数据库：[" + strings.Join(noDBs, "、") + "], 请检查配置，无法启动日志中心服务")
	}
	if err := lc.svr.Repo().Use(RuntimeLogDB).AutoMigrate(&logx.LogEntry{}); err != nil {
		return err
	}
	if err := lc.svr.Repo().Use(EventLogDB).AutoMigrate(&core.EventLog{}); err != nil {
		return err
	}
	return nil
}

func (lc *LogCenter) ReceiveCodes() []types.INTRANET_EVENT_TYPE {
	return []types.INTRANET_EVENT_TYPE{GW_T_W_RUNTIME_LOG_SUBMIT, GW_T_W_EVENT_LOG_SUBMIT, G_T_W_LOG_CENTER_QUERY}
}

func (lc *LogCenter) Handle(ctx types.WorkerContext, typz types.INTRANET_EVENT_TYPE) error {
	if lc == nil {
		return ctx.SetStatus(http.StatusInternalServerError).Response([]byte("日志中心未配置"))
	}
	switch typz {
	case GW_T_W_RUNTIME_LOG_SUBMIT:
		return lc.handlerRuntimeLog(ctx)
	case GW_T_W_EVENT_LOG_SUBMIT:
		return lc.handlerEventLog(ctx)
	case G_T_W_LOG_CENTER_QUERY:
		return lc.handlerQueryLog(ctx)
	default:
		return ctx.SetStatus(http.StatusBadRequest).Response([]byte("日志中心不存在类型: " + fmt.Sprintf("%d", typz)))
	}
}

type LogListParam struct {
	LogType     string `json:"logType"`
	SearchField string `json:"searchField"`
	SearchValue string `json:"searchValue"`
	Page        int    `json:"page"`
	Size        int    `json:"size"`
}

const batchSize = 100

func (lc *LogCenter) handlerRuntimeLog(ctx types.WorkerContext) error {
	// 获取日志信息并转成日志对象
	logs := []logx.LogEntry{}
	err := jsonx.UnmarshalFromBytes(ctx.Body(), &logs)
	if err != nil {
		return ctx.SetStatus(http.StatusBadRequest).Response([]byte("添加运行日志失败"))
	}
	// 找出已保存的日志并舍弃
	logIds := []string{}
	for _, log := range logs {
		logIds = append(logIds, log.ID)
	}
	logsInDB := []logx.LogEntry{}
	ctx.Server().Repo().Use(RuntimeLogDB).Where("id in?", logIds).Find(&logsInDB)
	if len(logsInDB) > 0 {
		for _, log := range logsInDB {
			for i, l := range logs {
				if l.ID == log.ID {
					// 移除存在的，避免重复插入
					logs = append(logs[:i], logs[i+1:]...)
				}
			}
		}
	}
	// 新建日志
	if len(logs) > 0 {
		// 分批插入
		for i := 0; i < len(logs); i += batchSize {
			end := i + batchSize
			if end > len(logs) {
				end = len(logs)
			}
			db := ctx.Server().Repo().Use(RuntimeLogDB).Create(logs[i:end])
			if db.Error != nil {
				logx.Error("新增日志失败: " + db.Error.Error())
				return ctx.SetStatus(http.StatusInternalServerError).Response([]byte("新增日志失败"))
			}
		}
	}
	return ctx.SetStatus(http.StatusOK).Response([]byte(constant.SUCCESS))
}

func (lc *LogCenter) handlerEventLog(ctx types.WorkerContext) error {
	// 获取日志信息并转成事件对象
	logs := []logx.LogEntry{}
	err := jsonx.UnmarshalFromBytes(ctx.Body(), &logs)
	if err != nil {
		return ctx.SetStatus(http.StatusBadRequest).Response([]byte("添加事件日志失败"))
	}
	eventLogs := []core.EventLog{}
	for _, log := range logs {
		one := core.EventLog{}
		err = jsonx.UnmarshalFromStr(log.Msg, &one)
		if err != nil {
			return ctx.SetStatus(http.StatusBadRequest).Response([]byte("事件日志格式错误"))
		} else {
			eventLogs = append(eventLogs, one)
		}
	}
	// 找出已保存的事件
	eventIds := []string{}
	for _, event := range eventLogs {
		eventIds = append(eventIds, event.ID)
	}
	eventInDB := []core.EventLog{}
	ctx.Server().Repo().Use(EventLogDB).Where("id in?", eventIds).Find(&eventInDB)
	if len(eventInDB) > 0 {
		for _, event := range eventInDB {
			for i, e := range eventLogs {
				if e.ID == event.ID {
					// 从创建数组移除存在数据库的记录
					eventLogs = append(eventLogs[:i], eventLogs[i+1:]...)
				}
			}
		}
	}
	// 新建事件
	if len(eventLogs) > 0 {
		// 分批插入
		for i := 0; i < len(eventLogs); i += batchSize {
			end := i + batchSize
			if end > len(eventLogs) {
				end = len(eventLogs)
			}
			db := ctx.Server().Repo().Use(EventLogDB).Create(eventLogs[i:end])
			if db.Error != nil {
				logx.Log().Error("新增事件失败: " + db.Error.Error())
				return ctx.SetStatus(http.StatusInternalServerError).Response([]byte("新增事件失败"))
			}
		}
	}
	// 更新事件
	if len(eventInDB) > 0 {
		// 分批更新
		for i := 0; i < len(eventInDB); i += batchSize {
			end := i + batchSize
			if end > len(eventInDB) {
				end = len(eventInDB)
			}
			db := ctx.Server().Repo().Use(EventLogDB).Save(eventInDB[i:end])
			if db.Error != nil {
				logx.Log().Error("更新事件失败: " + db.Error.Error())
				return ctx.SetStatus(http.StatusInternalServerError).Response([]byte("更新事件失败"))
			}
		}
	}
	return ctx.SetStatus(http.StatusOK).Response([]byte(constant.SUCCESS))
}

func (lc *LogCenter) handlerQueryLog(ctx types.WorkerContext) error {
	param := LogListParam{}
	err := jsonx.UnmarshalFromBytes(ctx.Body(), &param)
	if err != nil {
		return ctx.SetStatus(http.StatusBadRequest).Response([]byte("参数异常，查询日志失败"))
	}
	if param.Size < 0 || param.Page <= 0 {
		return ctx.SetStatus(http.StatusBadRequest).Response([]byte("参数异常，查询日志失败"))
	}

	if param.LogType == logx.LogTypeEvent {
		return lc.queryEventLog(ctx, &param)
	}
	if param.LogType == logx.LogTypeRuntime {
		return lc.queryRuntimeLog(ctx, &param)
	}
	return ctx.SetStatus(http.StatusBadRequest).Response([]byte("日志类型不存在"))
}

func (lc *LogCenter) queryEventLog(ctx types.WorkerContext, param *LogListParam) error {
	db := ctx.Server().Repo().Use(EventLogDB).Model(&core.EventLog{})
	var logList []*core.EventLog
	resp := jsonx.DefaultJsonWithMsg(constant.SUCCESS, "查询成功")
	var userIds []string
	if param.SearchValue != "" && param.SearchField != "" {
		if param.SearchField == "creator_name" {
			search := types.SearchByFieldParam{
				Field:   "name",
				Keyword: param.SearchValue,
				Page:    1,
				Size:    20,
			}
			userIds = dispatcher.GetUserIdsBySearch(search)
			db.Where("creator in ?", userIds)
		} else if param.SearchField == "creator_unique_code" {
			search := types.SearchByFieldParam{
				Field:   "unique_code",
				Keyword: param.SearchValue,
				Page:    1,
				Size:    20,
			}
			userIds = dispatcher.GetUserIdsBySearch(search)
			db.Where("creator in ?", userIds)
		} else {
			db.Where(param.SearchField+" LIKE ?", "%"+param.SearchValue+"%")
		}
	}
	db.
		Offset((param.Page - 1) * param.Size).
		Limit(param.Size).
		Order("finish_at desc").
		Find(&logList)
	if len(logList) > 0 {
		db := ctx.Server().Repo().Use(EventLogDB).Model(&core.EventLog{})
		var count int64
		if param.SearchValue != "" && param.SearchField != "" {
			if param.SearchField == "creator_name" {
				db.Where("creator in ?", userIds)
			} else if param.SearchField == "creator_unique_code" {
				db.Where("creator in ?", userIds)
			} else {
				db.Where(param.SearchField+" LIKE ?", "%"+param.SearchValue+"%")
			}
		}
		db.Count(&count)
		jsonx.SetJsonList[*core.EventLog](resp, logList, count, param.Page)
	} else {
		resp.Size = 0
	}
	return ctx.SetStatus(http.StatusOK).ResponseJson(resp)
}

func (lc *LogCenter) queryRuntimeLog(ctx types.WorkerContext, param *LogListParam) error {
	db := ctx.Server().Repo().Use(RuntimeLogDB).Model(&logx.LogEntry{})
	var logList []*logx.LogEntry
	resp := jsonx.DefaultJsonWithMsg(constant.SUCCESS, "查询成功")
	if param.SearchValue != "" {
		db.Where(param.SearchField+" LIKE ?", "%"+param.SearchValue+"%")
	}
	db.
		Offset((param.Page - 1) * param.Size).
		Limit(param.Size).
		Order("created_at desc").
		Find(&logList)
	if len(logList) > 0 {
		db := ctx.Server().Repo().Use(RuntimeLogDB).Model(&logx.LogEntry{})
		var count int64
		if param.SearchValue != "" {
			db.Where(param.SearchField+" LIKE ?", "%"+param.SearchValue+"%")
		}
		db.Count(&count)
		jsonx.SetJsonList[*logx.LogEntry](resp, logList, count, param.Page)
	} else {
		resp.Size = 0
	}
	return ctx.SetStatus(http.StatusOK).ResponseJson(resp)
}
