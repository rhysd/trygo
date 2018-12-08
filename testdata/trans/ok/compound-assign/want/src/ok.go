package foo

import (
	"fmt"
)

func f() error {
	n, _err0 := fmt.Println("hello")
	if _err0 != nil {
		return _err0
	}
	_0, _err1 := fmt.Println("hello")
	if _err1 != nil {
		return _err1
	}
	n += _0
	_1, _err2 := fmt.Println("hello")
	if _err2 != nil {
		return _err2
	}
	n -= _1
	_2, _err3 := fmt.Println("hello")
	if _err3 != nil {
		return _err3
	}
	n *= _2
	_3, _err4 := fmt.Println("hello")
	if _err4 != nil {
		return _err4
	}
	n /= _3
	_4, _err5 := fmt.Println("hello")
	if _err5 != nil {
		return _err5
	}
	n %= _4
	_5, _err6 := fmt.Println("hello")
	if _err6 != nil {
		return _err6
	}
	n &= _5
	return nil
}

func g() (int, error) {
	n, _err0 := fmt.Println("hello")
	if _err0 != nil {
		return 0, _err0
	}
	_0, _err1 := fmt.Println("hello")
	if _err1 != nil {
		return 0, _err1
	}
	n += _0
	_1, _err2 := fmt.Println("hello")
	if _err2 != nil {
		return 0, _err2
	}
	n -= _1
	_2, _err3 := fmt.Println("hello")
	if _err3 != nil {
		return 0, _err3
	}
	n *= _2
	_3, _err4 := fmt.Println("hello")
	if _err4 != nil {
		return 0, _err4
	}
	n /= _3
	_4, _err5 := fmt.Println("hello")
	if _err5 != nil {
		return 0, _err5
	}
	n %= _4
	_5, _err6 := fmt.Println("hello")
	if _err6 != nil {
		return 0, _err6
	}
	n &= _5
	return n, nil
}
