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
	"strings"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/types"
)

// WILLDO: 根据用户特征，把请求分配到相关的gateway上，以提高gateway本地缓存命中率

// GetUserIdsBySearch 根据搜索条件获取用户ID列表
func GetUserIdsBySearch(param types.SearchByFieldParam) []string {
	paramStr, err := jsonx.MarshalToStr(param)
	if err != nil {
		logx.Log().Error("参数序列化错误： " + err.Error())
		return []string{}
	}
	resp, err := Event(_mainGatewayEndpoint, types.W_T_G_SEARCH_USER_INFO, paramStr, nil)
	if err != nil {
		logx.Log().Error("查询用户信息失败，错误信息： " + err.Error())
		return []string{}
	}
	return strings.Split(resp.TemporaryData(), constant.SPLIT_CHAR)
}

// GetUserIdByCode 根据用户编码获取用户ID
func GetUserIdByCode(code string) string {
	if code == "" {
		return ""
	}
	resp, err := Event(_mainGatewayEndpoint, types.W_T_G_GET_USER_ID_BY_UCODE, code, nil)
	if err != nil {
		logx.Debug("查询用户信息失败，错误信息： " + err.Error())
		return ""
	}
	if resp.Status() != http.StatusOK {
		logx.Debug("查询用户信息失败，错误信息： 状态码不为200")
		return ""
	}
	return strings.TrimSpace(resp.TemporaryData())
}

func GetUserSensitiveInfo(token string, infoHash []string) (map[string]string, error) {
	if token == "" || len(infoHash) == 0 {
		return nil, nil
	}
	params := strings.Join(append([]string{token}, infoHash...), constant.SPLIT_CHAR)
	resp, err := Event(_mainGatewayEndpoint, types.W_T_G_GET_USER_SENSITIVE_INFO, params, nil)
	if err != nil {
		logx.Debug("查询用户敏感信息失败，错误信息： " + err.Error())
		return nil, err
	}
	if resp.Status() != http.StatusOK {
		logx.Debug("查询用户敏感信息失败，错误信息： 状态码不为200")
	}
	result := make(map[string]string)
	if err = jsonx.UnmarshalFromStr(resp.TemporaryData(), &result); err != nil {
		logx.Debug("解析用户敏感信息失败，错误信息： " + err.Error())
		return nil, err
	}
	return result, nil
}

// SaveUserSensitiveInfo 设置用户敏感信息,infos为map[string]string类型，key为敏感信息类型，value为对应的敏感信息值，如：
//
//	infos := map[string]string{
//		"phone": "13800138000",
//		"email": "123456@qq.com",
//	}
//
// 返回值：map[string]string, error
//
//	map[string]string, key为设置成功的敏感信息类型，value为对应的敏感信息Hash值，保存好Hash值后，可用于查询用户敏感信息
func SaveUserSensitiveInfo(token string, infos map[string]string) (map[string]string, error) {
	if token == "" || len(infos) == 0 {
		return nil, nil
	}
	infos["_token_"] = token
	resp, err := Event(_mainGatewayEndpoint, types.W_T_G_SAVE_USER_SENSITIVE_INFO, infos, nil)
	if err != nil {
		logx.Debug("设置用户敏感信息失败，错误信息： " + err.Error())
		return nil, err
	}
	if resp.Status() != http.StatusOK {
		logx.Debug("设置用户敏感信息失败，错误信息： 状态码不为200")
		return nil, errors.New("状态码不为200，错误：" + resp.TemporaryData())
	}
	result := make(map[string]string)
	if err := jsonx.UnmarshalFromStr(resp.TemporaryData(), &result); err != nil {
		logx.Debug("解析用户敏感信息Hash值失败，错误信息： " + err.Error())
		return nil, err
	}
	return result, nil
}

// GetUserDetailInfo 获取用户详细信息
func GetUserDetailInfo(token string) (*core.User, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}
	resp, err := Event(_mainGatewayEndpoint, types.W_T_G_GET_USER_DETAIL, token, nil)
	if err != nil {
		logx.Debug("查询用户详细信息失败，错误信息： " + err.Error())
		return nil, err
	}
	if resp.Status() != http.StatusOK {
		logx.Debug("查询用户详细信息失败，错误信息： 状态码不为200")
		return nil, errors.New("状态码不为200，错误：" + resp.TemporaryData())
	}
	var user core.User
	if err = jsonx.UnmarshalFromStr(resp.TemporaryData(), &user); err != nil {
		logx.Debug("解析用户详细信息失败，错误信息： " + err.Error())
		return nil, err
	}
	return &user, nil
}
