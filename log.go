package trygo

import (
	"fmt"
	"github.com/fatih/color"
	stdlog "log"
	"path/filepath"
	"strings"
)

var (
	logEnabled bool
	yellow     = color.New(color.FgYellow)
	red        = color.New(color.FgRed)
	green      = color.New(color.FgGreen)
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
		stdlog.Output(2, fmt.Sprintln(v...))
	}
}

func logf(format string, v ...interface{}) {
	if logEnabled {
		stdlog.Output(2, fmt.Sprintf(format+"\n", v...))
	}
}

// hi highlights text in log message with yellow color
//
//   log("Hellow", hi("important"), "message")
func hi(v ...interface{}) string {
	if !logEnabled {
		return ""
	}
	return yellow.Sprint(v...)
}

// ftl is for fatal message. This function should be used only for fatal error information
func ftl(v ...interface{}) string {
	if !logEnabled {
		return ""
	}
	return red.Sprint(v...)
}

// dbg is for debugging. This function should not be used usually, but used for temporary highlighting
// for debugging.
func dbg(v ...interface{}) string {
	if !logEnabled {
		return ""
	}
	return green.Sprint(v...)
}

func relpath(abspath string) string {
	if !logEnabled {
		return abspath
	}
	if !filepath.IsAbs(abspath) {
		return abspath
	}
	rel := strings.TrimPrefix(abspath, cwd)
	if len(abspath) <= len(rel) {
		return abspath
	}
	return "." + rel
}
