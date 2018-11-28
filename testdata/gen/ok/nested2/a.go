package a

import (
	"os"
)

func Foo() (int, error) {
	try(os.Mkdir("hello", 0755))
	f := try(os.Create("bye"))
	f = try(os.Create("bye"))
	f.Close()
	return 0, nil
}
