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

// Package logx 提供了日志系统的核心功能，包括日志轮转和写入操作
package logx

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// RotatingWriter 实现了一个自动轮转的日志写入器
// 支持按时间间隔自动切换日志文件，确保日志文件大小可控
type RotatingWriter struct {
	mu         sync.RWMutex  // 用于保护文件操作的互斥锁
	file       *os.File      // 当前正在写入的日志文件
	baseDir    string        // 日志文件存储的基础目录
	filePrefix string        // 日志文件名前缀
	logSuffix  string        // 日志文件后缀
	interval   time.Duration // 日志轮转的时间间隔
	ticker     *time.Ticker  // 用于触发日志轮转的定时器
}

// NewRotatingWriter 创建一个新的轮转日志写入器
// 参数：
//   - baseDir: 日志文件存储的基础目录
//   - filePrefix: 日志文件名前缀
//   - logSuffix: 日志文件后缀
//   - interval: 日志轮转的时间间隔（最小5秒）
//
// 返回：配置完成的日志写入器实例
func NewRotatingWriter(baseDir, filePrefix, logSuffix string, interval time.Duration) *RotatingWriter {
	if interval <= 5*time.Second {
		interval = 5 * time.Second
	}
	rw := &RotatingWriter{
		baseDir:    baseDir,
		interval:   interval,
		filePrefix: filePrefix,
		logSuffix:  logSuffix,
	}
	rw.rotate()
	go rw.startRotation()
	return rw
}

// Write 实现了io.Writer接口，将日志内容写入当前活动的日志文件
// 参数：
//   - p: 要写入的字节切片
//
// 返回：
//   - n: 写入的字节数
//   - err: 写入过程中的错误，如果有的话
func (rw *RotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.RLock()
	defer rw.mu.RUnlock()
	if rw.file == nil {
		return 0, errors.New("log file not initialized")
	}
	return rw.file.Write(p)
}

// startRotation 启动日志轮转的后台协程
// 按照配置的时间间隔定期触发日志文件的轮转
func (rw *RotatingWriter) startRotation() {
	rw.ticker = time.NewTicker(rw.interval)
	defer rw.ticker.Stop()

	for range rw.ticker.C {
		rw.rotate()
	}
}

// stop 停止日志轮转
// 停止定时器，不再触发新的日志轮转
func (rw *RotatingWriter) stop() {
	rw.ticker.Stop()
}

// rotate 执行日志文件的轮转操作
// 关闭当前日志文件，并创建一个新的日志文件
// 新文件名格式：{baseDir}/{prefix}.{timestamp}.{suffix}
func (rw *RotatingWriter) rotate() {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		_ = rw.file.Sync()
		_ = rw.file.Close()
	}

	timestamp := time.Now().Format("20060102_150405")
	newFileName := fmt.Sprintf("%s/%s.%s.%s", rw.baseDir, rw.filePrefix, timestamp, rw.logSuffix)
	file, err := os.OpenFile(newFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		Error("Failed to rotate log file: %v", err)
		return
	}

	rw.file = file
}
