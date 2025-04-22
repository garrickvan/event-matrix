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

package logcenter

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/garrickvan/event-matrix/constant"
	"github.com/garrickvan/event-matrix/core"
	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/garrickvan/event-matrix/worker/intranet/dispatcher"
)

type LogDaemonSubmitter struct {
	logCenterEndpoint string
	interval          time.Duration
	logSliceInterval  time.Duration
	logLocation       string
	stopChan          chan struct{} // 添加 stopChan 通道
}

var (
	submitter        *LogDaemonSubmitter
	LogEndpointEvent = &core.Event{
		Project: core.INTERNAL_PROJECT,
		Version: constant.INITIAL_VERSION,
		Context: LogCenterWorkerContext,
		Entity:  LogCenterWorkerEntity,
	}
)

func init() {
	LogEndpointEvent.GenerateSign()
}

func NewLogDaemonSubmitter(logLocationDir string) *LogDaemonSubmitter {
	if submitter != nil {
		submitter.StopDaemon()
	}
	submitter = &LogDaemonSubmitter{
		logCenterEndpoint: "",
		logLocation:       logLocationDir,
		interval:          5 * time.Second,     // 默认5秒检查一次日志目录
		logSliceInterval:  20 * time.Second,    // 默认20秒日志切片间隔
		stopChan:          make(chan struct{}), // 初始化 stopChan
	}
	return submitter
}

func (ls *LogDaemonSubmitter) ResetInterval(i time.Duration) {
	if i <= 3*time.Second {
		i = 3 * time.Second // 最小间隔为3秒
	}
	ls.interval = i
}

func (ls *LogDaemonSubmitter) ResetLogSliceInterval(i time.Duration) {
	if i <= 5*time.Second {
		i = 5 * time.Second // 最小间隔为10秒
	}
	ls.logSliceInterval = i
}

func (ls *LogDaemonSubmitter) StartDaemon() {
	go func() {
		// 等待系统初始化
		time.Sleep(5 * time.Second)
		for {
			select {
			case <-ls.stopChan:
				logx.Log().Info("日志提交守护进程已停止")
				return
			default:
				defer func() {
					if r := recover(); r != nil {
						if err, ok := r.(error); ok {
							logx.Error(fmt.Sprintf("日志提交守护进程出现异常中断： %v\n%s", err, debug.Stack()))
						}
						time.Sleep(ls.interval)
						ls.StartDaemon()
					}
				}()
				ls.submitLog()
				time.Sleep(ls.interval)
			}
		}
	}()
}

func (ls *LogDaemonSubmitter) StopDaemon() {
	close(ls.stopChan) // 关闭 stopChan 通道，通知守护进程停止
}

func (ls *LogDaemonSubmitter) submitLog() {
	if ls.logCenterEndpoint == "" {
		if LogEndpointEvent == nil {
			return
		}

		endpoint := dispatcher.GetWorkerEndpoint(LogEndpointEvent)
		if endpoint == "" {
			return
		} else {
			ls.logCenterEndpoint = endpoint
			logx.Log().Debug("日志提交网关地址获取成功")
		}
	}
	files, err := os.ReadDir(ls.logLocation)
	if err != nil {
		logx.Log().Error("日志目录:" + ls.logLocation + " 读取失败: " + err.Error())
		return
	}
	// 遍历文件
	for _, file := range files {
		if ls.logCenterEndpoint == "" {
			return
		}
		if !isValidLogFileName(file.Name(),
			logx.LogTypeEvent,
			logx.LogTypeRuntime,
			logx.LogSuffix) {
			// logx.Debug("日志文件:" + file.Name() + " 不符合日志文件名格式，忽略提交")
			continue
		}
		if !ls.isNeedToSubmit(file) {
			// logx.Debug("日志文件:" + file.Name() + " 不符合日志文件名格式，忽略提交")
			continue
		}
		if strings.HasPrefix(file.Name(), logx.LogTypeRuntime) {
			ls.parsingAndSubmitLog(file, logx.LogTypeRuntime)
		}
		if strings.HasPrefix(file.Name(), logx.LogTypeEvent) {
			ls.parsingAndSubmitLog(file, logx.LogTypeEvent)
		}
		time.Sleep(500 * time.Millisecond) // 防止日志积压，导致一次性提交过于频繁
	}
}

// 判断文件名是否符合特定格式
func isValidLogFileName(fileName, prefix1, prefix2, suffix string) bool {
	// 动态生成正则表达式，用于匹配前缀1或前缀2开头，中间是任意字符（除换行符外，.除外，可根据实际情况调整），最后是后缀
	// 使用 [^.] 表示匹配除. 之外的任意字符，.* 表示匹配零个或多个前面的表达式，也就是中间可以是任意内容（除换行符外，放宽限制后的效果）
	pattern := fmt.Sprintf(`^(%s|%s)\..*?\.%s$`, regexp.QuoteMeta(prefix1), regexp.QuoteMeta(prefix2), regexp.QuoteMeta(suffix))
	re := regexp.MustCompile(pattern)
	return re.MatchString(fileName)
}

// 检查分钟日志切片的修改时间是否在logSliceInterval以上
func (ls *LogDaemonSubmitter) isNeedToSubmit(file os.DirEntry) bool {
	fileInfo, _ := file.Info()
	if fileInfo != nil {
		if fileInfo.ModTime().Add(ls.logSliceInterval + 1*time.Second).Before(time.Now()) {
			return true
		}
	}
	return false
}

func (ls *LogDaemonSubmitter) parsingAndSubmitLog(file os.DirEntry, logType string) {
	filePath := filepath.Join(ls.logLocation, file.Name())
	f, err := os.Open(filePath)
	if err != nil {
		logx.Log().Error("日志文件:" + filePath + " 打开失败: " + err.Error())
		return
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	logs := make([]logx.LogEntry, 0)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			logx.Log().Error("日志文件:" + filePath + " 读取失败: " + err.Error())
			return
		}
		entry := logx.LogEntry{}
		err = jsonx.UnmarshalFromStr(line, &entry)
		if err != nil {
			logx.Log().Error("日志文件:" + filePath + " 解析失败: " + err.Error())
			return
		}
		logs = append(logs, entry)
	}
	logTypeInt := GW_T_W_RUNTIME_LOG_SUBMIT
	if logType == logx.LogTypeEvent {
		logTypeInt = GW_T_W_EVENT_LOG_SUBMIT
	}
	// 分批提交日志记录
	for i := 0; i < len(logs); i += batchSize {
		end := i + batchSize
		if end > len(logs) {
			end = len(logs)
		}
		batch := logs[i:end]
		batchStr, _ := jsonx.MarshalToStr(batch)
		resp, err := dispatcher.Event(ls.logCenterEndpoint, logTypeInt, batchStr, nil)
		if err != nil {
			ls.logCenterEndpoint = "" // 提交失败，重置日志中心地址，重新获取
			logx.Log().Error("日志文件:" + filePath + " 提交失败: " + err.Error())
			return
		}
		if resp.Status() != http.StatusOK {
			ls.logCenterEndpoint = "" // 提交失败，重置日志中心地址，重新获取
			logx.Log().Error("日志文件:" + filePath + " 提交失败: " + resp.TemporaryData())
			return
		}
	}
	// 当该文件中所有日志被处理完以后，才会删除原日志切片，防止日志记录丢失
	err = os.Remove(filePath)
	if err != nil {
		logx.Log().Error("日志文件:" + filePath + " 删除失败: " + err.Error())
	}
}
