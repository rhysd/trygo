package root

import (
	"fmt"
	"github.com/rhysd/trygo/testdata/trans/ok/import/src/a"
)

func Foo() (string, error) {
	n := try(a.Foo())
	i := try(a.Foo())
	return fmt.Sprint(n + i), nil
}
