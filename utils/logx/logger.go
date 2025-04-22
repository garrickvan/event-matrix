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
	"strings"
	"time"

	"github.com/garrickvan/event-matrix/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 日志类型和后缀常量定义
const (
	LogTypeEvent   = "event"     // 事件日志类型
	LogTypeRuntime = "runtime"   // 运行时日志类型
	LogSuffix      = "slice_log" // 日志文件切片后缀
)

// Logger 是一个日志记录器结构体，封装了 zap.Logger 并添加了一些自定义功能
type Logger struct {
	logger   *zap.Logger
	baseDir  string
	logLevel zapcore.Level
	logType  string
	serverId string
	writer   *RotatingWriter
}

// LogEntry 表示一条日志记录的结构
// 用于存储日志的基本信息，支持JSON序列化和数据库存储
type LogEntry struct {
	ID        string `json:"id" gorm:"index"`    // 日志唯一标识
	Level     string `json:"level" gorm:"index"` // 日志级别
	Caller    string `json:"caller"`             // 调用者信息
	Msg       string `json:"msg"`                // 日志消息内容
	CreatedAt int64  `json:"createdAt"`          // 创建时间戳
	Creator   string `json:"creator"`            // 创建者标识
}

// 全局日志记录器实例
var (
	runtimeLogger *Logger // 全局运行时日志记录器，用于记录系统运行时信息
	EventLogger   *Logger // 全局事件日志记录器，用于记录业务事件信息
)

// NewLogger 创建一个新的日志记录器实例
// 参数：
//   - baseDir: 日志文件存储的基础目录
//   - logType: 日志类型（event/runtime）
//   - logLevel: 日志级别（debug/info/warn/error/fatal）
//   - serverId: 服务实例的唯一标识
//   - slicePeriod: 日志文件切割的时间周期
//
// 返回：
//   - *Logger: 配置完成的日志记录器实例
func NewLogger(baseDir, logType, logLevel, serverId string, slicePeriod time.Duration) *Logger {
	if baseDir == "" {
		baseDir = "log"
	}
	utils.MakeDir(baseDir)
	// 日志切割
	writer := NewRotatingWriter(baseDir, logType, LogSuffix, slicePeriod)
	if writer == nil {
		panic("日志切割组件初始化失败")
	}
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:      "createdAt",
		LevelKey:     "level",
		NameKey:      "logger",
		CallerKey:    "caller",
		MessageKey:   "msg",
		LineEnding:   zapcore.DefaultLineEnding,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeTime:   utcTimeEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
	level := getLevelFromStr(logLevel)
	var writeSyncer zapcore.WriteSyncer
	// 开发模式下同时输出到控制台
	if level == zap.DebugLevel {
		writeSyncer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(writer), zapcore.AddSync(os.Stderr))
	} else {
		writeSyncer = zapcore.AddSync(writer)
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		level,
	)
	// 开启开发模式，堆栈跟踪
	caller := zap.AddCaller()
	// 构造日志
	logger := zap.New(core, caller)
	defer logger.Sync()
	return &Logger{
		logger:   logger,
		baseDir:  baseDir,
		logLevel: level,
		logType:  logType,
		serverId: serverId,
		writer:   writer,
	}
}

// utcTimeEncoder 自定义时间编码器
// 将时间转换为UTC时间并以毫秒级时间戳形式编码
// 用于确保日志时间戳的一致性和可比较性
func utcTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendInt64(t.UTC().UnixNano() / int64(time.Millisecond))
}

// getLevelFromStr 将字符串形式的日志级别转换为zap的日志级别枚举
// 支持debug、info、warn/warning、error、fatal五个级别
// 默认返回warn级别
func getLevelFromStr(level string) zapcore.Level {
	level = strings.ToLower(level)
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn", "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.WarnLevel
	}
}

// shutdown 关闭日志记录器，确保所有缓冲的日志都被写入磁盘并停止日志切割
func (log *Logger) shutdown() {
	_ = log.logger.Sync()
	log.writer.stop()
}

// InitEventLogger 初始化全局事件日志记录器
// 事件日志记录器固定使用info级别，用于记录重要的业务事件
// 如果已存在事件日志记录器，会先关闭原有实例再创建新实例
func InitEventLogger(baseDir, serverId string, slicePeriod time.Duration) {
	if EventLogger != nil {
		EventLogger.shutdown()
	}
	EventLogger = NewLogger(baseDir, LogTypeEvent, "info", serverId, slicePeriod)
}

// InitRuntimeLogger 初始化全局运行时日志记录器
// 运行时日志记录器用于记录系统运行时的各种信息，支持不同日志级别
// 如果已存在运行时日志记录器，会先关闭原有实例再创建新实例
func InitRuntimeLogger(baseDir, logLevel, serverId string, slicePeriod time.Duration) {
	if runtimeLogger != nil {
		runtimeLogger.shutdown()
	}
	runtimeLogger = NewLogger(baseDir, LogTypeRuntime, logLevel, serverId, slicePeriod)
}

// LogType 返回日志记录器的日志类型
func (log *Logger) LogType() string {
	return log.logType
}

// BaseDir 返回日志记录器的日志目录
func (log *Logger) BaseDir() string {
	return log.baseDir
}

// withCommonFields 为日志记录器添加公共字段
// 添加的字段包括：
//   - id: 使用utils.GenID()生成的唯一标识
//   - creator: 当前服务器的ID
//
// 返回添加了公共字段的新logger实例
func (log *Logger) withCommonFields() *zap.Logger {
	fs := []zap.Field{
		zap.String("id", utils.GenID()),
		zap.String("creator", log.serverId),
	}
	return log.logger.With(fs...)
}

// Log 获取带有公共字段的日志记录器实例
// 每次调用都会生成新的日志ID，确保日志的唯一性
// 返回标准的zap.Logger实例，支持结构化日志记录
func (log *Logger) Log() *zap.Logger {
	return log.withCommonFields()
}

// SugarLog 获取带有公共字段的语法糖风格日志记录器
// Sugar风格提供了更简单的API，支持printf风格的格式化
// 适用于需要简单快速记录日志的场景
func (log *Logger) SugarLog() *zap.SugaredLogger {
	return log.withCommonFields().Sugar()
}

// Log 返回全局运行时日志记录器并添加公共字段
func Log() *zap.Logger {
	if runtimeLogger == nil {
		panic("Runtime Logger is not initialized")
	}
	return runtimeLogger.withCommonFields()
}

// SugarLog 返回全局运行时日志记录器的 sugar 风格实例并添加公共字段
func SugarLog() *zap.SugaredLogger {
	if runtimeLogger == nil {
		panic("Runtime Logger is not initialized")
	}
	return runtimeLogger.withCommonFields().Sugar()
}

// Zap 返回原始的 zap.Logger 实例，不添加公共字段
func (log *Logger) Zap() *zap.Logger {
	return log.logger
}

// IsDebugging 检查当前是否处于调试模式
// 当运行时日志级别设置为Debug时返回true
// 如果运行时日志记录器未初始化，则返回false
func IsDebugging() bool {
	if runtimeLogger == nil {
		return false
	}
	return runtimeLogger.logLevel <= zap.DebugLevel
}

// log 内部日志打印函数
// 根据当前日志级别决定是否打印日志
// 支持添加带颜色的前缀，提高日志可读性
// 参数：
//   - level: 要打印的日志级别
//   - prefix: 日志前缀（支持ANSI颜色代码）
//   - a: 要打印的参数列表
func log(level zapcore.Level, prefix string, a ...any) {
	l := level
	if runtimeLogger != nil {
		l = runtimeLogger.logLevel
	}
	if l <= level {
		a = append([]any{prefix}, a...)
		fmt.Println(a...)
	}
}

// logf 内部格式化日志打印函数
// 根据当前日志级别决定是否打印日志
// 支持printf风格的格式化和带颜色的前缀
// 参数：
//   - level: 要打印的日志级别
//   - prefix: 日志前缀（支持ANSI颜色代码）
//   - format: printf风格的格式化字符串
//   - a: 格式化参数列表
func logf(level zapcore.Level, prefix, format string, a ...any) {
	l := level
	if runtimeLogger != nil {
		l = runtimeLogger.logLevel
	}
	if l <= level {
		fmt.Printf(prefix+" "+format+"\n", a...)
	}
}

// Debug 打印调试级别的日志
func Debug(a ...any) {
	log(zap.DebugLevel, "\x1b[32m[DEBUG]\x1b[0m", a...)
}

// Debugf 打印格式化的调试级别的日志
func Debugf(format string, a ...any) {
	logf(zap.DebugLevel, "\x1b[32m[DEBUG]\x1b[0m", format, a...)
}

// Info 打印信息级别的日志
func Info(a ...any) {
	log(zap.InfoLevel, "\x1b[34m[INFO]\x1b[0m", a...)
}

// Infof 打印格式化的信息级别的日志
func Infof(format string, a ...any) {
	logf(zap.InfoLevel, "\x1b[34m[INFO]\x1b[0m", format, a...)
}

// Warn 打印警告级别的日志
func Warn(a ...any) {
	log(zap.WarnLevel, "\x1b[33m[WARN]\x1b[0m", a...)
}

// Warnf 打印格式化的警告级别的日志
func Warnf(format string, a ...any) {
	logf(zap.WarnLevel, "\x1b[33m[WARN]\x1b[0m", format, a...)
}

// Error 打印错误级别的日志
func Error(a ...any) {
	log(zap.ErrorLevel, "\x1b[31m[ERROR]\x1b[0m", a...)
}

// Errorf 打印格式化的错误级别的日志
func Errorf(format string, a ...any) {
	logf(zap.ErrorLevel, "\x1b[31m[ERROR]\x1b[0m", format, a...)
}

// Fatal 打印致命级别的日志
func Fatal(a ...any) {
	log(zap.FatalLevel, "\x1b[31m[FATAL]\x1b[0m", a...)
}

// Fatalf 打印格式化的致命级别的日志
func Fatalf(format string, a ...any) {
	logf(zap.FatalLevel, "\x1b[31m[FATAL]\x1b[0m", format, a...)
}
