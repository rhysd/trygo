package a

import (
	"fmt"
)

func Foo() (int, error) {
	n, _err0 := fmt.Println("hello")
	if _err0 != nil {
		return 0, _err0
	}
	return n + 10, nil
}
