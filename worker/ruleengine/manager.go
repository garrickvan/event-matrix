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

package ruleengine

import (
	"errors"
	"net/http"
	"sync"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/rulego/rulego"
	rtypes "github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/action"
)

// RuleEngineManagerImpl 是规则引擎管理器的实现，负责管理多个规则引擎实例。
type RuleEngineManagerImpl struct {
	ws types.WorkerServer

	ruleEngines  sync.Map       // 存储规则引擎实例的映射，键为规则引擎的唯一标识
	globalConfig *rtypes.Config // 全局配置，用于创建新的规则引擎
}

// NewRuleEngineManager 创建一个新的 RuleEngineManagerImpl 实例。
func NewRuleEngineManager(ws types.WorkerServer) *RuleEngineManagerImpl {
	defaultCfg := rulego.NewConfig()
	return &RuleEngineManagerImpl{
		ws:           ws,
		globalConfig: &defaultCfg,
	}
}

// AddRuleEngine 根据 Worker 实例添加规则引擎。
// 从 Worker 对应的实体中获取业务规则，并将其添加到规则引擎中。
func (rm *RuleEngineManagerImpl) AddRuleEngine(w *types.Worker) error {
	entity := rm.ws.DomainCache().Entity(types.PathToEntityFromWorker(w))
	if entity == nil {
		return errors.New("entity is nil")
	}
	ruleStr := entity.BusinessRules
	if ruleStr == "" {
		return errors.New("ruleStr is nil")
	}
	rules := []core.BusinessRules{}
	err := jsonx.UnmarshalFromStr(ruleStr, &rules)
	if err != nil {
		return errors.New("解析规则失败: " + entity.Name + " ,错误信息: " + err.Error())
	}
	for _, rule := range rules {
		rm.updateRuleEngine(w.GetVersionEntityLabel(), rule)
	}
	return nil
}

// SetGlobalConfig 设置全局配置，用于创建新的规则引擎。
func (rm *RuleEngineManagerImpl) SetGlobalConfig(config *rtypes.Config) {
	if config == nil {
		return
	}
	rm.globalConfig = config
}

// RuleUpdateParam 是更新规则时使用的参数结构体。
type RuleUpdateParam struct {
	Rules              []core.BusinessRules `json:"rules"`                // 要更新的规则列表
	EntityVersionLabel string               `json:"entity_version_label"` // 实体版本标签
}

// HandleRuleUpdate 处理规则更新请求。
// 解析传入的规则更新参数，并更新相应的规则引擎。
func (rm *RuleEngineManagerImpl) HandleRuleUpdate(ctx types.WorkerContext, paramStr string) error {
	var params RuleUpdateParam
	err := jsonx.UnmarshalFromStr(paramStr, &params)
	if err != nil {
		logx.Log().Warn("解析规则更新请求失败: " + err.Error())
		return ctx.SetStatus(http.StatusBadRequest).Response([]byte(constant.FAIL_TO_PROCESS))
	}
	for _, rule := range params.Rules {
		rm.updateRuleEngine(params.EntityVersionLabel, rule)
	}
	return ctx.SetStatus(http.StatusOK).Response([]byte(constant.SUCCESS))
}

// updateRuleEngine 更新或创建规则引擎。
// 根据实体标签和规则ID，更新现有的规则引擎或创建一个新的规则引擎。
func (rm *RuleEngineManagerImpl) updateRuleEngine(entityLabel string, rule core.BusinessRules) {
	idKey := entityLabel + "_" + rule.ID
	var eg *rtypes.RuleEngine = nil
	if oldAny, ok := rm.ruleEngines.Load(idKey); ok && oldAny != nil {
		if old, ok := oldAny.(*rtypes.RuleEngine); ok {
			eg = old
		}
	}
	logx.Debug("更新规则引擎: " + entityLabel + " ,规则ID: " + rule.ID)
	if eg != nil {
		// 更新规则引擎
		if err := (*eg).ReloadSelf([]byte(rule.Context)); err != nil {
			logx.Log().Warn("更新规则引擎失败: " + entityLabel + " ,错误信息: " + err.Error())
			return
		}
		logx.Debug("更新规则引擎成功: " + entityLabel + " ,规则ID: " + rule.ID)
	} else {
		// 创建规则引擎
		var cfg rtypes.Config
		if rm.globalConfig != nil {
			cfg = *rm.globalConfig
		} else {
			cfg = rulego.NewConfig()
		}
		eg, err := rulego.New(rule.ID, []byte(rule.Context), rulego.WithConfig(cfg))
		if err != nil {
			logx.Log().Warn("创建规则引擎失败: " + entityLabel + " ,错误信息: " + err.Error())
			return
		} else {
			logx.Debug("创建规则引擎成功: " + entityLabel + " ,规则ID: " + rule.ID)
		}
		rm.ruleEngines.Store(idKey, &eg)
	}
}

// Engine 根据实体标签和规则ID获取对应的规则引擎实例。
func (rm *RuleEngineManagerImpl) Engine(entityLabel string, ruleId string) *rtypes.RuleEngine {
	idKey := entityLabel + "_" + ruleId
	if egAny, ok := rm.ruleEngines.Load(idKey); ok && egAny != nil {
		if eg, ok := egAny.(*rtypes.RuleEngine); ok {
			return eg
		}
	}
	return nil
}

// RegisterRuleFunc 注册自定义规则函数。
// 将自定义的函数注册到规则引擎中，以便在规则执行时调用。
func (rm *RuleEngineManagerImpl) RegisterRuleFunc(funcName string, ruleFunc types.RuleFunc) {
	action.Functions.Register(funcName, func(ctx rtypes.RuleContext, msg rtypes.RuleMsg) {
		ruleFunc(ctx, msg, rm.ws)
	})
}
