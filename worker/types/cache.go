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

package types

import (
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/cachex"
)

// DefaultCache 定义了一个默认缓存接口，用于处理用户相关的缓存操作。
type DefaultCache interface {
	// Impl 返回底层的 LocalCache 实例。
	Impl() *cachex.LocalCache
}

// DomainCache 定义了一个领域缓存接口，用于处理与项目和实体相关的缓存操作。
type DomainCache interface {
	// Constants 获取指定项目和字典的常量列表。
	Constants(project, dict string) []core.ConstantDict

	// Entity 根据路径获取实体信息。
	Entity(e PathToEntity) *core.Entity

	// EntityEvent 根据路径获取实体事件信息。
	EntityEvent(e PathToEvent) *core.EntityEvent

	// EntityEvents 根据路径获取实体的所有事件列表。
	EntityEvents(e PathToEntity) []core.EntityEvent

	// EntityAttrs 根据路径获取实体的所有属性列表。
	EntityAttrs(e PathToEntity) []core.EntityAttribute

	// Impl 返回底层的 LocalCache 实例。
	Impl() *cachex.LocalCache
}
