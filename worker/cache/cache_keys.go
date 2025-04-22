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

// event-matrix/worker/cache/cache_keys.go
package cache

import "strings"

// ConstantCacheKey 生成常量缓存键
func ConstantCacheKey(project, dict string) string {
	return strings.Join([]string{"constant", project, dict}, ":")
}

// EntityCacheKey 生成实体缓存键
func EntityCacheKey(project, context, entity, version string) string {
	parts := []string{
		"entity",
		project,
		context,
		entity,
		version,
	}
	return strings.Join(parts, ":")
}

// EntityAttrCacheKey 生成实体属性缓存键
func EntityAttrCacheKey(project, context, entity, version string) string {
	parts := []string{
		"entity_attr",
		project,
		context,
		entity,
		version,
	}
	return strings.Join(parts, ":")
}

// ContextCacheKey 生成上下文缓存键
func ContextCacheKey(project, version string) string {
	parts := []string{
		"context",
		project,
		version,
	}
	return strings.Join(parts, ":")
}

// EntityEventCacheKey 生成实体事件缓存键
func EntityEventCacheKey(project, context, entity, version string) string {
	parts := []string{
		"entity_event",
		project,
		context,
		entity,
		version,
	}
	return strings.Join(parts, ":")
}

// CachedUserUcodeKey 生成用户Ucode缓存键
func CachedUserUcodeKey(ucode string) string {
	return strings.Join([]string{"user", ucode}, ":")
}

// CachedUserIdKey 生成用户ID缓存键
func CachedUserIdKey(id string) string {
	return strings.Join([]string{"user_id", id}, ":")
}

// WorkerCacheKey 生成工作者缓存键
func WorkerCacheKey(project, context, entity, version string) string {
	return strings.Join([]string{"worker", project, context, entity, version}, ":")
}
