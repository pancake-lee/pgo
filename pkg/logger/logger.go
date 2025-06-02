package logger

import (
	"log"
	"pgo/pkg/util"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func init() {
	// log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.SetFlags(log.Ltime | log.Lshortfile)
}

// 该文件包含的主要是实际业务代码中打印日志使用的方法封装

// -----------------------------------------------------------------------
// svr 表示当前代码作为grpc服务端，接受到req请求，返回resp响应

func SvrReqLog(rpcName, reqId string, msg protoreflect.ProtoMessage) {
	if apiLogger != nil {
		myLogf(apiLogger.Debugf, 2, nil,
			"[%s]svr req [%s]: %v", reqId, rpcName, msg)
	}
	myLogf(errorLogger.Debugf, 2, nil,
		"[%s]svr req [%s]", reqId, rpcName)
}
func SvrRespLog(rpcName, reqId string, errNo int32, msg protoreflect.ProtoMessage) {
	if apiLogger != nil {
		myLogf(apiLogger.Debugf, 2, nil,
			"[%s]svr resp[%s][%d]: %v", reqId, rpcName, errNo, msg)
	}
	myLogf(errorLogger.Debugf, 2, nil,
		"[%s]svr resp[%s][%d]", reqId, rpcName, errNo)
}

// -----------------------------------------------------------------------
// cli 表示当前代码作为grpc客户端，send请求，recv响应

func ReqLogf(rpcName, reqId string, msg protoreflect.ProtoMessage) {
	if apiLogger != nil {
		myLogf(apiLogger.Debugf, 2, nil, "[%s]cli send[%s]: %v", reqId, rpcName, msg)
	}
	myLogf(errorLogger.Debugf, 2, nil, "[%s]cli send[%s]", reqId, rpcName)
}
func RespLogf(rpcName, reqId string, errNo int32, msg protoreflect.ProtoMessage) {
	if apiLogger != nil {
		myLogf(apiLogger.Debugf, 2, nil, "[%s]cli recv[%s][%d]: %v", reqId, rpcName, errNo, msg)
	}
	myLogf(errorLogger.Debugf, 2, nil, "[%s]cli recv[%s][%d]", reqId, rpcName, errNo)
}

// -----------------------------------------------------------------------
func ReqStrLogf(rpcName, reqId string, msg *string) {
	if apiLogger != nil {
		myLogf(apiLogger.Debugf, 2, nil, "[%s]cli send[%s]: %s", reqId, rpcName, *msg)
	}
	myLogf(errorLogger.Debugf, 2, nil, "[%s]cli send[%s]", reqId, rpcName)
}
func RespStrLogf(rpcName, reqId string, errNo int32, msg *string) {
	if apiLogger != nil {
		myLogf(apiLogger.Debugf, 2, nil, "[%s]cli recv[%s][%d]: %s", reqId, rpcName, errNo, *msg)
	}
	myLogf(errorLogger.Debugf, 2, nil, "[%s]cli recv[%s][%d]", reqId, rpcName, errNo)
}

// -----------------------------------------------------------------------
func Debug(args ...interface{}) {
	myLog(errorLogger.Debug, 2, nil, args...)
}

func Debugf(template string, args ...interface{}) {
	myLogf(errorLogger.Debugf, 2, nil, template, args...)
}

// -----------------------------------------------------------------------
func Info(args ...interface{}) {
	myLog(errorLogger.Info, 2, nil, args...)
}

func Infof(template string, args ...interface{}) {
	myLogf(errorLogger.Infof, 2, nil, template, args...)
}

// -----------------------------------------------------------------------
func Warn(args ...interface{}) {
	myLog(errorLogger.Warn, 2, nil, args...)
}

func Warnf(template string, args ...interface{}) {
	myLogf(errorLogger.Warnf, 2, nil, template, args...)
}

// -----------------------------------------------------------------------
func Error(args ...interface{}) {
	myLog(errorLogger.Error, 2, nil, args...)
}

func Errorf(template string, args ...interface{}) {
	myLogf(errorLogger.Errorf, 2, nil, template, args...)
}

// -----------------------------------------------------------------------
// 可以指定打印出调用者的信息，0表示打印当前函数位置，1表示上一级调用位置
func Log(lv zapcore.Level, callerLevel int, prefix []interface{},
	args ...interface{}) {

	switch lv {
	case zapcore.InfoLevel:
		myLog(errorLogger.Info, callerLevel+2, prefix, args...)

	case zapcore.WarnLevel:
		myLog(errorLogger.Warn, callerLevel+2, prefix, args...)

	case zapcore.ErrorLevel:
		myLog(errorLogger.Error, callerLevel+2, prefix, args...)

	default:
		myLog(errorLogger.Debug, callerLevel+2, prefix, args...)
	}
}

// 可以指定打印出调用者的信息，0表示打印当前函数位置，1表示上一级调用位置
func Logf(lv zapcore.Level, callerLevel int, prefix []interface{},
	template string, args ...interface{}) {

	switch lv {
	case zapcore.InfoLevel:
		myLogf(errorLogger.Infof, callerLevel+2, prefix, template, args...)

	case zapcore.WarnLevel:
		myLogf(errorLogger.Warnf, callerLevel+2, prefix, template, args...)

	case zapcore.ErrorLevel:
		myLogf(errorLogger.Errorf, callerLevel+2, prefix, template, args...)

	default:
		myLogf(errorLogger.Debugf, callerLevel+2, prefix, template, args...)
	}
}

// -----------------------------------------------------------------------
func LogErrCode(errNo int32) int32 {
	if errNo != 0 {
		myLogf(errorLogger.Warnf, 2, nil, "return errNo[%v]", errNo)
	}
	return errNo
}

func LogErr(err error, errNo int32) int32 {
	myLogf(errorLogger.Warnf, 2, nil, "get err[%s], return errNo[%v]", err.Error(), errNo)
	return errNo
}

// -----------------------------------------------------------------------
// 统计耗时的日志工具
type timeCostLogger struct {
	name      string
	sTime     time.Time
	keyList   []string
	pointList []time.Time
}

func NewTimeCostLogger(name string) *timeCostLogger {
	if name == "" {
		name = util.GetCallerFuncName(1)
	}

	tLogger := new(timeCostLogger)
	tLogger.name = name
	tLogger.sTime = time.Now()
	return tLogger
}

// 自动追加一个计时点，按0,1,2...作为key
func (tLogger *timeCostLogger) AddPoint1() {
	tLogger.addPoint(util.IntToStr(len(tLogger.keyList)))
}

// 在for循环嵌套的情况下，拼接成"level-num"的形式，level肉眼判断是第几层循环体
func (tLogger *timeCostLogger) AddPoint2(prefix string) {
	tLogger.addPoint(prefix + "-" + util.IntToStr(len(tLogger.keyList)))
}

func (tLogger *timeCostLogger) AddPoint(key string) {
	tLogger.addPoint(key)
}

// 打印输出最终时间统计信息，并且汇总整体耗时
func (tLogger *timeCostLogger) Log() {
	tLogger.addPoint("end")

	logStr := "time cost [" + tLogger.name + "] " +
		"sum[" + util.Int64ToStr(time.Since(tLogger.sTime).Milliseconds()) + "ms] " +
		"point list["
	//手拼一个json方便格式化查日志
	lastTime := &tLogger.sTime
	for i, v := range tLogger.keyList {
		curTime := &tLogger.pointList[i]
		//"距离开始时间多少毫秒|距离上一个时间点多少毫秒|计时点key"
		logStr += "\"" +
			util.Int64ToStr(curTime.Sub(tLogger.sTime).Milliseconds()) + "ms|" +
			util.Int64ToStr(curTime.Sub(*lastTime).Milliseconds()) + "ms|" +
			v + "\","
		lastTime = curTime
	}
	if len(tLogger.keyList) != 0 {
		logStr = strings.TrimSuffix(logStr, ",")
	}

	logStr += "]"
	myLog(errorLogger.Debug, 2, nil, logStr)
}

func (tLogger *timeCostLogger) addPoint(key string) {
	tLogger.keyList = append(tLogger.keyList, key)
	tLogger.pointList = append(tLogger.pointList, time.Now())
}
