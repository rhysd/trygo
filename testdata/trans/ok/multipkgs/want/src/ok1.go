package root

import (
	"fmt"
)

func Foo() (int, error) {
	x, _err0 := fmt.Println("hello")
	if _err0 != nil {
		return 0, _err0
	}
	var _err1 error
	x, _err1 = fmt.Println("bye")
	if _err1 != nil {
		return 0, _err1
	}
	return x, nil
}
