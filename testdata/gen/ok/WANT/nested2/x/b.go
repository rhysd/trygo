package b

import (
	a "github.com/rhysd/trygo/testdata/gen/ok/HAVE/nested2"
	"os"
)

func Bar() (int, error) {
	if _, err := a.Foo(); err != nil {
		return 0, err
	}
	if err := os.Mkdir("hello", 0755); err != nil {
		return 0, err
	}
	f, _err0 := os.Create("bye")
	if _err0 != nil {
		return 0, _err0
	}
	var _err1 error
	f, _err1 = os.Create("bye")
	if _err1 != nil {
		return 0, _err1
	}
	f.Close()
	return 42, nil
}
