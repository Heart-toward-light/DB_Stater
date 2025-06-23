package logger

import (
	"fmt"
	"os"
	"testing"
)

func TestStdLog(t *testing.T) {
	Infof("这是一条提示信息\n")
	Successf("这是一条成功信息\n")
	Warningf("这是一条警告信息\n")
	//Errorf("这是一条错误信息\n")
}

func TestSwitchLevelShowLog(t *testing.T) {
	Infof("这是一条带 Level 的提示信息\n")
	SwitchLevelShow(false)
	Infof("这是一条不带 Level 的提示信息\n")
}

func TestFileLog(t *testing.T) {
	defer func() {
		err := os.Remove("./dbup.log")
		if err != nil {
			fmt.Println(err)
		}
	}()

	// 设置日志文件位置
	SetLogFile("./dbup.log")

	Infof("这是一条提示信息\n")
	Successf("这是一条成功信息\n")
	Warningf("这是一条警告信息\n")
	//Errorf("这是一条错误信息\n")
}
