package b

import (
	"fmt"
)

func Bar() (string, error) {
	if _, err := Foo(); err != nil {
		return "", err
	}
	n, _err0 := Foo()
	if _err0 != nil {
		return "", _err0
	}
	var _err1 error
	n, _err1 = Foo()
	if _err1 != nil {
		return "", _err1
	}
	return fmt.Sprint(n), nil
}
