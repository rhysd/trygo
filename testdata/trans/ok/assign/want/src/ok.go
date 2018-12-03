package main

import (
	"fmt"
)

type S struct {
	i int
}

type I interface{}

func f() (int, error) {
	var n int
	var s S
	var i I
	var p *int
	var c chan int
	var _err0 error
	n, s, i, p, c, _err0 = f1()
	if _err0 != nil {
		return 0, _err0
	}
	fmt.Println(n, s, i, p, c)
	return n, nil
}

func f1() (int, S, I, *int, chan int, error) {
	var n int
	var _err0 error
	n, _err0 = fmt.Println("hello")
	if _err0 != nil {
		return 0, S{}, nil, nil, nil, _err0
	}
	return n, S{i: n}, n, &n, nil, nil
}

func g() (n int, err error) {
	var i int
	var _err0 error
	i, _err0 = fmt.Println("hello")
	if _err0 != nil {
		return 0, _err0
	}
	var _err1 error
	i, _err1 = fmt.Println("hello")
	if _err1 != nil {
		return 0, _err1
	}
	var _err2 error
	i, _err2 = fmt.Println("hello")
	if _err2 != nil {
		return 0, _err2
	}
	var _err3 error
	i, _err3 = fmt.Println("hello")
	if _err3 != nil {
		return 0, _err3
	}
	var _err4 error
	i, _err4 = fmt.Println("hello")
	if _err4 != nil {
		return 0, _err4
	}
	var _err5 error
	i, _err5 = fmt.Println("hello")
	if _err5 != nil {
		return 0, _err5
	}
	n = i
	return
}
