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
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// InvokeAiModel 调用AI, 输入系统提示和角色提示, 返回响应文本, 响应ID, 错误信息
func InvokeAiModel(config *AiAssistantConfig, params *AskParams) (response string, nextId string, err error) {
	if config == nil || params == nil {
		return "", "", fmt.Errorf("config and params are required")
	}
	if config.ApiKey == "" {
		return "", "", fmt.Errorf("API key is required")
	}
	temperature := params.Temperature
	if temperature == 0 {
		temperature = config.Temperature
	}

	switch config.Supplier {
	case AliyunSupplier:
		return invokeAliyunAiModel(config, params, temperature)
	}

	return "", "", errors.New("unsupported AI supplier")
}

// invokeAliyunAiModel 调用阿里云AI服务
func invokeAliyunAiModel(config *AiAssistantConfig, params *AskParams, temperature float32) (response string, nextId string, err error) {
	requestData := AliyunChatRequest{
		Model:  config.ModelName,
		Stream: config.Stream,
		Messages: []AliyunMessage{
			{Role: "system", Content: params.SystemPrompt},
			{Role: "user", Content: params.RolePrompt},
		},
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request data: %v", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest(
		"POST",
		"https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	var responseBuilder bytes.Buffer
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", "", fmt.Errorf("failed to read response: %v", err)
		}

		if bytes.HasPrefix(line, []byte("data: ")) {
			data := bytes.TrimPrefix(line, []byte("data: "))
			data = bytes.TrimSpace(data)

			if string(data) == "[DONE]" {
				break
			}

			var streamResponse AliyunStreamResponse
			if err := json.Unmarshal(data, &streamResponse); err != nil {
				fmt.Printf("解析错误: %v\n", err)
				continue
			}

			if len(streamResponse.Choices) > 0 {
				content := streamResponse.Choices[0].Delta.Content
				responseBuilder.WriteString(content)
			}
		}
	}
	return responseBuilder.String(), "", nil
}
