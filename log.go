package trygo

import (
	"fmt"
	"github.com/fatih/color"
	stdlog "log"
)

var (
	logEnabled bool
	yellow     = color.New(color.FgYellow)
)

func InitLog(enabled bool) {
	logEnabled = enabled
	if !enabled {
		return
	}
	stdlog.SetFlags(stdlog.Lshortfile)
}

func log(v ...interface{}) {
	if logEnabled {
		stdlog.Output(3, fmt.Sprintln(v...))
	}
}

func hi(v ...interface{}) string {
	if !logEnabled {
		return ""
	}
	return yellow.Sprint(v...)
}
