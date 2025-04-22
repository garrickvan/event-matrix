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

package cache

import (
	"github.com/garrickvan/event-matrix/utils/cachex"
	"github.com/garrickvan/event-matrix/worker/types"
)

type DefaultCacheImpl struct {
	cache *cachex.LocalCache
	ws    types.WorkerServer
}

func NewDefaultCacheImpl(maxMen int64, defaultTimeout int, ws types.WorkerServer) (*DefaultCacheImpl, error) {
	c := cachex.LocalCache{}
	err := c.InitCache(maxMen, defaultTimeout)
	if err != nil {
		return nil, err
	}
	return &DefaultCacheImpl{
		cache: &c,
		ws:    ws,
	}, err
}

func (c *DefaultCacheImpl) Impl() *cachex.LocalCache {
	return c.cache
}
