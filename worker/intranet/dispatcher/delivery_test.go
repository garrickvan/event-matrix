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
	"testing"
	"time"

	"github.com/garrickvan/event-matrix/utils/logx"
)

func initTestEnv() {
	logx.InitRuntimeLogger("logs", "debug", "", 20*time.Second)
	InitClient(
		10,
		10*time.Second,
		10*time.Second,
		"127.0.0.1:10001",
		"127.0.0.1",
		"d634xvmbnwg0Nu0G3dnNLlkJHXdHFKFALSIYTyrnPEX78PbZCN",
		"aes-256",
		false,
	)
}

const token = "e30.eyJ1IjoiRkFDMyIsImV4cCI6MTc0MTQ1NTM5OCwibWsiOiI3NjEzMyIsInMiOiJ3ZWJfYXBpIn0.ppbTJXWiQbq9tfAYpBySyyd5dEN6r4h1KBkW0KM1ivo"

func TestGetUserSensitiveInfo(t *testing.T) {
	initTestEnv()
	// 调用函数
	result, err := GetUserSensitiveInfo(token, []string{"2f604dca9fde5c9672be1be6e7517f088a473401"})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(result["2f604dca9fde5c9672be1be6e7517f088a473401"])
}

func TestSaveUserSensitiveInfo(t *testing.T) {
	initTestEnv()
	// 调用函数
	resp, err := SaveUserSensitiveInfo(token, map[string]string{
		"email_1": "example@qq.com",
	})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(resp)
}

func TestGetUserDetailInfo(t *testing.T) {
	initTestEnv()
	// 调用函数
	resp, err := GetUserDetailInfo(token)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(resp)
}
