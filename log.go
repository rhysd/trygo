package trygo

import (
	"fmt"
	"github.com/fatih/color"
	stdlog "log"
)

var (
	logEnabled bool
	yellowC    = color.New(color.FgYellow)
	redC       = color.New(color.FgRed)
	greenC     = color.New(color.FgGreen)
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

// hi highlights text in log message with yellow color
//
//   log("Hellow", hi("important"), "message")
func hi(v ...interface{}) string {
	if !logEnabled {
		return ""
	}
	return yellowC.Sprint(v...)
}

// ftl is for fatal message. This function should be used only for fatal error information
func ftl(v ...interface{}) string {
	if !logEnabled {
		return ""
	}
	return redC.Sprint(v...)
}

// dbg is for debugging. This function should not be used usually, but used for temporary highlighting
// for debugging.
func dbg(v ...interface{}) string {
	if !logEnabled {
		return ""
	}
	return greenC.Sprint(v...)
}
