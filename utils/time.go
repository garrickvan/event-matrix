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

// Package utils 提供通用工具函数
package utils

/*
time.go 提供统一的时间处理函数，确保系统中所有时间操作一致性。

所有时间函数均基于系统当前时区，提供秒、毫秒、微秒和纳秒级别的精度。
还包含时间格式转换、季度计算等辅助功能。
*/

import (
	"fmt"
	"strings"
	"time"
)

// GetNowSecond 获取当前时间的秒级Unix时间戳
func GetNowSecond() int64 {
	return time.Now().Unix()
}

// GetNowMilli 获取当前时间的毫秒级Unix时间戳
func GetNowMilli() int64 {
	return time.Now().UnixMilli()
}

// GetNowMicro 获取当前时间的微秒级Unix时间戳
func GetNowMicro() int64 {
	return time.Now().UnixMicro()
}

// GetNowNano 获取当前时间的纳秒级Unix时间戳
func GetNowNano() int64 {
	return time.Now().UnixNano()
}

// AddSecondsToCurrentTime 计算当前时间加上指定秒数后的时间戳
//
// N: 要添加的秒数
//
// 返回当前时间加上N秒后的毫秒级时间戳
func AddSecondsToCurrentTime(N int) int64 {
	return time.Now().Add(time.Duration(N) * time.Second).UnixMilli()
}

// GetNowQuarter 获取当前日期所在的季度
//
// 返回当前日期的季度数(1-4)
func GetNowQuarter() int {
	return GetQuarterFromTime(time.Now())
}

// GetQuarterFromTime 获取指定时间所在的季度
//
// t: 要计算季度的时间
//
// 返回指定时间的季度数(1-4)
func GetQuarterFromTime(t time.Time) int {
	month := t.Month()
	switch {
	case month >= 1 && month <= 3:
		return 1
	case month >= 4 && month <= 6:
		return 2
	case month >= 7 && month <= 9:
		return 3
	default:
		return 4
	}
}

// TimeStrToUTCMilli 将时间字符串转换为毫秒级Unix时间戳
//
// 支持的格式:
//
//	"2006-01-02 15:04:05" (带秒)
//	"2006-01-02 15:04" (不带秒)
//
// timestr: 要转换的时间字符串
//
// 返回转换后的毫秒时间戳，转换失败则返回0
func TimeStrToUTCMilli(timestr string) int64 {
	layout := "2006-01-02 15:04:05"
	if strings.Count(timestr, ":") == 2 {
		layout = "2006-01-02 15:04:05"
	} else if strings.Count(timestr, ":") == 1 {
		layout = "2006-01-02 15:04"
	}
	t, err := time.Parse(layout, timestr)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return 0
	}
	return t.UnixMilli()
}

// GetCurrentTimezoneOffset 获取当前系统时区相对于UTC的小时偏移量
//
// 返回时区偏移小时数，东区为正，西区为负
func GetCurrentTimezoneOffset() int {
	now := time.Now()
	_, offset := now.Zone()
	offsetHours := offset / 3600
	return offsetHours
}
