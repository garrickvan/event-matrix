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
	"net/http"
	"strings"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
)

// LoadSharedCfgFromGateway 从网关加载共享配置
//
// 参数:
//   - keys: []string 需要加载的配置键列表
//
// 返回值:
//   - map[string]*core.SharedConfigure: 返回加载的共享配置映射，键为配置键，值为配置对象
//
// 功能:
//
//	根据传入的配置键列表，从远程配置中心加载共享配置。如果加载失败或解析失败，返回 nil。
func LoadSharedCfgFromGateway(keys []string) map[string]*core.SharedConfigure {
	if len(keys) == 0 {
		logx.Debug("没有需要预加载的共享配置")
		return nil
	}
	resp, err := Event(_mainGatewayEndpoint, types.W_T_G_GET_SHARED_CONFIGURE, strings.Join(keys, constant.SPLIT_CHAR), nil)
	if err != nil {
		logx.Error("从远程配置中心获取共享配置失败: ", err.Error(), keys)
		return nil
	}
	if resp.TemporaryData() == "" {
		logx.Error("从远程配置中心获取共享配置失败，返回值为空")
		return nil
	}
	var configs []core.SharedConfigure
	err = jsonx.UnmarshalFromStr(resp.TemporaryData(), &configs)
	if err != nil {
		logx.Error("从远程配置中心获取共享配置(", keys, ")失败，解析返回值失败", err.Error(), "原值: ", resp)
		return nil
	}
	result := map[string]*core.SharedConfigure{}
	for _, v := range configs {
		result[v.Key] = &v
	}
	return result
}

// ReportConfigUsedBy 上报配置使用情况
//
// 参数:
//   - key: string 配置键
//   - workerId: string 工作节点ID
//
// 功能:
//
//	向远程配置中心上报指定配置键被某个工作节点使用的情况。如果上报失败，记录错误日志。
func ReportConfigUsedBy(key, workerId string) {
	if key == "" || workerId == "" {
		logx.Error("上报配置使用情况失败, key或workerId为空")
		return
	}
	go func() {
		paramStr := key + constant.SPLIT_CHAR + workerId
		resp, err := Event(_mainGatewayEndpoint, types.W_T_G_REPORT_CONF_USED_BY, paramStr, nil)
		if err != nil {
			logx.Error("向远程配置中心上报配置使用情况失败: ", err.Error())
			return
		}
		if resp.Status() != http.StatusOK {
			logx.Error("向远程配置中心上报配置使用情况失败, "+"key: "+key+" workerId: "+workerId+"，返回值: ", resp.TemporaryData())
		} else {
			logx.Debug("向远程配置中心上报配置使用情况成功，key: ", key, " workerId	: ", workerId)
		}
	}()
}
