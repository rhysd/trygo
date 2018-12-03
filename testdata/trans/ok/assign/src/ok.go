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
	n, s, i, p, c = try(f1())
	fmt.Println(n, s, i, p, c)
	return n, nil
}

func f1() (int, S, I, *int, chan int, error) {
	var n int
	n = try(fmt.Println("hello"))
	return n, S{i: n}, n, &n, nil, nil
}

func g() (n int, err error) {
	var i int
	i = try(fmt.Println("hello"))
	i = try(fmt.Println("hello"))
	i = try(fmt.Println("hello"))
	i = try(fmt.Println("hello"))
	i = try(fmt.Println("hello"))
	i = try(fmt.Println("hello"))
	n = i
	return
}
