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

package dispatcher

import (
	"fmt"
	"net/http"

	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
)

// GetWorkerEndpoint 根据给定的 PathToEntity 获取对应的 Worker Endpoint
//
// 参数:
//   - event: *core.Event 事件对象
//
// 返回值:
//   - string: 返回对应的 Worker Endpoint，如果获取失败则返回空字符串
//
// 功能:
//
//	向主网关发送请求，获取与给定事件相关的 Worker Endpoint。如果主网关Endpoint未设置或事件为空，返回空字符串。
func GetWorkerEndpoint(event *core.Event) string {
	// 检查主网关Endpoint是否已设置
	if _mainGatewayEndpoint == "" {
		logx.Log().Error("Main Gateway Endpoint is not set")
		return ""
	}

	// 检查 event 是否完整
	if event.IsEmpty() {
		return ""
	}

	// 向主网关发送请求，获取 Endpoint 信息
	resp, err := Event(_mainGatewayEndpoint, types.W_T_G_GET_ENDPOINT_BY_EVENT, event.Raw(), nil)
	if err != nil || resp.Status() != http.StatusOK {
		logx.Debug(fmt.Sprintf("GetWorkerEndpoint failed,err: %v, resp: %+v", err, resp))
		return ""
	}
	return resp.TemporaryData()
}

// ReportEndpoint 向主网关上报 Endpoint 信息
//
// 参数:
//   - endpoint: *core.Endpoint 需要上报的 Endpoint 对象
//
// 返回值:
//   - error: 如果上报成功返回 nil，否则返回错误信息
//
// 功能:
//
//	向主网关上报指定的 Endpoint 信息。如果 Endpoint 为空或上报失败，返回相应的错误信息。
func ReportEndpoint(endpoint *core.Endpoint) error {
	if endpoint == nil {
		return fmt.Errorf("endpoint is nil")
	}
	resp, err := Event(_mainGatewayEndpoint, types.GW_T_G_REPORT_ENDPOINT, endpoint, nil)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("report endpoint failed, response is nil")
	}
	if resp.Status() != http.StatusOK {
		return fmt.Errorf("report endpoint failed : %s", resp.TemporaryData())
	}
	return nil
}
