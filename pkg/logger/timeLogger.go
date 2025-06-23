package logger

import (
	"fmt"
	"time"

	"github.com/pancake-lee/pgo/pkg/util"
)

// 统计耗时的日志工具
type timeLogger struct {
	name      string
	sTime     time.Time
	keyList   []string
	pointList []time.Time
	prefixCnt map[string]int
}

func NewTimeLogger(name string) *timeLogger {
	if name == "" {
		name = util.GetCallerFuncName(1)
	}

	tLogger := new(timeLogger)
	tLogger.name = name
	tLogger.sTime = time.Now()
	return tLogger
}

func (tLogger *timeLogger) AddPoint(key string) {
	tLogger.keyList = append(tLogger.keyList, key)
	tLogger.pointList = append(tLogger.pointList, time.Now())
}

// 自动追加一个计时点，按0,1,2...作为key，类似i++
func (tLogger *timeLogger) AddPointInc() {
	tLogger.AddPoint(util.IntToStr(len(tLogger.keyList)))
}

// 在for循环嵌套的情况下，拼接成"prefix-i"的形式
func (tLogger *timeLogger) AddPointIncPrefix(prefix string) {
	if tLogger.prefixCnt == nil {
		tLogger.prefixCnt = make(map[string]int)
	}
	tLogger.prefixCnt[prefix]++
	tLogger.AddPoint(prefix + "-" + util.IntToStr(tLogger.prefixCnt[prefix]))
}

// 打印输出最终时间统计信息，并且汇总整体耗时
func (tLogger *timeLogger) Log() {
	tLogger.AddPoint("end")

	logStr := fmt.Sprintf("time log [%v] sum[%dms] point list [key|add|sum]",
		tLogger.name, time.Since(tLogger.sTime).Milliseconds())

	lastTime := &tLogger.sTime
	for i, v := range tLogger.keyList {
		curTime := &tLogger.pointList[i]
		logStr += fmt.Sprintf(" [%s|+%d|%d]", v,
			curTime.Sub(*lastTime).Milliseconds(),
			curTime.Sub(tLogger.sTime).Milliseconds())

		lastTime = curTime
	}
	myLog(zapLogger.Debug, 2, nil, logStr)
}
