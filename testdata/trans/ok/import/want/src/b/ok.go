package b

import (
	"github.com/rhysd/trygo/testdata/trans/ok/import/want/src"
	"github.com/rhysd/trygo/testdata/trans/ok/import/want/src/a"
)

func Foo() (string, int, error) {
	n, _err0 := a.Foo()
	if _err0 != nil {
		return "", 0, _err0
	}
	s, _err1 := root.Foo()
	if _err1 != nil {
		return "", 0, _err1
	}
	return s, n, nil
}
