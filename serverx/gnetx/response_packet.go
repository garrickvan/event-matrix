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

// Package gnetx 提供基于gnet的网络通信类型定义
package gnetx

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils/fastconv"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/golang/snappy"
)

// ResponsePacketImpl 定义服务器响应的消息结构
type ResponsePacketImpl struct {
	ContentType serverx.CONTENT_TYPE `json:"mt"` // 响应内容类型：PING、JSON或STRING
	StatusCode  int                  `json:"s"`  // 状态码，同HTTP协议
	Payload     string               `json:"p"`  // 响应内容，可能是JSON字符串或普通文本，存在buff池里的临时数据
	Timestamp   int64                `json:"ts"` // 时间戳，Unix毫秒时间戳
}

// Marshal 将ResponsePacketImpl序列化为二进制格式
func (rm *ResponsePacketImpl) Marshal() ([]byte, error) {
	// 检查数据长度是否超出限制
	if len(rm.Payload) > math.MaxUint32 {
		return nil, errors.New("data too long")
	}

	if rm.Timestamp == 0 {
		rm.Timestamp = time.Now().UnixMilli()
	}

	// 预先转换所有字段
	payloadBytes := fastconv.StringToBytes(rm.Payload)
	statusBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(statusBytes, uint32(rm.StatusCode))
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(rm.Timestamp))

	// 计算各字段长度
	statusLen := len(statusBytes)
	contentTypeLen := 1
	payloadLen := len(payloadBytes)
	timestampLen := len(timestampBytes)

	// 计算总缓冲区大小
	totalLen := 16 + statusLen + contentTypeLen + payloadLen + timestampLen
	data := make([]byte, totalLen)

	// 填充Header部分（手动展开循环提升性能）
	binary.BigEndian.PutUint32(data[0:4], uint32(statusLen))
	binary.BigEndian.PutUint32(data[4:8], uint32(contentTypeLen))
	binary.BigEndian.PutUint32(data[8:12], uint32(payloadLen))
	binary.BigEndian.PutUint32(data[12:16], uint32(timestampLen))

	// 填充Data部分（直接操作内存避免多次拷贝）
	offset := 16
	copy(data[offset:], statusBytes)
	offset += statusLen
	data[offset] = byte(rm.ContentType)
	offset += contentTypeLen
	copy(data[offset:], payloadBytes)
	offset += payloadLen
	copy(data[offset:], timestampBytes)

	return data, nil
}

// Unmarshal 将二进制数据反序列化为ResponsePacketImpl
func (rm *ResponsePacketImpl) Unmarshal(data []byte) error {
	// 检查数据长度是否足够包含头部
	if len(data) < 16 {
		return errors.New("invalid data length")
	}

	header := data[:16]
	body := data[16:]
	lengths := make([]uint32, 4)

	// 解析头部长度信息
	for i := 0; i < 4; i++ {
		lengths[i] = binary.BigEndian.Uint32(header[i*4 : (i+1)*4])
	}

	// 解析各字段
	offset := 0
	totalLen := len(body)

	// Status
	if lengths[0] != 4 || offset+4 > totalLen {
		return errors.New("invalid Status length")
	}
	rm.StatusCode = int(binary.BigEndian.Uint32(body[:4]))
	offset += 4

	// MessageType
	if lengths[1] != 1 || offset+1 > totalLen {
		return errors.New("invalid MessageType length")
	}
	rm.ContentType = serverx.CONTENT_TYPE(body[offset])
	offset += 1

	// Payload
	if end := offset + int(lengths[2]); end > totalLen {
		return errors.New("invalid Payload length")
	} else {
		rm.Payload = fastconv.BytesToString(body[offset:end])
		offset = end
	}

	// Timestamp
	if lengths[3] != 8 || offset+8 > totalLen {
		return errors.New("invalid Timestamp length")
	}
	rm.Timestamp = int64(binary.BigEndian.Uint64(body[offset : offset+8]))
	offset += 8

	return nil
}

// Pack 序列化响应消息
func (p *ResponsePacketImpl) Pack(compressed bool) []byte {
	// 填充时间戳
	p.Timestamp = time.Now().UnixMilli()
	data, err := p.Marshal()
	if err != nil {
		logx.Debug("pack response message failed: ", err.Error())
		return nil
	}

	if compressed {
		// 压缩数据
		data = snappy.Encode(nil, data)
	}
	return data
}

// UnPackResponse 反序列化响应消息
func UnPackResponse(data []byte, compressed bool) (serverx.ResponsePacket, error) {
	var pkg ResponsePacketImpl
	var err error

	if compressed {
		// 解压缩数据
		data, err = snappy.Decode(nil, data)
		if err != nil {
			return nil, fmt.Errorf("snappy decompress failed: %v", err)
		}
	}

	// 反序列化
	if err := pkg.Unmarshal(data); err != nil {
		return nil, fmt.Errorf("json unmarshal failed: %v", err)
	}

	// 检查消息是否过期
	if time.Since(time.UnixMilli(pkg.Timestamp)) > MESSAGE_SEND_TIMEOUT {
		return nil, errors.New("response message expired")
	}
	return &pkg, nil
}

// Status 获取响应包的状态码
func (p *ResponsePacketImpl) Status() int {
	return p.StatusCode
}

// Type 获取请求包的内容类型
func (p *ResponsePacketImpl) Type() serverx.CONTENT_TYPE {
	return p.ContentType
}

// TemporaryData 获取请求包的临时数据，当前请求结束即回收
func (p *ResponsePacketImpl) TemporaryData() string {
	return p.Payload
}

// CreateTime 获取请求包的时间戳
func (p *ResponsePacketImpl) CreateTime() int64 {
	return p.Timestamp
}
