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
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/serverx/gnetx"
	"github.com/garrickvan/event-matrix/utils"
	"github.com/garrickvan/event-matrix/utils/encryptx"
	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/spf13/cast"
)

var (
	_client              *IntraServiceClient
	_mainGatewayEndpoint string
)

type IntraServiceClient struct {
	client     *gnetx.Client
	secret     string
	secretAlgo string
}

// 创建一个新的 IntraServiceClient 实例
func NewIntraServiceClient(
	maxIdleConns int, connectionExpired, writeTimeout time.Duration,
	myIp, secret, secretAlgo string, compress bool,
) *IntraServiceClient {
	client := gnetx.NewClient(maxIdleConns, connectionExpired, writeTimeout)
	client.SetIp(myIp)
	client.SetCompress(compress)
	return &IntraServiceClient{
		client:     client,
		secret:     secret,
		secretAlgo: secretAlgo,
	}
}

// 向指定的 endpoint 发送 POST 请求，并对请求参数进行加密，响应数据进行解密
func (c *IntraServiceClient) Post(endpoint string, typz types.INTRANET_EVENT_TYPE, params string, callChain []string) (response serverx.ResponsePacket, err error) {
	paramsBytes := fastconv.StringToBytes(params)
	cipherParamsBytes, err := encryptx.Encrypt(paramsBytes, c.secret, c.secretAlgo)
	if err != nil {
		logx.Debug("encrypt params failed", err)
		return nil, err
	}
	response, err = c.client.Post(endpoint, serverx.CONTENT_TYPE_STRING, cipherParamsBytes, strconv.Itoa(int(typz)), callChain)
	if err != nil {
		logx.Debug("post request failed:", err, "endpoint:", endpoint)
		return nil, err
	}
	dataBytes := fastconv.StringToBytes(response.TemporaryData())
	decryptedBytes, err := encryptx.Decrypt(dataBytes, c.secret, c.secretAlgo)
	if err != nil {
		logx.Debug("decrypt response failed", err)
		return nil, err
	}
	if pkg, ok := response.(*gnetx.ResponsePacketImpl); ok {
		pkg.Payload = fastconv.BytesToString(decryptedBytes)
	} else {
		return nil, errors.New("invalid response packet type")
	}
	return response, nil
}

// 初始化 IntraServiceClient
func InitClient(
	maxIdleConns int, connectionExpired, writeTimeout time.Duration,
	gatewayEndpoint string,
	myIp, secret, secretAlgo string, compress bool,
) {
	if !utils.IsEndpoint(gatewayEndpoint) {
		logx.Log().Error("invalid gateway endpoint")
		return
	}

	_mainGatewayEndpoint = gatewayEndpoint
	if _client != nil {
		logx.Log().Info("client already initialized, you are going to overwrite the previous initialization")
		_client.client.Close()
	}
	_client = NewIntraServiceClient(
		maxIdleConns, connectionExpired, writeTimeout,
		myIp, secret, secretAlgo, compress,
	)
}

// 获取 IntraServiceClient 实例，如果没有初始化则创建一个默认实例
func client() *IntraServiceClient {
	if _client == nil {
		logx.Log().Warn("client not initialized, using default client")
		return NewIntraServiceClient(10, 10*time.Second, 10*time.Second, "127.0.0.1", "", "", false)
	}
	return _client
}

// 内部事件调用 WILLDO：对 gateway 请求进行负载均衡，并返回结果
// Event 函数用于处理事件请求，并将请求发送到指定的端点。
// 该函数会检查请求的调用链，防止循环调用，并收集调用链信息以供后续统计使用。
//
// 参数:
//   - endpoint: 目标端点的URL，表示请求将要发送到的地址。
//   - typz: 事件类型，表示请求的事件类型，类型为 types.INTRANET_EVENT_TYPE。
//   - params: 请求参数，表示要发送的请求参数，通常为字符串或结构体。
//   - request: 请求上下文，包含请求的调用链和事件信息，类型为 serverx.RequestContext。
//
// 返回值:
//   - response: 返回的响应消息，类型为 *gnetx.ResponsePacketImpl，表示从目标端点返回的响应。
//   - err: 返回的错误信息，表示在请求过程中发生的任何错误。
func Event(endpoint string, typz types.INTRANET_EVENT_TYPE, strOrJson interface{}, request serverx.RequestContext) (response serverx.ResponsePacket, err error) {
	var chains []string
	if request != nil {
		chains = request.CallChain()
		e := request.Event()
		if e != nil {
			label := e.GetFullEventLabel()
			if label != "" {
				if utils.StrContains(chains, label) {
					return nil, errors.New("event call chain circular")
				} else {
					chains = append(chains, label)
				}
			}
		}
	}
	// WILLDO: 收集调用链信息，提供给 gateway 进行数据统计
	if paramStr, ok := strOrJson.(string); ok {
		return client().Post(endpoint, typz, paramStr, chains)
	} else {
		paramStr, err := jsonx.MarshalToStr(strOrJson)
		if err != nil {
			return nil, err
		}
		return client().Post(endpoint, typz, paramStr, chains)
	}
}

// 获取指定 endpoint 的负载均衡状态
func EndpointLoadRate(endpoint string) float64 {
	resp, err := client().Post(endpoint, types.G_T_W_GET_LOADE_RATE, "", nil)
	if err != nil || resp.Status() != http.StatusOK {
		return -1
	}
	if rate, err := cast.ToFloat64E(resp.TemporaryData()); err == nil {
		return rate
	}
	return -1
}
