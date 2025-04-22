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

// Package gnetx 提供基于gnet的网络通信实现
package gnetx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils/buffertool"
	"github.com/garrickvan/event-matrix/utils/fastconv"
)

const (
	PING_TIMEOUT = 3 * time.Second // ping包响应超时时间
)

// gnetConnection 表示一个网络连接，并记录了该连接最后一次使用的时间
// 用于连接池的连接管理和过期检测
type gnetConnection struct {
	net.Conn           // 内嵌标准网络连接
	lastUsed time.Time // 最后一次使用时间
}

// pingPkg 预先构建的ping请求包，用于发送心跳检测
// 通过闭包在包初始化时构建，避免重复创建
func pingPkg() []byte {
	req := &RequestPacketImpl{
		PayloadType: serverx.CONTENT_TYPE_PING,
	}
	data, _ := req.Marshal()
	header := buildRpcHeader(data, false)
	return append(header, data...)
}

// Ping 发送ping包到连接并检查响应
// 用于验证连接是否仍然可用
func (c *gnetConnection) Ping() error {
	if _, err := c.Write(pingPkg()); err != nil {
		return err
	}
	c.SetReadDeadline(time.Now().Add(PING_TIMEOUT))
	receivedHeader := make([]byte, HEADER_LEN)
	if _, err := io.ReadFull(c, receivedHeader); err != nil {
		return fmt.Errorf("error reading ping response header: %v", err)
	}
	length, isCompressed, err := parseHeader(receivedHeader)
	if err != nil {
		return fmt.Errorf("error parsing ping response header: %v", err)
	}
	bodyBuf, release := buffertool.GetBuffer(int(length))
	defer release()
	if _, err := io.ReadFull(c, bodyBuf); err != nil {
		return fmt.Errorf("error reading ping message: %v", err)
	}
	var response serverx.ResponsePacket
	response, err = UnPackResponse(bodyBuf, isCompressed)
	if err != nil {
		return fmt.Errorf("error unmarshalling ping response: %v", err)
	}
	if response == nil {
		return errors.New("empty ping response")
	}
	if response.Status() != http.StatusOK {
		return fmt.Errorf("ping response status code: %d", response.Status())
	}
	return nil
}

// endpointPool 表示一个连接池，用于管理特定端点的连接
type endpointPool struct {
	pool chan *gnetConnection // 连接池通道
	mu   sync.Mutex           // 互斥锁，保护连接池操作
}

// Client 是一个网络客户端，负责管理连接池和发送请求
type Client struct {
	connectionExpired time.Duration // 连接过期时间
	writeTimeout      time.Duration // 写超时时间
	maxIdleConns      int           // 最大空闲连接数
	connPools         sync.Map      // 连接池映射，key为endpoint
	stopChan          chan struct{} // 停止信号通道
	statementIp       string        // 客户端声明的IP地址
	compress          bool          // 是否启用压缩
}

// NewClient 创建一个新的Client实例，并初始化连接池清理机制
//
// 参数：
//   - maxIdleConns: 每个endpoint的最大空闲连接数
//   - connectionExpired: 连接过期时间
//   - writeTimeout: 写操作超时时间
//
// 返回值：
//   - *Client: 新创建的客户端实例
func NewClient(maxIdleConns int, connectionExpired time.Duration, writeTimeout time.Duration) *Client {
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}
	if connectionExpired <= 0 {
		connectionExpired = 5 * time.Minute
	}
	if writeTimeout <= 0 {
		writeTimeout = 30 * time.Second
	}

	c := &Client{
		maxIdleConns:      maxIdleConns,
		connectionExpired: connectionExpired,
		writeTimeout:      writeTimeout,
		stopChan:          make(chan struct{}),
	}

	go c.cleanupPool()
	return c
}

// SetCompress 设置是否启用压缩
func (c *Client) SetCompress(compress bool) {
	c.compress = compress
}

// getConn 从连接池获取一个有效的连接，若无则新建连接
// 采用三阶段获取策略：快速单连接尝试、批量处理、创建新连接
func (c *Client) getConn(endpoint string) (*gnetConnection, error) {
	const maxBatchSize = 5 // 每批次最大处理连接数

	// 原子获取连接池
	poolAny, _ := c.connPools.LoadOrStore(endpoint, &endpointPool{
		pool: make(chan *gnetConnection, c.maxIdleConns),
	})
	pool := poolAny.(*endpointPool)

	// 第一阶段：快速单连接尝试（无锁竞争优化）
	if conn := c.tryGetSingleConn(pool); conn != nil {
		return conn, nil
	}

	// 第二阶段：批量处理连接（可控批次大小）
	if conn := c.tryBatchConn(pool, maxBatchSize); conn != nil {
		return conn, nil
	}

	// 第三阶段：最终兜底创建新连接
	return c.createNewConn(endpoint)
}

// tryGetSingleConn 无锁快速路径（单个连接检查）
// 尝试快速获取一个可用连接，最多尝试3次
func (c *Client) tryGetSingleConn(pool *endpointPool) *gnetConnection {
	for i := 0; i < 3; i++ { // 最多尝试3次快速获取
		pool.mu.Lock()
		if len(pool.pool) == 0 {
			pool.mu.Unlock()
			return nil
		}
		conn := <-pool.pool
		pool.mu.Unlock()

		if err := conn.Ping(); err == nil {
			return conn
		}
		conn.Close()
	}
	return nil
}

// tryBatchConn 批量处理路径（可控批次）
// 一次性检查多个连接，提高获取效率
func (c *Client) tryBatchConn(pool *endpointPool, batchSize int) *gnetConnection {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	var validConn *gnetConnection
	var staleConns []*gnetConnection

	// 批量取出部分连接
	for i := 0; i < batchSize && len(pool.pool) > 0; i++ {
		conn := <-pool.pool
		if err := conn.Ping(); err == nil {
			if validConn == nil {
				validConn = conn // 立即返回首个有效连接
			} else {
				staleConns = append(staleConns, conn) // 暂存其他有效连接
			}
		} else {
			conn.Close()
		}
	}

	// 回填剩余有效连接
	for _, conn := range staleConns {
		select {
		case pool.pool <- conn:
		default:
			conn.Close()
		}
	}

	return validConn
}

// createNewConn 创建新连接
// 当连接池中无可用连接时，创建一个新的TCP连接
func (c *Client) createNewConn(endpoint string) (*gnetConnection, error) {
	rawConn, err := net.DialTimeout("tcp", endpoint, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error connecting to server: %v", err)
	}
	return &gnetConnection{
		Conn:     rawConn,
		lastUsed: time.Now(),
	}, nil
}

// putConn 将连接放回连接池，若连接池已满则关闭连接
func (c *Client) putConn(endpoint string, conn *gnetConnection) {
	poolAny, exists := c.connPools.Load(endpoint)
	if !exists {
		conn.Close()
		return
	}
	pool := poolAny.(*endpointPool)

	pool.mu.Lock()
	defer pool.mu.Unlock()

	conn.lastUsed = time.Now()
	select {
	case pool.pool <- conn:
	default:
		conn.Close()
	}
}

// cleanupPool 定期清理过期的连接
// 在后台运行，直到收到停止信号
func (c *Client) cleanupPool() {
	ticker := time.NewTicker(c.connectionExpired / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopChan:
			return
		}
	}
}

// cleanup 检查并关闭过期的连接，移除空的连接池
func (c *Client) cleanup() {
	c.connPools.Range(func(key, value interface{}) bool {
		endpoint := key.(string)
		pool := value.(*endpointPool)

		pool.mu.Lock()
		defer pool.mu.Unlock()

		var validConns []*gnetConnection
		for len(pool.pool) > 0 {
			conn := <-pool.pool
			if time.Since(conn.lastUsed) < c.connectionExpired {
				validConns = append(validConns, conn)
			} else {
				conn.Close()
			}
		}

		if len(validConns) == 0 {
			c.connPools.Delete(endpoint)
		} else {
			for _, conn := range validConns {
				select {
				case pool.pool <- conn:
				default:
					conn.Close()
				}
			}
		}
		return true
	})
}

// Close 关闭Client实例，停止连接池清理机制
func (c *Client) Close() {
	close(c.stopChan)
}

// SetIp 设置客户端的源IP地址
func (c *Client) SetIp(ip string) {
	c.statementIp = ip
}

// emptyCallChain 空的调用链，用于默认值
var emptyCallChain = make([]string, 0)

// Ping 对指定的端点发送ping请求以检测连接状态
func (c *Client) Ping(endpoint string) error {
	conn, err := c.getConn(endpoint)
	if err != nil {
		return err
	}
	defer c.putConn(endpoint, conn)
	return conn.Ping()
}

// Post 向指定端点发送一个POST请求
//
// 参数：
//   - endpoint: 目标端点地址
//   - typz: 请求内容类型
//   - payload: 请求负载
//   - xdata: 额外数据
//   - callChain: 调用链信息
//
// 返回值：
//   - *ResponsePacketImpl: 响应消息
//   - error: 错误信息
func (c *Client) Post(endpoint string, typz serverx.CONTENT_TYPE, payload []byte, xdata string, callChain []string) (response serverx.ResponsePacket, err error) {
	if callChain == nil {
		callChain = emptyCallChain
	}
	msg := &RequestPacketImpl{
		PayloadType: typz,
		XData:       xdata,
		Payload:     fastconv.BytesToString(payload),
		SourceIP:    c.statementIp,
		CallChain:   strings.Join(callChain, constant.SPLIT_CHAR),
	}
	response, err = c.sendRequest(endpoint, msg, c.compress)
	if response == nil && err == nil {
		return nil, errors.New("nil response")
	}
	if response != nil && response.Status() == StatusGnetHeaderError {
		return nil, fmt.Errorf("protocol header error")
	}
	return response, err
}

// sendRequest 发送一个请求到指定端点，并接收响应
//
// 参数：
//   - endpoint: 目标端点地址
//   - msg: 请求消息
//   - compressed: 是否启用压缩
//
// 返回值：
//   - *ResponsePacketImpl: 响应消息
//   - error: 错误信息
func (c *Client) sendRequest(endpoint string, msg *RequestPacketImpl, compressed bool) (serverx.ResponsePacket, error) {
	conn, err := c.getConn(endpoint)
	if err != nil {
		return nil, err
	}

	defer func() {
		if r := recover(); r != nil || err != nil {
			// 发生panic或业务错误时关闭连接
			conn.Close()
		} else {
			// 仅当成功时才放回连接池
			c.putConn(endpoint, conn)
		}
	}()

	resp, err := send(conn.Conn, msg, compressed, c.writeTimeout)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// send 通过网络连接发送请求并接收响应
//
// 参数：
//   - conn: 网络连接
//   - msg: 请求消息
//   - compressed: 是否启用压缩
//   - timeout: 超时时间
//
// 返回值：
//   - *ResponsePacketImpl: 响应消息
//   - error: 错误信息
func send(conn net.Conn, msg *RequestPacketImpl, compressed bool, timeout time.Duration) (serverx.ResponsePacket, error) {
	conn.SetDeadline(time.Now().Add(timeout))
	defer conn.SetDeadline(time.Time{})

	var msgBytes []byte = msg.Pack(compressed)
	sendHeader := buildRpcHeader(msgBytes, compressed)

	if _, err := conn.Write(sendHeader); err != nil {
		return nil, fmt.Errorf("error writing header: %v", err)
	}

	if len(msgBytes) > 1024*1024 {
		if _, err := io.Copy(conn, bytes.NewReader(msgBytes)); err != nil {
			return nil, fmt.Errorf("error sending message body: %v", err)
		}
	} else {
		if _, err := conn.Write(msgBytes); err != nil {
			return nil, fmt.Errorf("error sending message body: %v", err)
		}
	}

	receivedHeader := make([]byte, HEADER_LEN)
	if _, err := io.ReadFull(conn, receivedHeader); err != nil {
		return nil, fmt.Errorf("error reading response header: %v", err)
	}
	length, isCompressed, err := parseHeader(receivedHeader)
	if err != nil {
		return nil, fmt.Errorf("error parsing response header: %v", err)
	}

	bodyBuf, release := buffertool.GetBuffer(int(length))
	defer release()

	if _, err := io.ReadFull(conn, bodyBuf); err != nil {
		return nil, fmt.Errorf("error reading response message: %v", err)
	}

	resp, err := UnPackResponse(bodyBuf, isCompressed)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("empty response")
	}
	// 拷贝一次临时数据，防止数据被覆写，简化释放逻辑
	if len(resp.TemporaryData()) > 0 {
		if pkg, ok := resp.(*ResponsePacketImpl); ok {
			pkg.Payload = strings.Clone(pkg.Payload)
		} else {
			return nil, errors.New("invalid response type")
		}
	}
	return resp, nil
}
