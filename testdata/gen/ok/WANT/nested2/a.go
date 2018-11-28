package a

import (
	"os"
)

func Foo() (int, error) {
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
	return 0, nil
}
