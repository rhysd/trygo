package main

import (
	"fmt"
)

type S struct {
	i int
}

type I interface{}

func f() (int, error) {
	n, s, i, p, c, _err0 := f1()
	if _err0 != nil {
		return 0, _err0
	}
	fmt.Println(n, s, i, p, c)
	return n, nil
}

func f1() (int, S, I, *int, chan int, error) {
	n, _err0 := fmt.Println("hello")
	if _err0 != nil {
		return 0, S{}, nil, nil, nil, _err0
	}
	return n, S{i: n}, n, &n, nil, nil
}

func g() (n int, err error) {
	i1, _err0 := fmt.Println("hello")
	if _err0 != nil {
		return 0, _err0
	}
	i2, _err1 := fmt.Println("hello")
	if _err1 != nil {
		return 0, _err1
	}
	i3, _err2 := fmt.Println("hello")
	if _err2 != nil {
		return 0, _err2
	}
	i4, _err3 := fmt.Println("hello")
	if _err3 != nil {
		return 0, _err3
	}
	i5, _err4 := fmt.Println("hello")
	if _err4 != nil {
		return 0, _err4
	}
	i6, _err5 := fmt.Println("hello")
	if _err5 != nil {
		return 0, _err5
	}
	n = i1 + i2 + i3 + i4 + i5 + i6
	return
}
