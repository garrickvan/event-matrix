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
	"errors"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/worker/intranet/dispatcher"
)

// AiCaller 是一个用于调用AI助手的服务调用者
type AiCaller struct {
	aiAssistantEndpoint string
}

// aiCaller 是 AiCaller 的单例实例
var (
	AiEndpointEvent = &core.Event{
		Project: core.INTERNAL_PROJECT,
		Version: constant.INITIAL_VERSION,
		Context: AiAssistantWorkerContext,
		Entity:  AiAssistantWorkerEntity,
	}
)

// init 是包的初始化函数，用于生成 AiEndpointEvent 的签名
func init() {
	AiEndpointEvent.GenerateSign()
}

// NewAiCaller 返回 AiCaller 的单例实例
func NewAiCaller() *AiCaller {
	return &AiCaller{
		aiAssistantEndpoint: "",
	}
}

// Ping 检查指定AI助手的可用性
func (ac *AiCaller) Ping(ids string) (map[string]bool, error) {
	if ac == nil || ids == "" {
		return nil, errors.New("AI调用器未初始化或请求为空")
	}
	if err := ac.initEndpoint(); err != nil {
		return nil, err
	}
	resp, err := dispatcher.Event(ac.aiAssistantEndpoint, G_T_W_AI_ASSISTANT_PING, ids, nil)
	if err != nil {
		ac.aiAssistantEndpoint = "" // 网络错误，清空aiAssistantEndpoint
		return nil, err
	}
	result := make(map[string]bool)
	if err := jsonx.UnmarshalFromStr(resp.TemporaryData(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// initEndpoint 加载AI助手的端点地址
func (ac *AiCaller) initEndpoint() error {
	if ac.aiAssistantEndpoint == "" {
		endpoint := dispatcher.GetWorkerEndpoint(AiEndpointEvent)
		if endpoint == "" {
			return errors.New("获取AI助手地址失败，网络错误或AI助手未启动")
		} else {
			ac.aiAssistantEndpoint = endpoint
		}
	}
	if ac.aiAssistantEndpoint == "" {
		return errors.New("AI助手地址为空")
	}
	return nil
}

// Ask 向AI助手发送请求并获取回答
func (ac *AiCaller) Ask(params AskParams) (string, error) {
	if err := ac.initEndpoint(); err != nil {
		return "", err
	}
	resp, err := dispatcher.Event(ac.aiAssistantEndpoint, G_T_W_AI_ASSISTANT_CHAT_STRING, params, nil)
	if err != nil {
		ac.aiAssistantEndpoint = "" // 网络错误，清空aiAssistantEndpoint
		return "", err
	}
	return resp.TemporaryData(), nil
}
