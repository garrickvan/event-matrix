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

// Package utils 的测试文件，包含对工具函数的单元测试
package utils

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/garrickvan/event-matrix/utils/encryptx"
	"github.com/garrickvan/event-matrix/utils/jsonx"
)

// TestStructToStr 测试将结构体转换为日志字符串的功能
// 验证 StructToLogStr 函数能正确处理带有标签的结构体
func TestStructToStr(t *testing.T) {
	type User struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	}
	user := User{
		Id:   1,
		Name: "张三",
	}
	str := StructToLogStr(user)
	println(str)
}

// TestEncrypts 测试AES加密和解密功能
// 使用预定义的密钥对文本进行加密和解密，验证整个过程的正确性
func TestEncrypts(t *testing.T) {
	// Predefined key (must be 16, 24, or 32 bytes long)
	key := "mysecretaeskey123" // 16 bytes key for AES-128

	plaintext := []byte("Hello, World!")
	fmt.Println("Original Text:", string(plaintext))

	ciphertext, err := encryptx.Encrypt(plaintext, key, "AES-192")
	if err != nil {
		fmt.Println("Error encrypting:", err)
		return
	}
	fmt.Println("Encrypted Text:", hex.EncodeToString(ciphertext))

	decrypted, err := encryptx.Decrypt(ciphertext, key, "AES-192")
	if err != nil {
		fmt.Println("Error decrypting:", err)
		return
	}
	fmt.Println("Decrypted Text:", string(decrypted))
}

// TestGenID 测试唯一ID生成功能
// 生成多个ID并打印，用于验证ID的唯一性和格式
func TestGenID(t *testing.T) {
	for i := 0; i < 20; i++ {
		id := GenID()
		println(id)
	}
}

// TestJson 测试JSON序列化功能
// 创建一个包含当前时间戳的map并序列化为JSON字符串
func TestJson(t *testing.T) {
	params := map[string]int64{
		"createdAt": GetNowMilli(),
	}
	res, _ := jsonx.MarshalToBytes(params)
	// println(err.Error())
	println(string(res))
}

// TestTimeStrToMilli 测试时间字符串转换为毫秒时间戳的功能
// 将格式化的时间字符串转换为毫秒级Unix时间戳
func TestTimeStrToMilli(t *testing.T) {
	timeStr := "2023-08-06 15:30:10.023"
	microseconds := TimeStrToUTCMilli(timeStr)
	println(microseconds)
}

// TestIpNumber 测试端口号验证功能
// 验证IsPortNumber函数能否正确识别有效的端口号
func TestIpNumber(t *testing.T) {
	println(IsPortNumber("10000"))
}

// TestGetIntFromJson 测试从JSON字符串中提取整数值的功能
// 使用点号分隔的路径从嵌套的JSON结构中提取整数值
func TestGetIntFromJson(t *testing.T) {
	jsonStr := `{"code": 0, "data": {"id": 123, "name": "张三"}}`
	id := jsonx.GetInt64FromJson(jsonStr, "data.id")
	println(id)
}
