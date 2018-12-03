package main

import (
	"fmt"
)

type S struct {
	i int
}

type I interface{}

func f() (int, error) {
	try(f1())
	return 0, nil
}

func f1() (int, S, I, *int, chan int, error) {
	try(fmt.Println("hello"))
	var n int
	return n, S{i: n}, n, &n, nil, nil
}

func g() (s string, err error) {
	s = "hello"
	try(fmt.Println(s))
	try(fmt.Println(s))
	try(fmt.Println(s))
	try(fmt.Println(s))
	try(fmt.Println(s))
	try(fmt.Println(s))
	return
}
