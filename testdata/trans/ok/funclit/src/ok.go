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
		n = try(fmt.Println("hello"))
		return n, S{n}, nil
	}
	n, _ := try(x())
	return n, nil
}

func g() error {
	x := func() (string, I, error) {
		n := try(fmt.Println("hello"))
		return "", &n, nil
	}
	try(x())
	return nil
}

func h() (b bool, err error) {
	x := func() (bool, *int, error) {
		n := try(fmt.Println("hello"))
		return true, &n, nil
	}
	b, _ = try(x())
	return
}

func i() (int, error) {
	f1 := func() (int, error) {
		try(fmt.Println("hello"))
		return 0, nil
	}
	f2 := func() (int, error) {
		n := try(fmt.Println("hello"))
		return n, nil
	}
	f3 := func() (int, error) {
		var n = try(fmt.Println("hello"))
		return n, nil
	}
	f4 := func() (int, error) {
		var n int
		n = try(fmt.Println("hello"))
		return n, nil
	}
	try(f1())
	n := try(f2())
	n = try(f3())
	var i = try(f4())
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
					n, p, s, _ := try(f())
					return n, p, s, nil
				}
				var n, p, _ = try(f())
				return n, p, nil
			}
			var n int
			n, _ = try(f())
			return n, nil
		}
		try(f())
		return nil
	}
	try(f())
	return nil
}
