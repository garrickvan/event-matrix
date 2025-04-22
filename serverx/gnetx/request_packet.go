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

// RequestPacketImpl 定义客户端请求的数据包结构
type RequestPacketImpl struct {
	PayloadType serverx.CONTENT_TYPE `json:"pt"` // 内容类型：PING、JSON或STRING
	XData       string               `json:"xd"` // 扩展数据，用于扩展协议
	Payload     string               `json:"pl"` // Payload 内容，请求的主体数据，存在buff池里的临时数据
	SourceIP    string               `json:"ip"` // SourceIP 客户端IP地址，不指定则根据gnet自动获取
	CallChain   string               `json:"cc"` // CallChain 调用链，上层注入的调用信息，防止内部接口的循环调用
	Timestamp   int64                `json:"ts"` // 时间戳，Unix毫秒时间戳
}

// Marshal 将RequestPacket序列化为二进制格式
func (r *RequestPacketImpl) Marshal() ([]byte, error) {
	// 检查负载长度是否超出限制
	if len(r.Payload) > math.MaxUint32 {
		return nil, errors.New("payload too long")
	}

	if r.Timestamp == 0 {
		r.Timestamp = time.Now().UnixMilli()
	}

	// 预先转换所有字段
	xDataBytes := fastconv.StringToBytes(r.XData)
	payloadBytes := fastconv.StringToBytes(r.Payload)
	sourceIPBytes := fastconv.StringToBytes(r.SourceIP)
	callChainBytes := fastconv.StringToBytes(r.CallChain)
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(r.Timestamp))

	// 计算各字段长度
	payloadTypeLen := 1
	xDataLen := len(xDataBytes)
	payloadLen := len(payloadBytes)
	sourceIPLen := len(sourceIPBytes)
	callChainLen := len(callChainBytes)
	timestampLen := 8

	// 计算总缓冲区大小
	totalLen := 24 + payloadTypeLen + xDataLen + payloadLen + sourceIPLen + callChainLen + timestampLen
	data := make([]byte, totalLen)

	// 填充Header部分（手动展开循环提升性能）
	binary.BigEndian.PutUint32(data[0:4], uint32(payloadTypeLen))
	binary.BigEndian.PutUint32(data[4:8], uint32(xDataLen))
	binary.BigEndian.PutUint32(data[8:12], uint32(payloadLen))
	binary.BigEndian.PutUint32(data[12:16], uint32(sourceIPLen))
	binary.BigEndian.PutUint32(data[16:20], uint32(callChainLen))
	binary.BigEndian.PutUint32(data[20:24], uint32(timestampLen))

	// 填充Data部分（直接操作内存避免多次拷贝）
	offset := 24
	data[offset] = byte(r.PayloadType)
	offset += payloadTypeLen
	copy(data[offset:], xDataBytes)
	offset += xDataLen
	copy(data[offset:], payloadBytes)
	offset += payloadLen
	copy(data[offset:], sourceIPBytes)
	offset += sourceIPLen
	copy(data[offset:], callChainBytes)
	offset += callChainLen
	binary.BigEndian.PutUint64(data[offset:], uint64(r.Timestamp))

	return data, nil
}

// Unmarshal 将二进制数据反序列化为RequestPacket
func (r *RequestPacketImpl) Unmarshal(data []byte) error {
	// 检查数据长度是否足够包含头部
	if len(data) < 24 {
		return errors.New("invalid data length")
	}

	header := data[:24]
	body := data[24:]
	lengths := make([]uint32, 6)

	// 解析头部长度信息
	for i := 0; i < 6; i++ {
		lengths[i] = binary.BigEndian.Uint32(header[i*4 : (i+1)*4])
	}

	// 解析各字段
	offset := 0
	totalLen := len(body)

	// PayloadType
	if lengths[0] != 1 || offset+1 > totalLen {
		return errors.New("invalid PayloadType length")
	}
	r.PayloadType = serverx.CONTENT_TYPE(body[0])
	offset += 1

	// XData
	if end := offset + int(lengths[1]); end > totalLen {
		return errors.New("invalid XData length")
	} else {
		r.XData = fastconv.BytesToString(body[offset:end])
		offset = end
	}

	// Payload
	if end := offset + int(lengths[2]); end > totalLen {
		return errors.New("invalid Payload length")
	} else {
		r.Payload = fastconv.BytesToString(body[offset:end])
		offset = end
	}

	// SourceIP
	if end := offset + int(lengths[3]); end > totalLen {
		return errors.New("invalid SourceIP length")
	} else {
		r.SourceIP = fastconv.BytesToString(body[offset:end])
		offset = end
	}

	// CallChain
	if end := offset + int(lengths[4]); end > totalLen {
		return errors.New("invalid CallChain length")
	} else {
		r.CallChain = fastconv.BytesToString(body[offset:end])
		offset = end
	}

	// Timestamp
	if lengths[5] != 8 || offset+8 > totalLen {
		return errors.New("invalid Timestamp length")
	}
	r.Timestamp = int64(binary.BigEndian.Uint64(body[offset : offset+8]))
	offset += 8

	return nil
}

// Pack 序列化请求数据包
func (p *RequestPacketImpl) Pack(compressed bool) []byte {
	// 填充时间戳
	p.Timestamp = time.Now().UnixMilli()
	data, err := p.Marshal()
	if err != nil {
		logx.Debug("pack request packet failed: ", err.Error())
		return nil
	}

	if compressed {
		// 压缩数据
		data = snappy.Encode(nil, data)
	}
	return data
}

// unPackRequest 反序列化请求数据包
func UnPackRequest(data []byte, compressed bool) (serverx.RequestPacket, error) {
	var packet RequestPacketImpl
	var err error

	if compressed { // 解压缩数据
		data, err = snappy.Decode(nil, data)
		if err != nil {
			return nil, fmt.Errorf("snappy decompress failed: %v", err)
		}
	}

	// 反序列化
	if err := packet.Unmarshal(data); err != nil {
		return nil, fmt.Errorf("json unmarshal failed: %v", err)
	}

	// 检查消息是否过期
	if time.Since(time.UnixMilli(packet.Timestamp)) > MESSAGE_SEND_TIMEOUT {
		return nil, errors.New("request packet expired")
	}
	return &packet, nil
}

// Type 返回请求包的内容类型
func (r *RequestPacketImpl) Type() serverx.CONTENT_TYPE {
	return r.PayloadType
}

// TemporaryData 返回请求包的临时数据
func (r *RequestPacketImpl) TemporaryData() string {
	return r.Payload
}

// IP 返回请求包的来源IP
func (r *RequestPacketImpl) IP() string {
	return r.SourceIP
}

// CallChains 返回请求包的调用链信息
func (r *RequestPacketImpl) CallChains() string {
	return r.CallChain
}

func (r *RequestPacketImpl) Extend() string {
	return r.XData
}

// CreateTime 返回请求包的时间戳
func (r *RequestPacketImpl) CreateTime() int64 {
	return r.Timestamp
}
