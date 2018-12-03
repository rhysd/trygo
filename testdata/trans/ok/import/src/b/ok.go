package b

import (
	"github.com/rhysd/trygo/testdata/trans/ok/import/src"
	"github.com/rhysd/trygo/testdata/trans/ok/import/src/a"
)

func Foo() (string, int, error) {
	n := try(a.Foo())
	s := try(root.Foo())
	return s, n, nil
}
