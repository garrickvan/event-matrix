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

const (
	AliyunSupplier  = "aliyun"
	TencentSupplier = "tencent"
)

// AiAssistantConfig 定义了AI助手的配置参数
type AiAssistantConfig struct {
	ModelName   string  `json:"model"`       // 模型名称，指定使用的AI模型
	Stream      bool    `json:"stream"`      // 是否启用流式响应
	Temperature float32 `json:"temperature"` // 温度参数，控制生成文本的随机性
	ApiType     string  `json:"api_type"`    // API类型，指定使用的API接口类型
	ApiKey      string  `json:"api_key"`     // API密钥，用于身份验证
	Supplier    string  `json:"supplier"`    // 提供商，指定AI服务提供商
}

// AskParams 定义了向AI助手提问时的参数
type AskParams struct {
	AssistantId  string  `json:"assistantId"`  // AI助手的ID
	SystemPrompt string  `json:"systemPrompt"` // 系统提示信息，指导AI助手的行为
	RolePrompt   string  `json:"rolePrompt"`   // 角色提示信息，指定用户和AI助手的角色
	Temperature  float32 `json:"temperature"`  // 温度参数，控制生成文本的随机性
}

// ChatRequest 定义了发送给AI助手的聊天请求参数
type AliyunChatRequest struct {
	Model    string          `json:"model"`    // 模型名称，指定使用的AI模型
	Messages []AliyunMessage `json:"messages"` // 消息列表，包含对话历史
	Stream   bool            `json:"stream"`   // 是否启用流式响应
}

// Message 定义了聊天消息的结构
type AliyunMessage struct {
	Role    string `json:"role"`    // 角色，可以是"user"或"assistant"
	Content string `json:"content"` // 消息内容
}

// StreamResponse 定义了流式响应的结构
type AliyunStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"` // 流式返回的内容片段
		} `json:"delta"`
	} `json:"choices"` // 响应的选择列表，通常只有一个选择
}
