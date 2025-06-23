package logger

import (
	"fmt"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"runtime"
	"strings"
	"sync"
)

var logger *logrus.Logger
var once sync.Once
var stdHook *stdoutHook

func init() {
	once.Do(func() {
		logger = logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		logger.Formatter = &logrus.JSONFormatter{}

		logger.Out = ioutil.Discard
		stdHook = NewStdoutHook()
		logger.AddHook(stdHook)
	})
}

func SetLogFile(logPath string)  {
	if logPath != "" {
		logger.AddHook(lfshook.NewHook(
			logPath,
			&logrus.JSONFormatter{CallerPrettyfier: CallerPretty},
		))
	}
}

func SwitchLevelShow(b bool)  {
	stdHook.showLevel = b
}

// 错误信息
func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
	logger.Exit(1)
}

// 警告信息
func Warningf(format string, args ...interface{}) {
	logger.Warningf(format, args...)
}

// 成功的提示信息
func Successf(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

// 普通提示信息
func Infof(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

// 将日志中记录的文件名 file 和方法名 func 转成短名字
func CallerPretty(caller *runtime.Frame) (function string, file string) {
	if caller == nil {
		return "", ""
	}

	short := caller.File
	i := strings.LastIndex(caller.File, "/")
	if i != -1 && i != len(caller.File)-1 {
		short = caller.File[i+1:]
	}

	fun := caller.Function
	j := strings.LastIndex(caller.Function, "/")
	if j != -1 && j != len(caller.Function)-1 {
		fun = caller.Function[j+1:]
	}

	return fun, fmt.Sprintf("%s:%d", short, caller.Line)
}
