package trygo

import (
	"bytes"
	stdlog "log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogInitLog(t *testing.T) {
	saved := logEnabled
	defer func() {
		logEnabled = saved
	}()

	InitLog(false)
	if logEnabled {
		t.Fatal("log should be disabled")
	}
	InitLog(true)
	if !logEnabled {
		t.Fatal("log should be enabled")
	}
}

func TestLogLogOutput(t *testing.T) {
	saved := logEnabled
	defer func() {
		logEnabled = saved
		stdlog.SetOutput(os.Stderr)
	}()
	InitLog(true)

	var buf bytes.Buffer
	stdlog.SetOutput(&buf)

	log("hello", hi("yellow"), ftl("red!"))
	logf("Answer: %d", 42)

	stderr := buf.String()

	if !strings.Contains(stderr, "hello") {
		t.Fatal("normal log", stderr)
	}
	if !strings.Contains(stderr, "yellow") {
		t.Fatal("highlight", stderr)
	}
	if !strings.Contains(stderr, "red!") {
		t.Fatal("fatal", stderr)
	}
	if !strings.Contains(stderr, "Answer: 42") {
		t.Fatal("formatted", stderr)
	}
}

func TestLogDbgOutput(t *testing.T) {
	saved := logEnabled
	defer func() {
		logEnabled = saved
		stdlog.SetOutput(os.Stderr)
	}()
	InitLog(true)

	var buf bytes.Buffer
	stdlog.SetOutput(&buf)

	dbg("hello", "hi!", "goodbye")

	stderr := buf.String()

	if !strings.Contains(stderr, "hello hi! goodbye") {
		t.Fatal("Debug log is unexpected:", stderr)
	}
}

func TestLogRelpath(t *testing.T) {
	saved := logEnabled
	defer func() {
		logEnabled = saved
	}()
	logEnabled = false

	have := relpath("foo/bar")
	want := "foo/bar"
	if have != want {
		t.Fatal("It should return the same value when log is disabled")
	}

	logEnabled = true
	for p, want := range map[string]string{
		filepath.Join(cwd, "foo/bar"): "./foo/bar",
		"foo/bar":                     "foo/bar",
		"/foo/bar":                    "/foo/bar",
		cwd:                           ".",
	} {
		p = filepath.FromSlash(p)
		want = filepath.FromSlash(want)
		have := relpath(p)
		if want != have {
			t.Fatal(p, "wanted", want, "but have", have)
		}
	}
}

func TestLogNoOutputOnDisabled(t *testing.T) {
	saved := logEnabled
	defer func() {
		logEnabled = saved
		stdlog.SetOutput(os.Stderr)
	}()
	InitLog(false)

	var buf bytes.Buffer
	stdlog.SetOutput(&buf)

	log("hello", hi("world"), ftl("goodbye"))
	logf("Answer is %d", 42)
	dbg("This is", "debug", "message")

	stderr := buf.String()
	if stderr != "" {
		t.Fatal("Log is output even if logging is disabled:", stderr)
	}
}
