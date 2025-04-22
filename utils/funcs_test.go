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

package utils

import (
	"testing"
	"unicode"
)

func TestGenRandStrWithTimeSalt(t *testing.T) {
	for i := 0; i < 100; i++ {
		size := 24
		result := genRandStrWithTimeSalt(size, true)

		// 检查长度
		if len(result) != size {
			t.Errorf("genRandStrWithTimeSalt: expected length %d, got %d", size, len(result))
		}

		// 检查是否包含时间戳
		if len(result) < 10 {
			t.Error("genRandStrWithTimeSalt: result too short to contain timestamp")
		}
	}
}

func TestGenRandString(t *testing.T) {
	for i := 0; i < 100; i++ {
		length := 16
		result := genRandString(length, true)
		println(result)

		// 检查长度
		if len(result) != length {
			t.Errorf("genRandString: expected length %d, got %d", length, len(result))
		}

		// 检查字符集（区分大小写）
		for _, r := range result {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				t.Errorf("genRandString: invalid character %c in result", r)
			}
		}

		// 检查大小写敏感性
		resultLower := genRandString(length, false)
		for _, r := range resultLower {
			if unicode.IsUpper(r) {
				t.Errorf("genRandString: found uppercase character %c in case-insensitive mode", r)
			}
		}
	}
}

func TestExtractIp(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"纯IP地址", "192.168.1.1", "192.168.1.1"},
		{"带端口", "192.168.1.1:8080", "192.168.1.1"},
		{"带http协议", "http://192.168.1.1", "192.168.1.1"},
		{"带https协议", "https://192.168.1.1", "192.168.1.1"},
		{"带协议和端口", "http://192.168.1.1:8080", "192.168.1.1"},
		{"无效IP", "256.256.256.256", ""},
		{"域名", "example.com", ""},
		{"带端口的域名", "example.com:8080", ""},
		{"空字符串", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractIp(tt.input); got != tt.expected {
				t.Errorf("ExtractIp() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     bool
	}{
		// 有效IP地址
		{"Valid IP with port", "192.168.1.1:8080", true},
		{"Valid IP without port", "192.168.1.1", true},
		{"Localhost IP", "127.0.0.1", true},
		{"Localhost IP with port", "127.0.0.1:3000", true},

		// 有效域名
		{"Valid domain with port", "example.com:8080", true},
		{"Valid domain without port", "example.com", true},
		{"Subdomain", "api.example.com", true},
		{"Localhost", "localhost", true},
		{"Localhost with port", "localhost:8080", true},

		// 无效情况
		{"With http protocol", "http://example.com", false},
		{"With https protocol", "https://192.168.1.1:8080", false},
		{"Invalid domain", "invalid..domain", false},
		{"Empty string", "", false},
		{"Only port", ":8080", false},
		{"Invalid characters", "example$%.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEndpoint(tt.endpoint); got != tt.want {
				t.Errorf("IsEndpoint(%q) = %v, want %v", tt.endpoint, got, tt.want)
			}
		})
	}
}
