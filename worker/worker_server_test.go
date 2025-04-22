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

package worker

import (
	"testing"
)

func TestGetTag(t *testing.T) {
	// attr := core.EntityAttribute{
	// 	Code:    "name",
	// 	Unique:  true,
	// 	Indexed: true,
	// }
	// res := getTag(&attr)
	// fmt.Println(res)
	// attr.Unique = false
	// res = getTag(&attr)
	// fmt.Println(res)
	// attr.Indexed = false
	// res = getTag(&attr)
	// fmt.Println(res)
}

func TestInitWorkerServer(t *testing.T) {
	s := TwoWayWorkerServerSettings{
		CfgKey:                  "w_1",
		IntranetSecret:          "d634xvmbnwg0Nu0G3dnNLlkJHXdHFKFALSIYTyrnPEX78PbZCN",
		IntranetSecretAlgor:     "aes-256",
		GatewayIntranetEndpoint: "127.0.0.1:10000",
	}
	_ = NewTwoWayWorkerServer(s)
}
