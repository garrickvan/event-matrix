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

package loadtool

import (
	"fmt"
	"sync"
	"time"

	"github.com/garrickvan/event-matrix/utils/logx"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

const (
	COLLECT_FREQ   = 5 * time.Second // 收集频率
	MAX_DISK_USAGE = 95              // 磁盘使用率达到 95% 时，认为系统负载较高
)

var (
	_calculator *LoadCalculator = nil
	once        sync.Once
)

// systemMetrics 用来存储系统的CPU、内存和磁盘占用率
type systemMetrics struct {
	CPUUsage    float64
	MemoryUsage float64
	DiskUsage   float64
}

// LoadCalculator 负责每分钟计算系统负载率
type LoadCalculator struct {
	Metrics    []systemMetrics
	SampleSize int
}

// NewLoadCalculator 创建 LoadCalculator 实例
func NewLoadCalculator(sampleSize int) *LoadCalculator {
	return &LoadCalculator{
		Metrics:    []systemMetrics{},
		SampleSize: sampleSize,
	}
}

// collectMetrics 收集系统当前的 CPU、内存和磁盘使用率
func (lc *LoadCalculator) collectMetrics() {
	// 获取 CPU 使用率
	cpuUsages, err := cpu.Percent(COLLECT_FREQ, false)
	if err != nil {
		logx.Log().Error("Error collecting CPU usage: " + err.Error() + "\n")
		return
	}
	cpuUsage := cpuUsages[0]

	// 获取内存使用率
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		logx.Log().Error("Error collecting memory usage: " + err.Error() + "\n")
		return
	}
	memoryUsage := memInfo.UsedPercent

	// 获取磁盘使用率
	diskInfo, err := disk.Usage("/")
	if err != nil {
		logx.Log().Error("Error collecting disk usage: " + err.Error() + "\n")
		return
	}
	diskUsage := diskInfo.UsedPercent

	// 创建系统指标数据
	metrics := systemMetrics{
		CPUUsage:    cpuUsage,
		MemoryUsage: memoryUsage,
		DiskUsage:   diskUsage,
	}

	if lc.SampleSize > 1 {
		// 限制 Metrics 长度, 保持最近的 lc.SampleSize 个数据
		if len(lc.Metrics) >= lc.SampleSize {
			lc.Metrics = lc.Metrics[1:]
		}
	} else if lc.SampleSize == 1 {
		lc.Metrics = []systemMetrics{}
	} else {
		return
	}
	lc.Metrics = append(lc.Metrics, metrics)
}

// calculateLoadRate 计算负载率
func (lc *LoadCalculator) CalculateLoadRate() float64 {
	var totalCPU, totalMemory float64

	if lc == nil || len(lc.Metrics) == 0 {
		return 0
	}

	count := float64(len(lc.Metrics))
	for _, metric := range lc.Metrics {
		totalCPU += metric.CPUUsage
		totalMemory += metric.MemoryUsage
	}

	// 如果 CPU、内存或磁盘使用率任一达到 98%，负载率达到最大值
	if totalCPU/count >= 98 || totalMemory/count >= 98 {
		return 100
	}

	// 检查磁盘使用率
	if lc != nil && len(lc.Metrics) > 0 {
		if lc.Metrics[len(lc.Metrics)-1].DiskUsage >= MAX_DISK_USAGE {
			return 100
		}
	}

	// 计算综合负载率（根据权重调整）
	averageLoad := ((totalCPU + totalMemory) / 2) / count
	// fmt.Printf("CPU: %.2f%%, Memory: %.2f%%, Disk: %.2f%%\n", totalCPU/count, totalMemory/count, lc.Metrics[len(lc.Metrics)-1].DiskUsage)
	return averageLoad
}

// Start 负责收集数据并计算平均负载率
func (lc *LoadCalculator) Start() {
	go func() {
		for {
			lc.collectMetrics()
			time.Sleep(COLLECT_FREQ)
		}
	}()
}

func main() {
	lc := NewLoadCalculator(6) // 设置采样大小为 6
	go lc.Start()
	for {
		// 每分钟计算一次负载率
		if len(lc.Metrics) == lc.SampleSize {
			loadRate := lc.CalculateLoadRate()
			fmt.Printf("系统负载率: %.2f%%\n", loadRate)
		}
		time.Sleep(COLLECT_FREQ)
	}
}

// sampleSize 采样大小，即最近多少分钟的指标数据用于计算负载率
func Init(sampleSize int) {
	once.Do(func() {
		_calculator = NewLoadCalculator(sampleSize)
		_calculator.Start()
	})
}

func GetLoadRate() float64 {
	return _calculator.CalculateLoadRate()
}
