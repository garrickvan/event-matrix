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

package aiassistant

import (
	"net/http"
	"sync"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/intranet/dispatcher"
	"github.com/garrickvan/event-matrix/worker/types"
)

// AiAssistantCenter 定义了一个AI助手中心，用于管理AI助手的配置和处理相关事件
type AiAssistantCenter struct {
	worker *types.Worker
	svr    types.WorkerServer

	cfgs sync.Map // 使用 sync.Map 替代 map
}

// AiAssistantWorkerContext 定义了AI助手中心的上下文名称
const (
	AiAssistantWorkerContext = "gateway"
	AiAssistantWorkerEntity  = "ai_assistant"

	G_T_W_AI_ASSISTANT_PING        types.INTRANET_EVENT_TYPE = 31500
	G_T_W_AI_ASSISTANT_CHAT_STRING types.INTRANET_EVENT_TYPE = 31501
	G_T_W_AI_ASSISTANT_CHAT_STREAM types.INTRANET_EVENT_TYPE = 31502
)

// aiAssistant 是全局的AI助手中心实例
// aiAssistantWorker 是AI助手的工作者配置
var (
	aiAssistantWorker = types.Worker{
		Project:      core.INTERNAL_PROJECT,
		VersionLabel: constant.INITIAL_VERSION,
		Context:      AiAssistantWorkerContext,
		Entity:       AiAssistantWorkerEntity,
		SyncSchema:   false,
	}
)

// NewAiAssistantCenter 创建一个新的AI助手中心实例
func NewAiAssistantCenter(ws types.WorkerServer, cfgKey string) *AiAssistantCenter {
	aiAssistantWorker.CfgKey = cfgKey
	return &AiAssistantCenter{
		worker: &aiAssistantWorker,
		svr:    ws,
	}
}

// Setup 注册AI助手中心到网关，并初始化AI助手配置
func (ai *AiAssistantCenter) Setup() error {
	if err := ai.svr.RegisterWorker(ai.worker); err != nil {
		return err
	}
	ai.svr.RegisterPlugin(ai)
	if ai.worker.CfgKey != "" {
		dispatcher.ReportConfigUsedBy(ai.worker.CfgKey, ai.worker.ID)
	}
	return nil
}

// ReceiveCodes 返回AI助手中心能够处理的事件类型列表
func (a *AiAssistantCenter) ReceiveCodes() []types.INTRANET_EVENT_TYPE {
	return []types.INTRANET_EVENT_TYPE{G_T_W_AI_ASSISTANT_PING, G_T_W_AI_ASSISTANT_CHAT_STRING, G_T_W_AI_ASSISTANT_CHAT_STREAM}
}

// Handle 根据接收到的事件类型调用相应的处理方法
func (a *AiAssistantCenter) Handle(ctx types.WorkerContext, typz types.INTRANET_EVENT_TYPE) error {
	switch typz {
	case G_T_W_AI_ASSISTANT_PING:
		return a.HandlePing(ctx)
	case G_T_W_AI_ASSISTANT_CHAT_STRING:
		return a.HandleChatString(ctx)
	case G_T_W_AI_ASSISTANT_CHAT_STREAM: // WILLDO: 处理流式聊天
		return ctx.SetStatus(http.StatusNotImplemented).ResponseString("not supported yet")
	}
	return ctx.SetStatus(http.StatusNotFound).ResponseString(string(constant.UNSUPPORTED_EVENT))
}

// HandlePing 处理AI助手的PING请求，检查配置是否存在并返回结果
func (a *AiAssistantCenter) HandlePing(ctx types.WorkerContext) error {
	cfgKeys := fastconv.SafeSplitFromBytes(ctx.Body(), constant.SPLIT_CHAR)
	result := make(map[string]bool)
	for _, cfgKey := range cfgKeys {
		// check if cfgKey is already registered
		if _, ok := a.cfgs.Load(cfgKey); ok {
			result[cfgKey] = true
			continue
		}
		svr := ctx.Server()
		// get the config from shared config
		cfg := svr.SharedConfigure(cfgKey)
		if cfg == nil {
			logx.Debug("shared config not found: ", cfgKey)
			result[cfgKey] = false
			continue
		}
		// unmarshal the config
		c := &AiAssistantConfig{}
		err := jsonx.UnmarshalFromStr(cfg.Value, c)
		if err != nil {
			logx.Log().Error(err.Error())
			continue
		}
		// register the config
		a.cfgs.Store(cfgKey, c)
		result[cfgKey] = true
		dispatcher.ReportConfigUsedBy(cfgKey, a.worker.ID)
	}
	return ctx.SetStatus(http.StatusOK).ResponseJson(result)
}

// HandleChatString 处理AI助手的字符串聊天请求，调用AI模型并返回响应
func (a *AiAssistantCenter) HandleChatString(ctx types.WorkerContext) error {
	params := AskParams{}
	err := jsonx.UnmarshalFromBytes(ctx.Body(), &params)
	if err != nil {
		return ctx.SetStatus(http.StatusBadRequest).ResponseString(err.Error())
	}
	cfg, ok := a.cfgs.Load(params.AssistantId)
	if !ok {
		return ctx.SetStatus(http.StatusNotFound).ResponseString("assistant not found")
	}
	resp, _, err := InvokeAiModel(cfg.(*AiAssistantConfig), &params)
	if err != nil {
		return ctx.SetStatus(http.StatusInternalServerError).ResponseString(err.Error())
	}
	return ctx.SetStatus(http.StatusOK).ResponseString(resp)
}
