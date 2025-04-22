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

package logx

import (
	"fmt"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 测试
func TestLogger(t *testing.T) {
	type User struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	}
	user := User{
		Id:   1,
		Name: "张三",
	}
	InitRuntimeLogger("test_log", "debug", "", 10*time.Second)
	Log().Info("test2")
	Log().Debug("test3")
	SugarLog().Info(user)

	customLogger := NewLogger("test_log", string(LogTypeRuntime), "debug", "", 10*time.Second)
	customLogger.Log().Info("test")
	customLogger.Log().Debug("test")
}

func TestRotateLogger(t *testing.T) {
	// Create log directory
	baseDir := "test_log"
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create log directory: %v", err))
	}

	// Initialize rotating writer
	writer := NewRotatingWriter(baseDir, "app_log", "log", 5*time.Second)

	// Create zap logger
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:      "time",
		LevelKey:     "level",
		MessageKey:   "msg",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig), // Use JSON format
		zapcore.AddSync(writer),               // Use rotating writer
		zap.InfoLevel,                         // Log level
	)

	logger := zap.New(core, zap.AddCaller())
	defer logger.Sync()

	// Start logging 30 seconds
	counter := 0
	for {
		if counter >= 10 {
			break
		}
		logger.Info("This is a test log message", zap.String("context", "demo"))
		counter++
		time.Sleep(2 * time.Second)
	}
}
