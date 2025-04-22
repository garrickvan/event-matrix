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

package encryptx

import (
	"testing"
)

var (
	testKey       = "test-key-12345678"
	testPlaintext = []byte("This is a test plaintext for benchmarking")
)

func BenchmarkEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Encrypt(testPlaintext, testKey, "AES-256")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	// 先加密得到密文
	ciphertext, err := Encrypt(testPlaintext, testKey, "AES-256")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Decrypt(ciphertext, testKey, "AES-256")
		if err != nil {
			b.Fatal(err)
		}
	}
}

/**
goos: darwin
goarch: amd64
pkg: github.com/garrickvan/event-matrix/utils/encryptx
cpu: Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz
=== RUN   BenchmarkDecrypt
BenchmarkDecrypt
BenchmarkDecrypt-8       1518421               799.2 ns/op           656 B/op          5 allocs/op
**/
