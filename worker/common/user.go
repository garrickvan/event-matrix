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

package common

import (
	"net/http"
	"strings"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/intranet/dispatcher"
	"github.com/garrickvan/event-matrix/worker/types"
	"github.com/spf13/cast"
)

// 获取用户ID，如果需要认证，则验证用户认证，否则从事件中获取用户ID
func GetUserId(ctx types.WorkerContext, event *core.Event, needAuth bool, ignoreExpired bool) (string, constant.RESPONSE_CODE) {
	if needAuth {
		return verifyUserAuth(ctx, event, ignoreExpired)
	}
	return tryGetUserId(ctx, event), constant.SUCCESS
}

// 从事件中获取用户code，并根据code获取用户ID
func tryGetUserId(ctx types.WorkerContext, event *core.Event) string {
	userId := ""
	if event == nil {
		return userId
	}
	if event.AccessToken != "" {
		claims, err := jsonx.GetJwtTokenClaims(event.AccessToken)
		if err != nil && claims != nil {
			ucode := cast.ToString(claims["u"])
			ucode = strings.TrimSpace(ucode)
			if ucode != "" {
				userId = dispatcher.GetUserIdByCode(ucode)
			}
		}
	}
	return userId
}

// 验证用户认证
func verifyUserAuth(ctx types.WorkerContext, e *core.Event, ignoreExpired bool) (string, constant.RESPONSE_CODE) {
	inType := types.W_T_G_VERIFY_EVENT
	if ignoreExpired {
		inType = types.W_T_G_VERIFY_EVENT_WITHOUT_EXPIRED
	}
	resp, err := dispatcher.Event(ctx.Server().GatewayIntranetEndpoint(), inType, e.Raw(), ctx)
	if err != nil {
		logx.Log().Error("内部调用错误： " + err.Error())
		return "", constant.INVALID_PARAM
	}
	if resp.Status() == http.StatusOK {
		return resp.TemporaryData(), constant.SUCCESS
	}
	return "", constant.RESPONSE_CODE(resp.TemporaryData())
}
