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

// Package limiter 实现了电路断路器（Circuit Breaker）模式
// 该实现参考了 Microsoft 的断路器模式设计 (https://msdn.microsoft.com/en-us/library/dn589784.aspx)
// 以及 Sony 的 gobreaker 实现 (https://github.com/sony/gobreaker)
//
// 电路断路器模式用于处理分布式系统中的故障，通过三种状态（关闭、半开、打开）来管理对可能失败的操作的访问：
// - 关闭状态：允许请求通过，跟踪失败次数
// - 打开状态：立即拒绝请求，不调用实际服务
// - 半开状态：允许有限数量的请求通过，用于测试服务是否恢复
package limiter

import (
	"errors"
	"sync"
	"time"
)

// State 表示电路断路器的当前状态
// 断路器在这三种状态之间转换，以实现故障隔离和服务恢复
type State int

// 电路断路器的三种基本状态
const (
	StateClosed   State = iota // 关闭状态：正常工作，允许请求通过
	StateHalfOpen              // 半开状态：允许有限请求通过，用于探测服务是否恢复
	StateOpen                  // 打开状态：服务被认为不可用，快速拒绝所有请求
)

// 断路器状态的字符串表示
const (
	CIRCUITBREAKER_STATE_CLOSED  = "CLOSED"  // 关闭状态的字符串表示
	CIRCUITBREAKER_STATE_OPEN    = "OPEN"    // 打开状态的字符串表示
	CIRCUITBREAKER_STATE_HALF    = "HALF"    // 半开状态的字符串表示
	CIRCUITBREAKER_STATE_UNKNOWN = "UNKNOWN" // 未知状态的字符串表示
)

var (
	// ErrTooManyRequests 在断路器处于半开状态且请求数量超过限制时返回
	// 这种情况发生在服务正在恢复，但允许的探测请求数已用完时
	ErrTooManyRequests = errors.New("请求过多")

	// ErrOpenState 在断路器处于打开状态时返回
	// 表示服务当前被认为是不可用的，需要等待一段时间后才能尝试访问
	ErrOpenState = errors.New("失败次数过多，稍后重试")
)

// String 实现了stringer接口，将断路器状态转换为可读的字符串形式
// 用于日志记录和状态展示
func (s State) String() string {
	switch s {
	case StateClosed:
		return CIRCUITBREAKER_STATE_CLOSED
	case StateHalfOpen:
		return CIRCUITBREAKER_STATE_HALF
	case StateOpen:
		return CIRCUITBREAKER_STATE_OPEN
	default:
		return CIRCUITBREAKER_STATE_UNKNOWN
	}
}

// Counts 记录断路器的各种计数指标
// 这些计数用于决定是否触发断路器状态转换
// 计数会在状态改变时或在闭合状态达到指定间隔时被重置
type Counts struct {
	Requests             uint32 // 总请求次数
	TotalSuccesses       uint32 // 总成功次数
	TotalFailures        uint32 // 总失败次数
	ConsecutiveSuccesses uint32 // 连续成功次数
	ConsecutiveFailures  uint32 // 连续失败次数
}

// onRequest 在收到新请求时调用，递增请求计数器
// 这个方法在每个请求开始处理时被调用
func (c *Counts) onRequest() {
	c.Requests++
}

// onSuccess 在请求成功时调用，更新相关计数
// - 增加总成功次数和连续成功次数
// - 重置连续失败次数
func (c *Counts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

// onFailure 在请求失败时调用，更新相关计数
// - 增加总失败次数和连续失败次数
// - 重置连续成功次数
func (c *Counts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

// clear 重置所有计数器为0
// 在断路器状态改变或达到重置间隔时调用
func (c *Counts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// Settings 定义了电路断路器的配置参数
// 这些参数决定了断路器的行为特征，包括状态转换条件、超时时间等
type Settings struct {
	Name          string                                  // 断路器名称，用于标识和日志记录
	MaxRequests   uint32                                  // 半开状态时允许的最大请求数，为0时默认允许1个
	Interval      time.Duration                           // 关闭状态时清除计数的时间间隔，<=0时不清除
	Timeout       time.Duration                           // 打开状态持续时间，超时后转为半开状态，<=0时默认60秒
	ReadyToTrip   func(counts Counts) bool                // 决定是否触发断路器打开的函数，nil时使用默认策略
	OnStateChange func(name string, from State, to State) // 状态变化时的回调函数
	IsSuccessful  func(err error) bool                    // 判断请求是否成功的函数，nil时所有非nil错误都视为失败
}

// CircuitBreaker 实现了一个通用的电路断路器模式
// 泛型参数T允许断路器处理任意类型的返回值
// 通过状态机制防止持续发送可能失败的请求，从而避免级联故障
type CircuitBreaker[T any] struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	isSuccessful  func(err error) bool
	onStateChange func(name string, from State, to State)

	mutex      sync.Mutex
	state      State
	generation uint64
	counts     Counts
	expiry     time.Time
}

// TwoStepCircuitBreaker类似于CircuitBreaker，但它不是将功能包裹在一个函数周围，
// 而是仅检查请求是否可以继续进行，并期望调用者使用回调函数在单独的步骤中报告结果。
type TwoStepCircuitBreaker[T any] struct {
	cb *CircuitBreaker[T]
}

// NewCircuitBreaker根据给定的Settings返回一个新配置的电路断路器实例。
func NewCircuitBreaker[T any](st Settings) *CircuitBreaker[T] {
	cb := new(CircuitBreaker[T])

	cb.name = st.Name
	cb.onStateChange = st.OnStateChange

	if st.MaxRequests == 0 {
		cb.maxRequests = 1
	} else {
		cb.maxRequests = st.MaxRequests
	}

	if st.Interval <= 0 {
		cb.interval = defaultInterval
	} else {
		cb.interval = st.Interval
	}

	if st.Timeout <= 0 {
		cb.timeout = defaultTimeout
	} else {
		cb.timeout = st.Timeout
	}

	if st.ReadyToTrip == nil {
		cb.readyToTrip = defaultReadyToTrip
	} else {
		cb.readyToTrip = st.ReadyToTrip
	}

	if st.IsSuccessful == nil {
		cb.isSuccessful = defaultIsSuccessful
	} else {
		cb.isSuccessful = st.IsSuccessful
	}

	cb.toNewGeneration(time.Now())

	return cb
}

// NewTwoStepCircuitBreaker根据给定的Settings返回一个新配置的TwoStepCircuitBreaker实例。
func NewTwoStepCircuitBreaker[T any](st Settings) *TwoStepCircuitBreaker[T] {
	return &TwoStepCircuitBreaker[T]{
		cb: NewCircuitBreaker[T](st),
	}
}

const defaultInterval = time.Duration(0) * time.Second
const defaultTimeout = time.Duration(60) * time.Second

// defaultReadyToTrip是默认的ReadyToTrip函数实现，当连续失败次数超过5次时返回true。
func defaultReadyToTrip(counts Counts) bool {
	return counts.ConsecutiveFailures > 5
}

// defaultIsSuccessful是默认的IsSuccessful函数实现，对于非nil的错误返回false。
func defaultIsSuccessful(err error) bool {
	return err == nil
}

// Name返回电路断路器的名称。
func (cb *CircuitBreaker[T]) Name() string {
	return cb.name
}

// State返回电路断路器的当前状态。
func (cb *CircuitBreaker[T]) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts返回内部的计数器信息。
func (cb *CircuitBreaker[T]) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

// Execute运行给定的请求（如果电路断路器接受该请求）。
// 如果电路断路器拒绝请求，Execute会立即返回一个错误。
// 否则，Execute返回请求的结果。
// 如果在请求过程中发生了panic，电路断路器会将其当作一个错误处理，并再次抛出相同的panic。
func (cb *CircuitBreaker[T]) Execute(req func() (T, error)) (T, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		var defaultValue T
		return defaultValue, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, cb.isSuccessful(err))
	return result, err
}

// Name返回TwoStepCircuitBreaker的名称。
func (tscb *TwoStepCircuitBreaker[T]) Name() string {
	return tscb.cb.Name()
}

// State返回TwoStepCircuitBreaker的当前状态。
func (tscb *TwoStepCircuitBreaker[T]) State() State {
	return tscb.cb.State()
}

// Counts返回内部的计数器信息。
func (tscb *TwoStepCircuitBreaker[T]) Counts() Counts {
	return tscb.cb.Counts()
}

// Allow检查新请求是否可以继续进行。它返回一个回调函数，该回调函数应在单独的步骤中用于
// 注册请求的成功或失败情况。如果电路断路器不允许请求，它将返回一个错误。
func (tscb *TwoStepCircuitBreaker[T]) Allow() (done func(success bool), err error) {
	generation, err := tscb.cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	return func(success bool) {
		tscb.cb.afterRequest(generation, success)
	}, nil
}

// beforeRequest在请求前进行一些前置判断和操作，比如检查当前状态是否允许请求等，
// 并返回当前的请求批次（generation）以及可能出现的错误（如果不允许请求）。
func (cb *CircuitBreaker[T]) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrOpenState
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}

	cb.counts.onRequest()
	return generation, nil
}

// afterRequest在请求结束后根据请求是否成功来进行相应的状态更新等操作，
// 需要传入请求前获取的请求批次（before）以及请求是否成功（success）的标识。
func (cb *CircuitBreaker[T]) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// onSuccess在请求成功时根据当前电路断路器的状态（闭合或半开）进行相应的计数更新等操作，
// 比如在半开状态下成功次数达到一定数量时会将状态切换为闭合状态。
func (cb *CircuitBreaker[T]) onSuccess(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.onSuccess()
	case StateHalfOpen:
		cb.counts.onSuccess()
		if cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

// onFailure在请求失败时根据当前电路断路器的状态（闭合或半开）进行相应的状态切换等操作，
// 比如在闭合状态下失败次数满足条件时会将状态切换为打开状态。
func (cb *CircuitBreaker[T]) onFailure(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.onFailure()
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

// currentState根据当前时间等因素判断并返回电路断路器当前的实际状态以及对应的请求批次（generation）。
func (cb *CircuitBreaker[T]) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// setState用于设置电路断路器的状态，同时在状态改变时更新相关的计数、请求批次等信息，
// 并且如果配置了状态变更的回调函数（onStateChange），则会调用该函数。
func (cb *CircuitBreaker[T]) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// toNewGeneration用于切换到新的请求批次，会递增请求批次号，清除计数信息，
// 并根据当前的电路断路器状态（闭合、打开或半开）设置相应的过期时间。
func (cb *CircuitBreaker[T]) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts.clear()

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}
