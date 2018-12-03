package foo

import (
	"fmt"
)

func f() (int, error) {
	n, _err0 := fmt.Println("hello")
	if _err0 != nil {
		return 0, _err0
	}
	if x := n + 10; x != 11 {
		return x, nil
	}

	var _err1 error
	n, _err1 = fmt.Println("bye")
	if _err1 != nil {
		return 0, _err1
	}
	return n, nil
}
