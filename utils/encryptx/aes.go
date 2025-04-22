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
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"strings"

	"github.com/garrickvan/event-matrix/utils/fastconv"
)

// generateHash 根据给定的密钥和AES算法生成哈希值
func generateHash(key []byte, aesAlgor string) []byte {
	if len(key) == 0 {
		return []byte{}
	}
	aesAlgor = strings.ToUpper(aesAlgor)
	switch aesAlgor {
	case "AES-128", "AES128", "AES", "AES-128-CBC", "AES128CBC":
		hasher := md5.New()
		hasher.Write(key)
		return hasher.Sum(nil)
	case "AES-192", "AES192", "AES-192-CBC", "AES192CBC":
		hasher := sha256.New224()
		hasher.Write(key)
		return hasher.Sum(nil)[:24]
	case "AES-256", "AES256", "AES-256-CBC", "AES256CBC":
		hasher := sha256.New()
		hasher.Write(key)
		return hasher.Sum(nil)
	case "NONE", "", "NULL", "NULL-CBC":
		return []byte{}
	default:
		hasher := md5.New()
		hasher.Write(key)
		return hasher.Sum(nil)
	}
}

// Encrypt 使用给定的密钥和AES算法加密明文，返回密文
func Encrypt(plaintext []byte, key, aesAlgor string) ([]byte, error) {
	keyBytes := generateHash(fastconv.StringToBytes(key), aesAlgor)
	// If key is empty or aesAlgor is "NONE", return plaintext as is.
	if len(keyBytes) == 0 {
		return plaintext, nil
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}
	padded := make([]byte, aes.BlockSize+len(plaintext))
	iv := padded[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(padded[aes.BlockSize:], plaintext)
	return padded, nil
}

// Decrypt 使用给定的密钥和AES算法解密密文，返回明文
func Decrypt(ciphertext []byte, key, aesAlgor string) ([]byte, error) {
	keyBytes := generateHash(fastconv.StringToBytes(key), aesAlgor)
	if len(keyBytes) == 0 {
		return ciphertext, nil
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short, must be at least 16 bytes")
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}
