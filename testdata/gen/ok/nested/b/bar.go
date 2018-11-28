package b

import (
	"github.com/rhysd/trygo/testdata/gen/ok/nested/a"
	"os"
)

func Bar() (int, error) {
	try(a.Foo())
	try(os.Mkdir("hello", 0755))
	f := try(os.Create("bye"))
	f = try(os.Create("bye"))
	f.Close()
	return 42, nil
}
