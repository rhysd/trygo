package b

import (
	"github.com/rhysd/trygo/testdata/gen/ok/nested/a"
	"os"
)

func Bar() (int, error) {
	if err := a.Foo(); err != nil {
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
