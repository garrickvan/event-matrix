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

package gnetimpl

import (
	"github.com/garrickvan/event-matrix/serverx/gnetx"
	"github.com/garrickvan/event-matrix/worker/types"
)

type WorkerIntranetServer struct {
	*gnetx.IntranetServer

	cfg *types.WorkerServerConfig
	ws  types.WorkerServer
}

func NewWorkerIntranetServer(cfg *types.WorkerServerConfig, ws types.WorkerServer) *WorkerIntranetServer {
	s := &WorkerIntranetServer{
		cfg: cfg,
		ws:  ws,
	}
	// 初始化内部对象
	s.IntranetServer = gnetx.NewIntranetServer(
		cfg.ServerId,
		cfg.IntranetPort,
		cfg.IntranetSecret,
		cfg.IntranetSecretAlgor,
		routeEntrance,
		s)
	return s
}
