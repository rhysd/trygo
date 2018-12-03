package root

import (
	"fmt"
	"github.com/rhysd/trygo/testdata/trans/ok/import/want/src/a"
)

func Foo() (string, error) {
	n, _err0 := a.Foo()
	if _err0 != nil {
		return "", _err0
	}
	i, _err1 := a.Foo()
	if _err1 != nil {
		return "", _err1
	}
	return fmt.Sprint(n + i), nil
}
