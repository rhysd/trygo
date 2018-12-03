package main

import (
	"fmt"
)

type S struct {
	i int
}

type I interface{}

func f() (int, error) {
	x := func() (int, S, error) {
		var n int
		var _err0 error
		n, _err0 = fmt.Println("hello")
		if _err0 != nil {
			return 0, S{}, _err0
		}
		return n, S{n}, nil
	}
	n, _, _err0 := x()
	if _err0 != nil {
		return 0, _err0
	}
	return n, nil
}

func g() error {
	x := func() (string, I, error) {
		n, _err0 := fmt.Println("hello")
		if _err0 != nil {
			return "", nil, _err0
		}
		return "", &n, nil
	}
	if _, _, err := x(); err != nil {
		return err
	}
	return nil
}

func h() (b bool, err error) {
	x := func() (bool, *int, error) {
		n, _err0 := fmt.Println("hello")
		if _err0 != nil {
			return false, nil, _err0
		}
		return true, &n, nil
	}
	var _err0 error
	b, _, _err0 = x()
	if _err0 != nil {
		return false, _err0
	}
	return
}

func i() (int, error) {
	f1 := func() (int, error) {
		if _, err := fmt.Println("hello"); err != nil {
			return 0, err
		}
		return 0, nil
	}
	f2 := func() (int, error) {
		n, _err0 := fmt.Println("hello")
		if _err0 != nil {
			return 0, _err0
		}
		return n, nil
	}
	f3 := func() (int, error) {
		var n, _err0 = fmt.Println("hello")
		if _err0 != nil {
			return 0, _err0
		}
		return n, nil
	}
	f4 := func() (int, error) {
		var n int
		var _err0 error
		n, _err0 = fmt.Println("hello")
		if _err0 != nil {
			return 0, _err0
		}
		return n, nil
	}
	if _, err := f1(); err != nil {
		return 0, err
	}
	n, _err0 := f2()
	if _err0 != nil {
		return 0, _err0
	}
	var _err1 error
	n, _err1 = f3()
	if _err1 != nil {
		return 0, _err1
	}
	var i, _err2 = f4()
	if _err2 != nil {
		return 0, _err2
	}
	return n + i, nil
}

func j() error {
	f := func() error {
		f := func() (int, error) {
			f := func() (int, *int, error) {
				f := func() (int, *int, S, error) {
					f := func() (int, *int, S, I, error) {
						var n int
						return n, &n, S{n}, &n, nil
					}
					n, p, s, _, _err0 := f()
					if _err0 != nil {
						return 0, nil, S{}, _err0
					}
					return n, p, s, nil
				}
				var n, p, _, _err0 = f()
				if _err0 != nil {
					return 0, nil, _err0
				}
				return n, p, nil
			}
			var n int
			var _err0 error
			n, _, _err0 = f()
			if _err0 != nil {
				return 0, _err0
			}
			return n, nil
		}
		if _, err := f(); err != nil {
			return err
		}
		return nil
	}
	if err := f(); err != nil {
		return err
	}
	return nil
}
