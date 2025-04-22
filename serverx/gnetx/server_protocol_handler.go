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

// Package gnetx 提供基于gnet的网络通信协议处理实现
package gnetx

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils/buffertool"
	"github.com/garrickvan/event-matrix/utils/encryptx"
	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/panjf2000/gnet/v2"
)

const (
	maxBufferSize         = 1024 * 1024     // 最大缓冲区大小，1MB
	StatusGnetHeaderError = 40000           // GNet 协议头错误状态码
	HEADER_LEN            = 8               // 消息头长度，包含4字节长度、1字节压缩标志、1字节协议版本、2字节CRC校验
	PROTOCOL_VERSION      = 1               // 协议版本号
	MESSAGE_SEND_TIMEOUT  = 5 * time.Second // 发送超时时间，单位秒
)

// buildRpcHeader 构建RPC消息头（包含CRC校验）
func buildRpcHeader(data []byte, compressed bool) []byte {
	header := make([]byte, HEADER_LEN)
	binary.BigEndian.PutUint32(header[:4], uint32(len(data))) // 添加消息长度
	if compressed {                                           // 添加压缩标志
		header[4] = 0x01
	} else {
		header[4] = 0x00
	}
	header[5] = PROTOCOL_VERSION // 添加协议版本号

	// 计算前6字节的CRC16校验值
	crc := crc16(header[:6])
	binary.BigEndian.PutUint16(header[6:8], crc)

	return header
}

// parseHeader 解析RPC消息头（带CRC校验）
func parseHeader(header []byte) (uint32, bool, error) {
	if len(header) != HEADER_LEN {
		return 0, false, errors.New("invalid header length")
	}

	// 验证CRC校验码
	dataPart := header[:6] // 现在包含协议版本
	expectedCRC := binary.BigEndian.Uint16(header[6:8])
	actualCRC := crc16(dataPart)

	if actualCRC != expectedCRC {
		return 0, false, errors.New("header CRC check failed")
	}

	// 解析长度和压缩标志
	dataLength := binary.BigEndian.Uint32(header[:4])
	var isCompressed bool
	switch header[4] {
	case 0x00:
		isCompressed = false
	case 0x01:
		isCompressed = true
	default:
		return 0, false, errors.New("invalid compression flag")
	}

	// 检查协议版本
	if header[5] != PROTOCOL_VERSION {
		return 0, false, errors.New("unsupported protocol version")
	}

	return dataLength, isCompressed, nil
}

// crc16 CRC16-CCITT算法实现（多项式0x1021，初始值0xFFFF）
func crc16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// invalidHeaderResponse 预构建的无效头部响应消息
// 当收到无效的消息头时返回此响应
var invalidHeaderResponse = func() []byte {
	r := ResponsePacketImpl{
		StatusCode:  StatusGnetHeaderError,
		ContentType: serverx.CONTENT_TYPE_STRING,
	}
	data := r.Pack(false)
	header := buildRpcHeader(data, false)
	return append(header, data...)
}()

// pingResponse 预构建的ping响应消息
// 用于心跳检测的响应
var pingResponse = &ResponsePacketImpl{
	StatusCode: http.StatusOK,
}

// serverBusyResponse 预构建的服务器繁忙响应消息
// 当服务器内存使用超出限制时返回此响应
var serverBusyResponse = &ResponsePacketImpl{
	StatusCode:  http.StatusServiceUnavailable,
	ContentType: serverx.CONTENT_TYPE_STRING,
	Payload:     "server busy, memory usage exceeds limit, please try again later",
}

// OnTraffic 处理网络流量的核心方法
// 实现了消息的接收、解析和处理流程
func (s *IntranetServer) OnTraffic(c gnet.Conn) gnet.Action {
	for {
		// 检查缓冲区数据是否足够一个消息头
		if c.InboundBuffered() < HEADER_LEN {
			return gnet.None
		}

		// 读取消息头
		header, err := c.Peek(HEADER_LEN)
		if err != nil {
			logx.Error("Peek header error: ", err)
			return gnet.Close
		}

		// 解析消息头，获取消息体长度和压缩标志
		bodyLen, compressed, err := parseHeader(header)
		if err != nil {
			atomic.AddInt64(&s.errorCounter, 1)
			// 处理无效的消息头
			logx.Debug("Parse header error: ", err)
			if _, err = c.Write(invalidHeaderResponse); err != nil {
				logx.Error("Write invalid header response error: ", err)
				return gnet.Close
			}
			if err = c.Flush(); err != nil {
				logx.Error("Flush invalid header response error: ", err)
				return gnet.Close
			}
			return gnet.Close
		}
		fullLen := HEADER_LEN + int(bodyLen)

		// 检查消息大小是否超出限制
		if fullLen > maxBufferSize {
			logx.Error("Message size", fullLen, " exceeds limit")
			return gnet.Close
		}

		// 检查缓冲区数据是否足够完整消息
		if c.InboundBuffered() < fullLen {
			return gnet.None
		}

		// 读取完整消息
		msgBytes, err := c.Peek(fullLen)
		if err != nil {
			logx.Error("Peek message error: ", err)
			return gnet.Close
		}

		// 增加请求计数，并获取消息缓冲区
		atomic.AddInt64(&s.reqCounter, 1)
		bodyBuf, bufRelease := buffertool.GetBuffer(int(fullLen))
		// 复制消息到缓冲区
		copy(bodyBuf, msgBytes)

		// 异步处理消息
		// MAYDO: 添加内存使用限制检查
		// if !utils.MemoryRunout(s.maxMemoryUsage) {
		// } else {
		//     bufRelease()
		//     s.sendResponse(c, serverBusyResponse, false)
		// }
		go s.asyncProcess(c, bodyBuf, bufRelease, compressed)

		// 丢弃已处理的消息
		if _, err = c.Discard(fullLen); err != nil {
			logx.Error("Discard error: ", err)
			return gnet.Close
		}
	}
}

// asyncProcess 异步处理请求
// 负责解包、解密、路由处理和响应发送的完整流程
func (s *IntranetServer) asyncProcess(c gnet.Conn, msg []byte, bufRelease func(), compressed bool) {
	defer func() {
		if r := recover(); r != nil {
			atomic.AddInt64(&s.errorCounter, 1)
			logx.Error(fmt.Sprintf("Process panic: %v\n%s", r, debug.Stack()))
			s.sendErrorResponse(c, http.StatusInternalServerError, "server error", compressed)
		}
		bufRelease() // 释放缓冲区资源，此处导致底层的字符串内存会回收至缓冲池，所以Body是临时数据
	}()

	// 解包请求
	req, err := UnPackRequest(msg[HEADER_LEN:], compressed)
	if err != nil {
		logx.Debug(err)
		atomic.AddInt64(&s.errorCounter, 1)
		s.sendErrorResponse(c, http.StatusBadRequest, err.Error(), compressed)
		return
	}

	// 处理ping请求
	if req.Type() == serverx.CONTENT_TYPE_PING {
		s.sendResponse(c, pingResponse, false)
		return
	}

	// 解密请求数据
	decrypted, err := encryptx.Decrypt(
		fastconv.StringToBytes(req.TemporaryData()),
		s.intranetSecret,
		s.algorithm,
	)
	if err != nil {
		atomic.AddInt64(&s.errorCounter, 1)
		s.sendErrorResponse(c, http.StatusForbidden, "decryption failed", compressed)
		return
	}

	// 解包请求数据
	if pkg, ok := req.(*RequestPacketImpl); ok {
		pkg.Payload = fastconv.BytesToString(decrypted)
	} else {
		atomic.AddInt64(&s.errorCounter, 1)
		s.sendErrorResponse(c, http.StatusBadRequest, "invalid request implementation", compressed)
		return
	}
	// 调用路由函数处理请求
	resp := s.router(req, c, s.routerImpl)
	if resp == nil {
		ctx := NewRequestContext(c, req)
		if s.unHandleFunc != nil {
			s.unHandleFunc(ctx)
			resp = ctx.response
		}
		if resp == nil {
			resp = &ResponsePacketImpl{
				StatusCode:  http.StatusNotImplemented,
				ContentType: serverx.CONTENT_TYPE_STRING,
				Payload:     "unimplemented intranet request",
			}
		}
	}
	// 发送响应
	s.sendResponse(c, resp, compressed)
}

// sendResponse 发送响应
// 负责加密响应数据并异步写入连接
func (s *IntranetServer) sendResponse(c gnet.Conn, resp serverx.ResponsePacket, compressed bool) {
	if len(resp.TemporaryData()) != 0 {
		// 加密响应数据
		encrypted, err := encryptx.Encrypt(
			fastconv.StringToBytes(resp.TemporaryData()),
			s.intranetSecret,
			s.algorithm,
		)
		if err != nil {
			logx.Error("Encrypt response failed: ", err)
			return
		}
		if pkg, ok := resp.(*ResponsePacketImpl); ok {
			pkg.Payload = fastconv.BytesToString(encrypted)
		} else {
			logx.Debug("Invalid response implementation")
		}
	}

	// 打包响应数据
	var respData []byte = resp.Pack(compressed)

	// 构建响应头并发送响应
	header := buildRpcHeader(respData, compressed)
	fullData := append(header, respData...)

	if err := c.AsyncWrite(fullData, func(c gnet.Conn, err error) error {
		if err != nil {
			atomic.AddInt64(&s.errorCounter, 1)
			logx.Error("Async write failed: " + err.Error())
		}
		return nil
	}); err != nil {
		logx.Error("Queue async write failed: ", err)
	}
}

// sendErrorResponse 发送错误响应
// 封装错误信息为响应消息并发送
func (s *IntranetServer) sendErrorResponse(c gnet.Conn, status int, msg string, compressed bool) {
	resp := &ResponsePacketImpl{
		StatusCode:  status,
		ContentType: serverx.CONTENT_TYPE_STRING,
		Payload:     msg,
	}
	s.sendResponse(c, resp, compressed)
}
