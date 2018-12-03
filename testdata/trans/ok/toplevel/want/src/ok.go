package main

import (
	"fmt"
)

type S struct {
	i int
}

type I interface{}

func f() (int, error) {
	if _, _, _, _, _, err := f1(); err != nil {
		return 0, err
	}
	return 0, nil
}

func f1() (int, S, I, *int, chan int, error) {
	if _, err := fmt.Println("hello"); err != nil {
		return 0, S{}, nil, nil, nil, err
	}
	var n int
	return n, S{i: n}, n, &n, nil, nil
}

func g() (s string, err error) {
	s = "hello"
	if _, err := fmt.Println(s); err != nil {
		return "", err
	}
	if _, err := fmt.Println(s); err != nil {
		return "", err
	}
	if _, err := fmt.Println(s); err != nil {
		return "", err
	}
	if _, err := fmt.Println(s); err != nil {
		return "", err
	}
	if _, err := fmt.Println(s); err != nil {
		return "", err
	}
	if _, err := fmt.Println(s); err != nil {
		return "", err
	}
	return
}
