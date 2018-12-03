package main

import (
	"fmt"
)

type S struct {
	i int
}

type I interface{}

func f() (int, error) {
	n, s, i, p, c := try(f1())
	fmt.Println(n, s, i, p, c)
	return n, nil
}

func f1() (int, S, I, *int, chan int, error) {
	n := try(fmt.Println("hello"))
	return n, S{i: n}, n, &n, nil, nil
}

func g() (n int, err error) {
	i1 := try(fmt.Println("hello"))
	i2 := try(fmt.Println("hello"))
	i3 := try(fmt.Println("hello"))
	i4 := try(fmt.Println("hello"))
	i5 := try(fmt.Println("hello"))
	i6 := try(fmt.Println("hello"))
	n = i1 + i2 + i3 + i4 + i5 + i6
	return
}
