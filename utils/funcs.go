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

// Package utils 提供了一组通用的工具函数，包括字符串处理、ID生成、文件操作、
// 数据验证等功能，用于支持事件矩阵系统的各个组件。
package utils

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/oklog/ulid/v2"
	"github.com/shirou/gopsutil/mem"
)

var (
	// randPool 是一个随机数生成器池，用于提高随机数生成的性能
	randPool = sync.Pool{
		New: func() interface{} {
			return rand.New(rand.NewSource(time.Now().UnixNano()))
		},
	}
	// charsetUpper 包含大小写字母和数字的字符集
	charsetUpper = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// charsetLower 仅包含小写字母和数字的字符集
	charsetLower = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// MakeDir 创建指定目录，如果目录不存在则创建。
// 使用os.ModePerm权限创建目录及其所有必需的父目录。
// dirName: 要创建的目录路径
func MakeDir(dirName string) {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		os.MkdirAll(dirName, os.ModePerm)
	}
}

// IsEmail 验证字符串是否为有效的电子邮箱格式。
// 使用正则表达式验证邮箱格式，支持常见格式：
// - 本地部分可包含字母、数字、下划线和点号
// - 域名部分必须符合标准域名格式
// email: 要验证的邮箱字符串
// 返回: 如果是有效邮箱返回true，否则返回false
func IsEmail(email string) bool {
	pattern := `\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*` //匹配电子邮箱
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(email)
}

// IsPhoneNumber 验证全球手机号格式，支持中国和其他国家手机号。
// 如果没有+号前缀，默认视为中国手机号并自动添加+86前缀。
// 使用E.164格式验证：+{国家代码}{本地号码}
// phoneNumber: 要验证的手机号字符串
// 返回: 如果是有效手机号返回true，否则返回false
func IsPhoneNumber(phoneNumber string) bool {
	// 如果没有 + 号，默认视为中国手机号，补充 +86
	if !strings.HasPrefix(phoneNumber, "+") {
		phoneNumber = "+86" + phoneNumber
	}
	// 使用正则表达式验证手机号格式
	// E.164 格式：+{国家代码}{本地号码}
	phoneNumberRegex := `^(\+)[1-9]\d{0,3}\d{3,15}$`
	match, err := regexp.MatchString(phoneNumberRegex, phoneNumber)
	if err != nil {
		// 处理正则表达式匹配错误
		fmt.Println("Regex error:", err)
		return false
	}
	return match
}

// GenID 生成一个唯一标识符，基于 ULID (Universally Unique Lexicographically Sortable Identifier)。
// 生成的ID具有以下特性：
// - 按时间排序
// - 保证唯一性
// - 使用大写字母和数字
// 如果ULID生成失败，将回退到使用时间戳加随机字符串的方式生成ID。
func GenID() string {
	entropy := rand.New(rand.NewSource(time.Now().UnixNano()))
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)
	if err != nil {
		fmt.Println("ULID error:", err.Error())
		return Rand36BaseStrByTimeSalt(36)
	}
	return id.String()
}

// GenTUUID 生成一个带有时间戳前缀的UUID。
// 格式为：{36进制时间戳}-{标准UUID}
// prefix: 可选的前缀字符串
// 返回: 生成的UUID字符串
func GenTUUID(prefix string) string {
	u, err := uuid.NewRandom()
	if err != nil {
		fmt.Println("UUID error:", err.Error())
		return Rand36BaseStrByTimeSalt(36)
	}
	return get36HexTimeStamp() + "-" + u.String()
}

// get36HexTimeStamp 获取当前时间戳的36进制表示，精确到毫秒。
// 将毫秒级时间戳转换为36进制字符串，可以得到更短的字符串表示。
// 使用0-9和a-z作为36进制的字符集。
func get36HexTimeStamp() string {
	now := time.Now()
	milliseconds := now.UnixNano() / int64(time.Millisecond)
	// 定义36进制的字符集
	charset := "0123456789abcdefghijklmnopqrstuvwxyz"
	var result string
	base := int64(len(charset))
	for milliseconds > 0 {
		remainder := milliseconds % base
		result = string(charset[remainder]) + result
		milliseconds /= base
	}
	return result
}

// Rand36BaseStrByTimeSalt 生成指定长度的随机字符串，使用36进制字符集（小写字母和数字）。
// 字符串由时间戳和随机字符组成，确保唯一性。
// 参数：
//   - size: 生成的字符串长度
func Rand36BaseStrByTimeSalt(size int) string {
	return genRandStrWithTimeSalt(size, false)
}

// Rand62BaseStrByTimeSalt 生成指定长度的随机字符串，使用62进制字符集（大小写字母和数字）。
// 字符串由时间戳和随机字符组成，确保唯一性。
// 参数：
//   - size: 生成的字符串长度
func Rand62BaseStrByTimeSalt(size int) string {
	return genRandStrWithTimeSalt(size, true)
}

// Rand36BaseStr 生成指定长度的纯随机字符串，使用36进制字符集（小写字母和数字）。
// 参数：
//   - size: 生成的字符串长度
func Rand36BaseStr(size int) string {
	return genRandString(size, false)
}

// Rand62BaseStr 生成指定长度的纯随机字符串，使用62进制字符集（大小写字母和数字）。
// 参数：
//   - size: 生成的字符串长度
func Rand62BaseStr(size int) string {
	return genRandString(size, true)
}

func genRandStrWithTimeSalt(size int, caseSensitive bool) string {
	if size == 0 {
		size = 24
	}

	timeSalt := strconv.FormatInt(time.Now().UnixNano(), 36)
	randLen := size - len(timeSalt)
	if randLen <= 0 {
		return timeSalt[:size]
	}

	result := make([]byte, size)
	copy(result, timeSalt)
	copy(result[len(timeSalt):], genRandString(randLen, caseSensitive))

	return string(result)
}

func genRandString(length int, caseSensitive bool) string {
	charset := charsetUpper
	if !caseSensitive {
		charset = charsetLower
	}

	r := randPool.Get().(*rand.Rand)
	defer randPool.Put(r)

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[r.Intn(len(charset))]
	}
	return string(result)
}

// GenMarkFromStr 根据输入字符串生成固定长度的标识符。
// 使用MD5算法对输入字符串进行哈希，并截取指定长度的前缀作为标识。
// 参数：
//   - str: 输入字符串
//   - len: 需要的标识符长度
//
// 返回值：大写的标识符字符串
func GenMarkFromStr(str string, len int) string {
	hashByte := md5.Sum([]byte(str))
	hashStr := hex.EncodeToString(hashByte[:])
	return strings.ToUpper(hashStr)[:len]
}

// GetSha1FromStr 计算字符串的SHA1哈希值。
// 返回值为十六进制编码的字符串。
// 参数：
//   - str: 需要计算哈希的字符串
func GetSha1FromStr(str string) string {
	h := sha1.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// StrContains 检查字符串数组中是否包含指定的子字符串。
// 参数：
//   - strs: 字符串数组
//   - substr: 要查找的子字符串
//
// 返回值：如果找到返回true，否则返回false
func StrContains(strs []string, substr string) bool {
	for _, str := range strs {
		if str == substr {
			return true
		}
	}
	return false
}

// CountCodeSize 根据总记录数计算合适的分享码长度。
// 使用36进制（0-9和a-z）计算，并取每个数量级容量的十分之一，以降低冲突概率。
// 参数：
//   - total: 总记录数
//
// 返回值：建议的分享码长度
// 容量对照：
//   - 4位: 支持到20万条记录
//   - 5位: 支持到500万条记录
//   - 6位: 支持到1亿条记录
//   - 以此类推...
func CountCodeSize(total int64) int {
	if total < 200000 { // 6位 1679616
		return 4
	} else if total < 5000000 { // 7位 60466176
		return 5
	} else if total < 100000000 { // 9位 2176782336
		return 6
	} else if total < 5000000000 { // 10位 78364164096
		return 7
	} else if total < 100000000000 { // 12位 2821109907456
		return 8
	} else if total < 10000000000000 { // 14位 101559956668416
		return 9
	} else if total < 100000000000000 { // 16位 3656158440062976
		return 10
	} else if total < 1000000000000000 { // 17位 1.316217038422671e17
		return 11
	} else if total < 40000000000000000 { // 18位 4.738381338321617e18
		return 12
	}
	return 16
}

// GetSyncMapLength 获取sync.Map的元素数量。
// 由于sync.Map没有直接提供获取长度的方法，这个函数通过遍历来计算元素数量。
// 参数：
//   - m: 同步Map对象指针
//
// 返回值：Map中的元素数量
func GetSyncMapLength(m *sync.Map) int {
	length := 0
	m.Range(func(key, value interface{}) bool {
		length++
		return true
	})
	return length
}

// IsPortNumber 检查字符串是否为有效的端口号。
// 有效端口号范围为1-65535。
// 参数：
//   - s: 要检查的字符串
//
// 返回值：如果是有效的端口号返回true，否则返回false
func IsPortNumber(s string) bool {
	num, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	if num >= 1 && num <= 65535 {
		return true
	}
	return false
}

// IsEqualsStrArray 比较两个字符串数组是否包含相同的元素（忽略顺序）。
// 使用map来统计元素出现次数，确保两个数组包含相同的元素且出现次数一致。
// 参数：
//   - slice1: 第一个字符串数组
//   - slice2: 第二个字符串数组
//
// 返回值：如果两个数组包含相同元素返回true，否则返回false
func IsEqualsStrArray(slice1, slice2 []string) bool {
	if len(slice1) != len(slice2) {
		return false
	}
	counts := make(map[string]int)
	for _, s := range slice1 {
		counts[s]++
	}
	for _, s := range slice2 {
		if counts[s] == 0 {
			return false
		}
		counts[s]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

// StructToLogStr 将结构体转换为格式化的字符串键值对，用于日志输出。
// 使用反射获取结构体字段名和值，生成 "字段名: 值" 格式的字符串。
// 支持普通结构体、结构体指针和map类型。
// 参数：
//   - obj: 要转换的结构体、结构体指针或map
//
// 返回值：格式化后的字符串，字段之间使用 " --|-- " 分隔
func StructToLogStr(obj interface{}) string {
	var sb strings.Builder
	// 使用反射获取结构体的值
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Map {
		return fmt.Sprintf("%v", obj)
	}
	// 如果是指针类型，则获取指针指向的值
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	// 检查是否为结构体
	if v.Kind() != reflect.Struct {
		return ""
	}
	t := v.Type()
	// 遍历结构体的字段
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		sb.WriteString(field.Name)
		sb.WriteString(": ")
		sb.WriteString(fmt.Sprintf("%v", value.Interface()))
		if i < v.NumField()-1 {
			sb.WriteString(" --|-- ")
		}
	}
	return sb.String()
}

func InStrArray(str string, arr []string) bool {
	for _, s := range arr {
		if s == str {
			return true
		}
	}
	return false
}

func InIntArray(num int64, arr []int64) bool {
	for _, n := range arr {
		if n == num {
			return true
		}
	}
	return false
}

func InFloat64Array(num float64, arr []float64) bool {
	for _, n := range arr {
		if n == num {
			return true
		}
	}
	return false
}

// ExtractIp 从endpoint字符串中提取IP地址。
// 支持多种格式：
//   - 纯IP地址：192.168.1.1
//   - 带端口的IP地址：192.168.1.1:8080
//   - 带协议的IP地址：http://192.168.1.1
//   - 带协议和端口的IP地址：http://192.168.1.1:8080
//
// 返回值：
//   - 成功：返回提取的IP地址
//   - 失败：返回空字符串
//
// 支持格式：
// 1. 纯IP地址：192.168.1.1
// 2. 带端口的IP地址：192.168.1.1:8080
// 3. 带协议的IP地址：http://192.168.1.1
// 4. 带协议和端口的IP地址：http://192.168.1.1:8080
func ExtractIp(endpoint string) string {
	// 去除协议部分
	if strings.Contains(endpoint, "://") {
		parts := strings.Split(endpoint, "://")
		if len(parts) > 1 {
			endpoint = parts[1]
		}
	}

	// 去除端口部分
	if strings.Contains(endpoint, ":") {
		s := strings.Split(endpoint, ":")
		if len(s) > 0 {
			endpoint = s[0]
		}
	}

	// 验证是否是合法IP地址
	ipPattern := `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	if matched, _ := regexp.MatchString(ipPattern, endpoint); matched {
		return endpoint
	}

	return ""
}

// IsUserName 检查用户名是否符合规范。
// 规则：
//   - 长度在2-20个字符之间
//   - 只能包含小写字母、数字和下划线
//
// 参数：
//   - username: 要检查的用户名
//
// 返回值：如果符合规范返回true，否则返回false
func IsUserName(username string) bool {
	return len(username) >= 2 && len(username) <= 20 && strings.ContainsAny(username, "abcdefghijklmnopqrstuvwxyz0123456789_")
}

func IsEndpoint(endpoint string) bool {
	// 检查是否包含协议
	if strings.Contains(endpoint, "://") {
		return false
	}

	// 分割地址和端口
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		// 如果没有端口，直接验证是否为有效域名或IP
		host = endpoint
	} else {
		// 如果有端口，验证端口是否有效
		if _, err := strconv.Atoi(port); err != nil {
			return false
		}
	}

	// 验证是否为有效IP地址
	if ip := net.ParseIP(host); ip != nil {
		return true
	}

	// 验证是否为有效域名
	// 先进行格式验证
	if !isValidDomain(host) {
		return false
	}
	// 再进行DNS解析验证
	if _, err := net.LookupHost(host); err != nil {
		return false
	}

	return true
}

// 新增辅助函数验证域名格式
func isValidDomain(domain string) bool {
	// 域名长度限制
	if len(domain) < 1 || len(domain) > 253 {
		return false
	}

	// 检查每个标签
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		// 每个标签长度限制
		if len(label) < 1 || len(label) > 63 {
			return false
		}
		// 标签只能包含字母、数字和连字符
		for _, r := range label {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-') {
				return false
			}
		}
		// 标签不能以连字符开头或结尾
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}

	return true
}

// MemoryRunout 检查系统内存使用是否达到指定阈值。
// 参数：
//   - maxMemoryUsage: 内存使用百分比阈值（0-100）
//
// 返回值：如果内存使用超过阈值返回true，否则返回false
func MemoryRunout(maxMemoryUsage int) bool {
	v, _ := mem.VirtualMemory()
	return v.UsedPercent >= float64(maxMemoryUsage)
}

// GetEnv 从环境变量中获取指定键的值，支持多级配置加载。
// 加载顺序：
//  1. 首先尝试从当前目录的 .env 文件加载配置
//  2. 如果未找到，则尝试从 init.env 文件加载配置
//  3. 最后从系统环境变量中获取
//
// 参数：
//   - key: 要获取的环境变量键名
//
// 返回值：
//   - 如果找到对应值则返回字符串值，否则返回空字符串
func GetEnv(key string) string {
	result := ""
	// 从配置文件中读取配置
	err := godotenv.Overload(".env")
	if err == nil {
		result = os.Getenv(key)
	}
	if result != "" {
		return result
	}
	// 从init.env中读取配置
	err = godotenv.Overload("init.env")
	if err == nil {
		result = os.Getenv(key)
	}
	result = os.Getenv(key)
	if result != "" {
		return result
	}
	return result
}
