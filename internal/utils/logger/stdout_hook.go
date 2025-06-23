package logger

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

var (
	// ColorErrorMsg is the ansi color formatter for error messages
	ColorErrorMsg = color.New(color.FgHiRed)
	// ColorSuccessMsg is the ansi color formatter for success messages
	ColorSuccessMsg = color.New(color.FgHiGreen)
	// ColorWarningMsg is the ansi color formatter for warning messages
	ColorWarningMsg = color.New(color.FgHiYellow)
	// ColorKeyword is the ansi color formatter for cluster name
	//ColorKeyword = color.New(color.FgHiBlue, color.Bold)
)

var colorMap = map[logrus.Level]*color.Color{
	logrus.InfoLevel:  ColorSuccessMsg,
	logrus.ErrorLevel: ColorErrorMsg,
	logrus.WarnLevel:  ColorWarningMsg,
}

var labelMap = map[logrus.Level]string{
	logrus.DebugLevel: "[INFO]",
	logrus.InfoLevel: "[SUCCESS]",
	logrus.WarnLevel: "[WARNING]",
	logrus.ErrorLevel: "[ERROR]",
}

type stdoutHook struct {
	showLevel bool
}

func NewStdoutHook() *stdoutHook {
	return &stdoutHook{showLevel: true}
}

func (h *stdoutHook) Fire(entry *logrus.Entry) error {
	label,_ := labelMap[entry.Level]

	if c, ok := colorMap[entry.Level]; ok {
		if h.showLevel {
			_, _ = c.Printf("%s%s", label, entry.Message)
		} else {
			if entry.Level == logrus.ErrorLevel {
				_, _ = c.Printf("Error: %s", entry.Message)
			} else {
				_, _ = c.Printf("%s", entry.Message)
			}
		}
	} else {
		if h.showLevel {
			fmt.Printf("%s%s", label, entry.Message)
		} else {
			fmt.Printf("%s", entry.Message)
		}
	}
	return nil
}

func (h *stdoutHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
