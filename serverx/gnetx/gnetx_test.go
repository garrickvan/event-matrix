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

package gnetx

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/garrickvan/event-matrix/serverx"
	"github.com/garrickvan/event-matrix/utils/jsonx"
)

// 测试数据
var testPacket = &RequestPacketImpl{
	PayloadType: serverx.CONTENT_TYPE_JSON,
	Payload:     `{"key":"vadsfsfasfasdfasfasdsxsdsfasfasdfsaaaaaaaaaaaaaaaaaaaaaaaaaadddddddddddddlue"}`,
	SourceIP:    "127.0.0.1",
	CallChain:   "test,test2",
}

// 基准测试：UnPackRequestPacket
func BenchmarkUnPackRequestPacket(b *testing.B) {
	data := testPacket.Pack(false)
	for i := 0; i < b.N; i++ {
		_, err := UnPackRequest(data, false)
		if err != nil {
			b.Fatalf("UnPackRequestPacket failed: %v", err)
		}
	}
}

// 基准测试：Pack
func BenchmarkPack(b *testing.B) {
	testPacket.Pack(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = testPacket.Pack(false)
	}
}

// 基准测试：UnPackRequestCompressed
func BenchmarkUnPackRequestCompressed(b *testing.B) {
	data := testPacket.Pack(true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := UnPackRequest(data, true)
		if err != nil {
			b.Fatalf("UnPackRequestCompressed failed: %v", err)
		}
	}
}

// 基准测试：PackCompressed
func BenchmarkPackCompressed(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = testPacket.Pack(true)
	}
}

func TestDataSizeComparison(t *testing.T) {
	// 测试数据
	packet := &RequestPacketImpl{
		PayloadType: serverx.CONTENT_TYPE_JSON,
		Payload:     `{"key":"vadsfsfasfasdfasfasdsxsdsfasfasdfsaaaaaaaaaaaaaaaaaaaaaaaaaadddddddddddddlue"}`,
		SourceIP:    "127.0.0.1",
		CallChain:   "test,test2",
	}

	// 非压缩数据
	nonCompressedData := packet.Pack(false)
	nonCompressedSize := len(nonCompressedData)

	// 压缩数据
	compressedData := packet.Pack(true)
	compressedSize := len(compressedData)

	// 输出结果
	t.Logf("Non-compressed data size: %d bytes", nonCompressedSize)
	t.Logf("Compressed data size: %d bytes", compressedSize)
	t.Logf("Compression ratio: %.2f%%", float64(compressedSize)/float64(nonCompressedSize)*100)
}

/**
goos: darwin
goarch: amd64
pkg: github.com/garrickvan/event-matrix/serverx
cpu: Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz
BenchmarkUnPackRequestPacket-8                   2457898               487.0 ns/op           176 B/op          4 allocs/op
BenchmarkUnPackRequestPacket-8                   2413492               483.7 ns/op           176 B/op          4 allocs/op
BenchmarkUnPackRequestPacket-8                   2455898               496.7 ns/op           176 B/op          4 allocs/op
BenchmarkUnPackRequestPacket-8                   2475040               500.9 ns/op           176 B/op          4 allocs/op
BenchmarkUnPackRequestPacket-8                   2479938               481.9 ns/op           176 B/op          4 allocs/op
BenchmarkPack-8                                  4078100               295.8 ns/op           104 B/op          2 allocs/op
BenchmarkPack-8                                  4070109               300.7 ns/op           104 B/op          2 allocs/op
BenchmarkPack-8                                  3884868               293.3 ns/op           104 B/op          2 allocs/op
BenchmarkPack-8                                  4007078               293.0 ns/op           104 B/op          2 allocs/op
BenchmarkPack-8                                  4093960               293.0 ns/op           104 B/op          2 allocs/op
BenchmarkUnPackRequestPacketCompressed-8         2135320               565.6 ns/op           256 B/op          5 allocs/op
BenchmarkUnPackRequestPacketCompressed-8         2132413               557.2 ns/op           256 B/op          5 allocs/op
BenchmarkUnPackRequestPacketCompressed-8         2085955               564.5 ns/op           256 B/op          5 allocs/op
BenchmarkUnPackRequestPacketCompressed-8         2116506               563.2 ns/op           256 B/op          5 allocs/op
BenchmarkUnPackRequestPacketCompressed-8         2111631               559.3 ns/op           256 B/op          5 allocs/op
BenchmarkPackCompressed-8                        2437315               492.1 ns/op           216 B/op          3 allocs/op
BenchmarkPackCompressed-8                        2385805               490.4 ns/op           216 B/op          3 allocs/op
BenchmarkPackCompressed-8                        2390061               489.7 ns/op           216 B/op          3 allocs/op
BenchmarkPackCompressed-8                        2422322               494.7 ns/op           216 B/op          3 allocs/op
BenchmarkPackCompressed-8                        2300032               495.8 ns/op           216 B/op          3 allocs/op
**/

func TestSend(t *testing.T) {
	// 建立TCP连接
	conn, err := net.Dial("tcp", "127.0.0.1:10001")
	if err != nil {
		fmt.Printf("Error connecting to server: %v", err)
	}
	defer conn.Close()

	// 准备测试数据
	req := RequestPacketImpl{
		PayloadType: serverx.CONTENT_TYPE_JSON,
		Payload:     `{"test":"value"}`,
		SourceIP:    "127.0.0.1",
		CallChain:   "test",
	}

	// 调用被测试函数
	resp, err := send(conn, &req, true, 5*time.Second)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// 验证响应
	if resp.Status() != 200 {
		t.Errorf("Expected status 200, got %d", resp.Status())
	}
	fmt.Println(resp.TemporaryData())
}

// 测试数据结构
type TestData struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func BenchmarkPostJSON(b *testing.B) {
	// 初始化客户端
	client := NewClient(100, 5*time.Minute, 30*time.Second)

	// 测试数据
	testData := TestData{
		Name:  "test",
		Value: 123,
	}

	// 重置计时器，排除初始化时间
	b.ResetTimer()

	// 运行基准测试
	for i := 0; i < b.N; i++ {
		// 使用mock endpoint，实际测试时需要替换为真实地址
		data, _ := jsonx.MarshalToBytes(testData)
		_, err := client.Post("localhost:10001", serverx.CONTENT_TYPE_JSON, data, "", []string{})
		if err != nil {
			b.Fatalf("PostJSON failed: %v", err)
		}
	}
}

// 结果如下：
// goos: darwin
// goarch: amd64
// pkg: github.com/garrickvan/event-matrix/serverx/rpcx
// cpu: Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz
// === RUN   BenchmarkPostJSON
// BenchmarkPostJSON
// BenchmarkPostJSON-8        19917             58336 ns/op             540 B/op         12 allocs/op
// PASS

func TestPostJSON(t *testing.T) {
	// 初始化客户端
	client := NewClient(10, 5*time.Minute, 30*time.Second)

	// 测试数据
	testData := TestData{
		Name:  "test",
		Value: 123,
	}

	// 测试用例
	tests := []struct {
		name     string
		endpoint string
		data     interface{}
		wantErr  bool
	}{
		{
			name:     "valid request",
			endpoint: "localhost:10001",
			data:     testData,
			wantErr:  false,
		},
		{
			name:     "invalid endpoint",
			endpoint: "invalid:address",
			data:     testData,
			wantErr:  true,
		},
		{
			name:     "nil data",
			endpoint: "localhost:10001",
			data:     nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client.SetCompress(true)
			data, _ := jsonx.MarshalToBytes(tt.data)
			response, err := client.Post(tt.endpoint, serverx.CONTENT_TYPE_JSON, data, "", []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("PostJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if response.Status() != 200 {
				t.Errorf("Expected status 200, got %d", response.Status())
			}
			if response.TemporaryData() != "ok" {
				t.Errorf("Expected response data 'ok', got %s", response.TemporaryData())
			}
		})
	}
}

func BenchmarkPing(b *testing.B) {
	// 初始化客户端
	client := NewClient(100, 5*time.Minute, 30*time.Second)
	port := "10001"

	// 预热
	_ = client.Ping("localhost:" + port)
	_ = client.Ping("localhost:" + port)

	// 重置计时器，排除初始化时间
	b.ResetTimer()

	// 运行基准测试
	for i := 0; i < b.N; i++ {
		err := client.Ping("localhost:" + port)
		if err != nil {
			b.Fatalf("PostJSON failed: %v", err)
		}
	}
}

func TestPingGateway(t *testing.T) {
	// 初始化客户端
	client := NewClient(10, 5*time.Minute, 30*time.Second)
	port := "10001"

	for i := 0; i < 15*10000; i++ {
		err := client.Ping("localhost:" + port)
		if err != nil {
			t.Errorf("Ping() error = %v", err)
			return
		}
	}

}

func TestPingWorker(t *testing.T) {
	// 初始化客户端
	client := NewClient(10, 5*time.Minute, 30*time.Second)
	port := "25001"

	for i := 0; i < 15*10000; i++ {
		err := client.Ping("localhost:" + port)
		if err != nil {
			t.Errorf("Ping() error = %v", err)
			return
		}
	}

}
