package foo

import (
	"os"
)

func Bar() (int, error) {
	try(Foo())
	try(os.Mkdir("hello", 0755))
	f := try(os.Create("bye"))
	f = try(os.Create("bye"))
	f.Close()
	return 42, nil
}
