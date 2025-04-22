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
	"fmt"
	"net/http"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/cachex"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/intranet/dispatcher"
	"github.com/garrickvan/event-matrix/worker/types"
)

// DomainCacheImpl 实现了域缓存的功能
type DomainCacheImpl struct {
	cache *cachex.LocalCache // 本地缓存实例
	ws    types.WorkerServer // 工作服务器实例
}

// NewDomainCacheImpl 创建一个新的域缓存实例
func NewDomainCacheImpl(maxMen int64, defaultTimeout int, wm types.WorkerServer) (*DomainCacheImpl, error) {
	c := cachex.LocalCache{}
	err := c.InitCache(maxMen, defaultTimeout)
	if err != nil {
		return nil, err
	}
	return &DomainCacheImpl{
		cache: &c,
		ws:    wm,
	}, err
}

// EntityEvent 根据事件路径获取实体事件
func (dc *DomainCacheImpl) EntityEvent(e types.PathToEvent) *core.EntityEvent {
	// 版本为0.0.0的事件, 视为内部事件，直接返回空
	if e.Version == constant.INITIAL_VERSION {
		return nil
	}
	events := dc.EntityEvents(types.PathToEntityFromPathToEvent(e))
	if events == nil || len(events) == 0 {
		return nil
	}
	for _, event := range events {
		if event.Code == e.Event {
			return &event
		}
	}
	return nil
}

// Entity 根据实体路径获取实体
func (dc *DomainCacheImpl) Entity(e types.PathToEntity) *core.Entity {
	if e.Version == constant.INITIAL_VERSION {
		return nil
	}
	key := EntityCacheKey(e.Project, e.Context, e.Entity, e.Version)
	entity, found := dc.cache.GetOrHook(key, func() interface{} {
		w := dc.ws.GetWorkerByEvent(e)
		if w == nil {
			w = &types.Worker{}
		}
		p := types.PathToEntityFromWorker(w)
		resp, err := dispatcher.Event(dc.ws.GatewayIntranetEndpoint(), types.W_T_G_GET_ENTITY, p.ToStrArg(), nil)
		if err != nil || resp.Status() != http.StatusOK {
			logx.Debug("内部调用错误： " + err.Error())
			return nil
		}
		if resp.TemporaryData() == "" {
			logx.Debug("内部调用返回数据为空，entity: " + e.Entity + "不存在")
			return nil
		}
		entity := core.Entity{}
		err = jsonx.UnmarshalFromStr(resp.TemporaryData(), &entity)
		if err != nil {
			logx.Debug("内部调用返回数据格式错误： " + err.Error())
			return nil
		}
		return &entity
	})
	if found {
		if entity, ok := entity.(*core.Entity); ok && entity != nil {
			return entity
		}
	}
	return nil
}

var emptyEntityAttrs = make([]core.EntityAttribute, 0)

// EntityAttrs 根据实体路径获取实体属性
func (dc *DomainCacheImpl) EntityAttrs(e types.PathToEntity) []core.EntityAttribute {
	if e.IsIncomplete() {
		return emptyEntityAttrs
	}

	key := EntityAttrCacheKey(e.Project, e.Context, e.Entity, e.Version)
	data, found := dc.cache.GetOrHook(key, func() interface{} {
		resp, err := dispatcher.Event(dc.ws.GatewayIntranetEndpoint(), types.W_T_G_GET_ENTITY_ATTRS, e.ToStrArg(), nil)
		if err != nil || resp == nil || resp.Status() != http.StatusOK {
			logx.Error(fmt.Sprintf("获取属性失败 [%s] 错误: %v, 响应: %+v", e.ToStrArg(), err, resp))
			return emptyEntityAttrs
		}

		var rawData []interface{}
		if err := jsonx.UnmarshalFromStr(resp.TemporaryData(), &rawData); err != nil {
			logx.Error("属性数据解析失败: " + err.Error())
			return emptyEntityAttrs
		}

		attrs := make([]core.EntityAttribute, 0, len(rawData))
		for _, item := range rawData {
			if attr := core.NewEntityAttributeFromMap(item); attr != nil {
				attrs = append(attrs, *attr)
			}
		}
		return attrs
	})

	if !found {
		logx.Debug("属性未缓存: " + key)
	}

	if attrs, ok := data.([]core.EntityAttribute); ok {
		return attrs
	}
	return emptyEntityAttrs
}

var emptyEntityEvents = make([]core.EntityEvent, 0)

// EntityEvents 根据实体路径获取实体事件
func (dc *DomainCacheImpl) EntityEvents(e types.PathToEntity) []core.EntityEvent {
	if e.IsIncomplete() || e.Version == constant.INITIAL_VERSION {
		return emptyEntityEvents
	}

	key := EntityEventCacheKey(e.Project, e.Context, e.Entity, e.Version)
	data, found := dc.cache.GetOrHook(key, func() interface{} {
		resp, err := dispatcher.Event(dc.ws.GatewayIntranetEndpoint(), types.W_T_G_GET_ENTITY_EVENTS, e.ToStrArg(), nil)
		if err != nil || resp == nil || resp.Status() != http.StatusOK {
			logx.Error(fmt.Sprintf("获取事件失败 [%s] 错误: %v, 响应: %+v", e.ToStrArg(), err, resp))
			return emptyEntityEvents
		}

		var rawData []interface{}
		if err := jsonx.UnmarshalFromStr(resp.TemporaryData(), &rawData); err != nil {
			logx.Error("事件数据解析失败: " + err.Error())
			return emptyEntityEvents
		}

		events := make([]core.EntityEvent, 0, len(rawData))
		for _, item := range rawData {
			if event := core.NewEntityEventFromMap(item); event != nil {
				events = append(events, *event)
			}
		}
		return events
	})

	if !found {
		logx.Debug("事件未缓存: " + key)
	}

	if events, ok := data.([]core.EntityEvent); ok {
		return events
	}
	return emptyEntityEvents
}

// Constants 根据项目和字典名称获取常量
func (dc *DomainCacheImpl) Constants(project, dict string) []core.ConstantDict {
	if project == "" || dict == "" {
		return nil
	}
	key := ConstantCacheKey(project, dict)
	ins, find := dc.cache.GetOrHook(key, func() interface{} {
		paramStr := project + constant.SPLIT_CHAR + dict
		resp, err := dispatcher.Event(dc.ws.GatewayIntranetEndpoint(), types.W_T_G_GET_CONSTANTS, paramStr, nil)
		if err != nil {
			logx.Log().Error("内部调用错误： " + err.Error())
			return nil
		}
		if resp.TemporaryData() == "" {
			logx.Debug("内部调用返回数据为空，constant: " + project + "." + dict + "不存在")
			return nil
		}
		constantDicts := []core.ConstantDict{}
		err = jsonx.UnmarshalFromStr(resp.TemporaryData(), &constantDicts)
		if err != nil {
			logx.Log().Error("内部调用返回数据格式错误： " + err.Error())
			return nil
		}
		return constantDicts
	})
	if find && ins != nil {
		if constants, ok := ins.([]core.ConstantDict); ok && constants != nil {
			return constants
		}
	}
	return nil
}

// Impl 返回本地缓存实例
func (dc *DomainCacheImpl) Impl() *cachex.LocalCache {
	return dc.cache
}
